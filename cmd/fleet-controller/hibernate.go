// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/mattermost/fleet-controller/internal/metrics"
	cmodel "github.com/mattermost/mattermost-cloud/model"
)

func init() {
	hibernate.PersistentFlags().String("server", "http://localhost:8075", "The provisioning server whose API will be queried.")
	hibernate.PersistentFlags().String("thanos-url", "", "The URL to query thanos metrics from.")
	hibernate.PersistentFlags().Bool("dry-run", true, "Whether the autoscaler will perform scaling actions or just print actions that would be taken.")
	hibernate.PersistentFlags().Bool("unlock", false, "Whether the autoscaler will unlock installations to update their size or not.")
	hibernate.PersistentFlags().Int("days", 7, "The number of days back to check if an installation has received new posts since.")
	hibernate.PersistentFlags().Int64("max-users", 100, "The number of users where the installation won't be hibernated regardless of activity.")

	// Installation filters
	hibernate.PersistentFlags().String("owner", "", "The owner ID value to filter installations by.")
	hibernate.PersistentFlags().String("group", "", "The group ID value to filter installations by.")
}

var hibernate = &cobra.Command{
	Use:   "hibernate",
	Short: "Hibernate installations based on activity metrics",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		logger := logger.WithField("fleet-controller", "hibernate")
		runID := "none"
		productionLogs, _ := command.Flags().GetBool("production-logs")
		if productionLogs {
			logger, runID = setupProductionLogging(logger)
		}

		logger.Info("Starting installation hibernator")

		start := time.Now()

		serverAddress, _ := command.Flags().GetString("server")
		thanosURL, _ := command.Flags().GetString("thanos-url")
		dryrun, _ := command.Flags().GetBool("dry-run")
		unlock, _ := command.Flags().GetBool("unlock")
		days, _ := command.Flags().GetInt("days")
		maxUsers, _ := command.Flags().GetInt64("max-users")
		owner, _ := command.Flags().GetString("owner")
		group, _ := command.Flags().GetString("group")
		webhookURL, _ := command.Flags().GetString("mm-webhook-url")

		if len(serverAddress) == 0 {
			return errors.New("server value must be defined")
		}
		if len(thanosURL) == 0 {
			return errors.New("thanos-url value must be defined")
		}

		client := cmodel.NewClient(serverAddress)

		logger.WithFields(logrus.Fields{
			"owner-filter": owner,
			"group-filter": group,
		}).Info("Obtaining current installations")
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

		tc := metrics.NewThanosClient(thanosURL)

		logger.Info("Gathering installation user metrics")
		userMetrics, err := tc.GetInstallationUserMetrics()
		if err != nil {
			return errors.Wrap(err, "failed to obtain installation metrics")
		}

		logger.Infof("Calculating hibernate actions on %d stable installations", len(installations))
		var errorSkipCount, maxUserSkipCount int
		var installationsToHibernate []*cmodel.InstallationDTO
		for i, installation := range installations {
			current := i + 1
			if current%10 == 0 {
				logger.Debugf("Processing installation %d of %d", current, len(installations))
			}

			logger.WithField("installation", installation.ID)

			shouldHibernate, err := shouldHibernate(installation, userMetrics, tc, unlock, days, maxUsers)
			if shouldHibernate && err != nil {
				logger.WithField("reason", err.Error()).Info("Skipping valid hibernation target")
				maxUserSkipCount++
				continue
			}
			if err != nil {
				logger.WithError(err).Warn("Failed hibernation determination")
				errorSkipCount++
				continue
			}
			if !shouldHibernate {
				continue
			}

			installationsToHibernate = append(installationsToHibernate, installation)
		}

		logger.WithFields(logrus.Fields{
			"hibernation-count":              len(installationsToHibernate),
			"hibernation-calculation-errors": errorSkipCount,
			"hibernation-skip-from-users":    maxUserSkipCount,
		}).Info("Hibernation calculations complete")

		if len(installationsToHibernate) == 0 {
			logger.Info("No installations requre hibernation; exiting...")
			return nil
		}

		logger.Infof("Hibernating %d installations", len(installationsToHibernate))
		if dryrun {
			logger.Info("Dry run complete")
			return nil
		}

		for _, installation := range installationsToHibernate {
			logger.WithField("installation", installation.ID).Info("Hibernating installation")

			err = hibernateInstallation(installation, client)
			if err != nil {
				return errors.Wrap(err, "failed to hibernate installation")
			}

			// Another sleep to slow the API calls to the provisioner.
			time.Sleep(500 * time.Millisecond)
		}

		runtime := fmt.Sprintf("%s", time.Now().Sub(start))

		if len(webhookURL) != 0 {
			logger.Info("Sending hibernation report webhook")

			webhookText := fmt.Sprintf(hibernateReportMessage, wrapInlineCode(runID), runtime, len(installations), len(installationsToHibernate), maxUserSkipCount, errorSkipCount)
			err = sendHibernateReportWebhook(webhookURL, webhookText)
			if err != nil {
				logger.WithError(err).Error("Failed to send Mattermost webhook")
			}
		}

		logger.WithField("runtime", runtime).Info("Hibernation check complete")

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

// shouldHibernate determines if an installation should be hibernated or not.
// If the installation should be hibernated, but an error is also returned then
// that indicates that the installation meets hibernation criteria, but was also
// whitelisted due to another metric such as user count.
func shouldHibernate(installation *cmodel.InstallationDTO, userMetrics map[string]int64, mc metricsClient, unlock bool, days int, maxUsers int64) (bool, error) {
	if installation.State != cmodel.InstallationStateStable {
		return false, errors.Errorf("expected only stable installations (%s)", installation.State)
	}
	if installation.APISecurityLock && !unlock {
		return false, errors.New("installation is locked and hibernator is not set to perform unlocks")
	}

	// A small sleep to help prevent hitting the metrics host too hard.
	// Using the force a bit here. May need to be tweaked.
	time.Sleep(100 * time.Millisecond)

	hasNoNewPosts, err := mc.DetermineIfInstallationHasNoNewPosts(installation.ID, days)
	if err != nil {
		return false, errors.Wrap(err, "failed to deterimine if installation has new posts")
	}
	if !hasNoNewPosts {
		// The installation is still active.
		return false, nil
	}
	userCount, ok := userMetrics[installation.ID]
	if !ok {
		return false, errors.New("no user metrics found")
	}
	if userCount >= maxUsers {
		return true, errors.Errorf("installation would be hibernated, but has %d users", userCount)
	}

	return true, nil
}
