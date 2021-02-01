// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"fmt"
	"time"

	cmodel "github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	pmodel "github.com/prometheus/common/model"
)

func init() {
	hibernate.PersistentFlags().String("server", "http://localhost:8075", "The provisioning server whose API will be queried.")
	hibernate.PersistentFlags().String("thanos-url", "", "The URL to query thanos metrics from.")
	hibernate.PersistentFlags().Bool("dry-run", true, "Whether the autoscaler will perform scaling actions or just print actions that would be taken.")
	hibernate.PersistentFlags().Bool("unlock", false, "Whether the autoscaler will unlock installations to update their size or not.")
	hibernate.PersistentFlags().Int("days", 7, "The number of days back to check if an installation has received new posts since.")

	// Installation filters
	hibernate.PersistentFlags().String("owner", "", "The owner ID value to filter installations by.")
	hibernate.PersistentFlags().String("group", "", "The group ID value to filter installations by.")
}

var hibernate = &cobra.Command{
	Use:   "hibernate",
	Short: "Hibernate installations based on activity metrics",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		logger.Info("Starting installation hibernator")

		serverAddress, _ := command.Flags().GetString("server")
		thanosURL, _ := command.Flags().GetString("thanos-url")
		dryrun, _ := command.Flags().GetBool("dry-run")
		unlock, _ := command.Flags().GetBool("unlock")
		days, _ := command.Flags().GetInt("days")
		owner, _ := command.Flags().GetString("owner")
		group, _ := command.Flags().GetString("group")

		if len(serverAddress) == 0 {
			return errors.New("server value must be defined")
		}
		if len(thanosURL) == 0 {
			return errors.New("thanos-url value must be defined")
		}

		client := cmodel.NewClient(serverAddress)

		logger.Info("Obtaining current installations")
		installations, err := client.GetInstallations(&cmodel.GetInstallationsRequest{
			State:                       cmodel.InstallationStateStable,
			OwnerID:                     owner,
			GroupID:                     group,
			Page:                        0,
			PerPage:                     cmodel.AllPerPage,
			IncludeGroupConfig:          false,
			IncludeGroupConfigOverrides: false,
			IncludeDeleted:              false,
		})
		if err != nil {
			return errors.Wrap(err, "failed to get installations")
		}

		logger.Infof("Calculating hibernate actions on %d stable installations", len(installations))
		var installationsToHibernate []*cmodel.InstallationDTO
		for i, installation := range installations {
			current := i + 1
			if current%10 == 0 {
				logger.Debugf("Processing installation %d of %d", current, len(installations))
			}

			if installation.State != cmodel.InstallationStateStable {
				logger.Warnf("%s - Expected only stable installations (%s); skipping...", installation.ID, installation.State)
				continue
			}
			if installation.APISecurityLock && !unlock {
				logger.Warnf("%s - Installation is locked and hibernator is not set to perform unlocks; skipping...", installation.ID)
				continue
			}

			// A small sleep to help prevent hitting the metrics host too hard.
			// Using the force a bit here. May need to be tweaked.
			time.Sleep(100 * time.Millisecond)

			hibernate, err := determineIfInstallationRequiresHibernation(installation.ID, thanosURL, days)
			if err != nil {
				logger.WithError(err).Warnf("%s - Failed to determine if installation should hibernate", installation.ID)
				continue
			}
			if !hibernate {
				continue
			}

			installationsToHibernate = append(installationsToHibernate, installation)
		}

		if len(installationsToHibernate) == 0 {
			logger.Info("No installations requre hibernation; exiting...")
			return nil
		}

		logger.Infof("Hibernating %d installations", len(installationsToHibernate))
		if dryrun {
			logger.Info("Dry run complete")
		} else {
			for _, installation := range installationsToHibernate {
				logger.Infof("%s - Hibernating installation", installation.ID)

				err = hibernateInstallation(installation, client)
				if err != nil {
					return errors.Wrap(err, "failed to hibernate installation")
				}

				// Another sleep to slow the API calls to the provisioner.
				time.Sleep(500 * time.Millisecond)
			}

			logger.Info("Hibernation check complete")
		}

		return nil
	},
}

func hibernateInstallation(installation *cmodel.InstallationDTO, client *cmodel.Client) error {
	var relock bool
	var err error

	if installation.APISecurityLock {
		err = client.UnlockAPIForInstallation(installation.ID)
		if err != nil {
			return errors.Wrapf(err, "failed to unlock installation %s", installation.ID)
		}
		relock = true
	}

	installation, err = client.HibernateInstallation(installation.ID)
	if err != nil {
		return errors.Wrap(err, "failed to hibernate installation")
	}

	if relock {
		err = client.LockAPIForInstallation(installation.ID)
		if err != nil {
			return errors.Wrapf(err, "failed to relock installation %s", installation.ID)
		}
	}

	return nil
}

func determineIfInstallationRequiresHibernation(installationID, thanosURL string, days int) (bool, error) {
	now := time.Now()
	startTime := now.AddDate(0, 0, -days)

	r := v1.Range{
		Start: startTime,
		End:   now,
		Step:  time.Duration(days) * 24 * time.Hour,
	}
	rawMetrics, err := queryRangeInstallationMetrics(thanosURL, fmt.Sprintf("max(mattermost_post_total{installationId=\"%s\"})", installationID), r)
	if err != nil {
		return false, errors.Wrap(err, "failed to query thanos")
	}
	if len(rawMetrics) == 0 {
		return false, errors.New("no post metrics found for this installation")
	}

	return shouldHibernate(rawMetrics), nil
}

func shouldHibernate(rawMetrics pmodel.Matrix) bool {
	if len(rawMetrics) == 0 {
		return false
	}

	// Loop through each metric. Depending on the prometheus query, there could
	// be multiple results. For instance, we could have a max post count collected
	// from multiple pods.
	for _, rawMetric := range rawMetrics {
		// We expect two values for each metric. The first value will be the most
		// recent and the second will be the value from a previous point in time.
		if len(rawMetric.Values) != 2 {
			return false
		}
		if rawMetric.Values[0].Value != rawMetric.Values[1].Value {
			return false
		}
	}

	return true
}
