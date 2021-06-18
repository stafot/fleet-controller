// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"math/rand"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/mattermost/fleet-controller/internal/metrics"
	"github.com/mattermost/fleet-controller/model"
	cmodel "github.com/mattermost/mattermost-cloud/model"
)

func init() {
	scaleCmd.PersistentFlags().String("server", "http://localhost:8075", "The provisioning server whose API will be queried.")
	scaleCmd.PersistentFlags().String("thanos-url", "", "The URL to query thanos metrics from.")
	scaleCmd.PersistentFlags().Bool("dry-run", true, "Whether the autoscaler will perform scaling actions or just print actions that would be taken.")
	scaleCmd.PersistentFlags().Bool("unlock", false, "Whether the autoscaler will unlock installations to update their size or not.")
	scaleCmd.PersistentFlags().Int64("max-updating", 5, "The maximum number of installations that can be currently updating before resizing another batch.")
	scaleCmd.PersistentFlags().Int32("batch-size", 3, "The maximum number of installations to resize in a single batch.")

	scaleCmd.PersistentFlags().Bool("fun-mode", true, "Randomizes installation scaling order when disabled which distributes load better. Turn this off if you hate adventure, being generally awesome, and hanging out with the cloud family in the prod alerts channel...")

	// Installation filters
	scaleCmd.PersistentFlags().String("owner", "", "The owner ID value to filter installations by.")
	scaleCmd.PersistentFlags().String("group", "", "The group ID value to filter installations by.")
}

var scaleCmd = &cobra.Command{
	Use:   "scale",
	Short: "Scale installations based on user counts",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		productionLogs, _ := command.Flags().GetBool("production-logs")
		logger := setupLogger("scale", productionLogs)

		logger.Info("Starting installation autoscaler")

		serverAddress, _ := command.Flags().GetString("server")
		thanosURL, _ := command.Flags().GetString("thanos-url")
		dryrun, _ := command.Flags().GetBool("dry-run")
		unlock, _ := command.Flags().GetBool("unlock")
		funMode, _ := command.Flags().GetBool("fun-mode")
		maxUpdating, _ := command.Flags().GetInt64("max-updating")
		batchSize, _ := command.Flags().GetInt32("batch-size")
		owner, _ := command.Flags().GetString("owner")
		group, _ := command.Flags().GetString("group")

		if len(serverAddress) == 0 {
			return errors.New("server value must be defined")
		}
		if len(thanosURL) == 0 {
			return errors.New("thanos-url value must be defined")
		}

		client := cmodel.NewClient(serverAddress)

		for {
			logger.Info("Obtaining current installation sizes")
			installations, err := client.GetInstallations(&cmodel.GetInstallationsRequest{
				OwnerID:                     owner,
				GroupID:                     group,
				State:                       cmodel.InstallationStateStable,
				IncludeGroupConfig:          false,
				IncludeGroupConfigOverrides: false,
				Paging: cmodel.Paging{
					Page:           0,
					PerPage:        cmodel.AllPerPage,
					IncludeDeleted: false,
				},
			})
			if err != nil {
				return errors.Wrap(err, "failed to get installations")
			}

			var scaled, updating int32
			if !model.InstallationsUpdatingIsBelowMax(maxUpdating, client, logger) {
				logger.Info("Requeing scale actions after 15 seconds")
				time.Sleep(15 * time.Second)
				continue
			}

			tc := metrics.NewThanosClient(thanosURL)

			logger.Info("Gathering installation user metrics")
			metrics, err := tc.GetInstallationUserMetrics()
			if err != nil {
				return errors.Wrap(err, "failed to obtain installation metrics")
			}

			if !funMode {
				rand.Seed(time.Now().UnixNano())
				rand.Shuffle(len(installations), func(i, j int) {
					installations[i], installations[j] = installations[j], installations[i]
				})
			}

			logger.Info("Calculating scale actions")
			for _, installation := range installations {
				if batchSize != 0 && scaled >= batchSize {
					break
				}

				userCount, ok := metrics[installation.ID]
				if !ok {
					logger.Warnf("%s - No user metrics found; skipping...", installation.ID)
					continue
				}
				newSize, err := getSuggestedScaleSize(installation.Size, userCount)
				if err != nil {
					return errors.Wrap(err, "failed to determine if installation should be scaled")
				}

				if installation.Size != newSize {
					logger.Debugf("%s - %s -> %s (%d users)", installation.ID, installation.Size, newSize, userCount)

					if installation.State != cmodel.InstallationStateStable {
						logger.Warnf("%s - Installation is not stable; skipping...", installation.ID)
						continue
					}

					if installation.APISecurityLock && !unlock {
						logger.Warnf("%s - Installation is locked and autoscaler is not set to perform unlocks; skipping...", installation.ID)
						continue
					}

					scaled++
				}
				if dryrun || installation.Size == newSize {
					continue
				}

				// Take resizing action.
				err = scaleInstallation(newSize, installation, client)
				if err != nil {
					return errors.Wrap(err, "failed to scale installation")
				}

				time.Sleep(500 * time.Millisecond)
			}

			logger.Infof("Scaling Stats: %d total, %d scale, %d currently updating", len(installations), scaled, updating)

			if scaled == 0 || dryrun {
				break
			}

			logger.Info("Requeing scale actions after 15 seconds")
			time.Sleep(15 * time.Second)
		}

		if dryrun {
			logger.Info("Dry run complete")
		} else {
			logger.Info("Scaling complete")
		}

		return nil
	},
}

func scaleInstallation(newSize string, installation *cmodel.InstallationDTO, client *cmodel.Client) error {
	var relock bool
	var err error

	if installation.APISecurityLock {
		err = client.UnlockAPIForInstallation(installation.ID)
		if err != nil {
			return errors.Wrapf(err, "failed to unlock installation %s", installation.ID)
		}
		relock = true
	}

	installation, err = client.UpdateInstallation(installation.ID, &cmodel.PatchInstallationRequest{
		Size: &newSize,
	})
	if err != nil {
		return errors.Wrap(err, "failed to update installation size")
	}

	if relock {
		err = client.LockAPIForInstallation(installation.ID)
		if err != nil {
			return errors.Wrapf(err, "failed to relock installation %s", installation.ID)
		}
	}

	return nil
}
