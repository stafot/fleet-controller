// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	cmodel "github.com/mattermost/mattermost-cloud/model"
)

func init() {
	wakeupCmd.PersistentFlags().String("server", "http://localhost:8075", "The provisioning server whose API will be queried.")
	wakeupCmd.PersistentFlags().Bool("dry-run", true, "Whether the fleet controller will perform actions or just print actions that would be taken.")
	wakeupCmd.PersistentFlags().Bool("unlock", false, "Whether the fleet controller will unlock installations to wake them up or not.")

	// Installation filters
	wakeupCmd.PersistentFlags().String("owner", "", "The owner ID value to filter installations by.")
	wakeupCmd.PersistentFlags().String("group", "", "The group ID value to filter installations by.")
}

var wakeupCmd = &cobra.Command{
	Use:   "wake-up",
	Short: "Wake up installations",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		productionLogs, _ := command.Flags().GetBool("production-logs")
		logger := setupLogger("wake-up", productionLogs)

		logger.Info("Waking up installations")

		start := time.Now()

		serverAddress, _ := command.Flags().GetString("server")
		dryrun, _ := command.Flags().GetBool("dry-run")
		unlock, _ := command.Flags().GetBool("unlock")
		owner, _ := command.Flags().GetString("owner")
		group, _ := command.Flags().GetString("group")

		if len(serverAddress) == 0 {
			return errors.New("server value must be defined")
		}

		client := cmodel.NewClient(serverAddress)

		logger.WithFields(log.Fields{
			"owner-filter": owner,
			"group-filter": group,
		}).Info("Obtaining current installations")
		installations, err := client.GetInstallations(&cmodel.GetInstallationsRequest{
			State:                       cmodel.InstallationStateHibernating,
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

		logger.Infof("Calculating wake up actions on %d hibernating installations", len(installations))
		var errorSkipCount int
		var installationsToWakeUp []*cmodel.InstallationDTO
		for i, installation := range installations {
			current := i + 1
			if current%10 == 0 {
				logger.Debugf("Processing installation %d of %d", current, len(installations))
			}

			logger := logger.WithField("installation", installation.ID)

			err := shouldWakeUp(installation, unlock)
			if err != nil {
				logger.WithError(err).Warn("Failed wake up determination")
				errorSkipCount++
				continue
			}

			installationsToWakeUp = append(installationsToWakeUp, installation)
		}

		logger.WithFields(log.Fields{
			"wakeup-count":              len(installationsToWakeUp),
			"wakeup-calculation-errors": errorSkipCount,
		}).Info("Wake up calculations complete")

		if len(installationsToWakeUp) == 0 {
			logger.Info("No installations requre waking up; exiting...")
			return nil
		}

		logger.Infof("Waking %d installations up", len(installationsToWakeUp))
		if dryrun {
			logger.Info("Dry run complete")
			return nil
		}

		for _, installation := range installationsToWakeUp {
			logger.WithField("installation", installation.ID).Info("Waking installation up")

			err = wakeupInstallation(installation, client)
			if err != nil {
				return errors.Wrap(err, "failed to wake up installation")
			}

			// Another sleep to slow the API calls to the provisioner.
			time.Sleep(500 * time.Millisecond)
		}

		runtime := fmt.Sprintf("%s", time.Now().Sub(start))

		logger.WithField("runtime", runtime).Info("Wake up check complete")

		return nil
	},
}

func wakeupInstallation(installation *cmodel.InstallationDTO, client *cmodel.Client) error {
	var relock bool
	var err error

	if installation.APISecurityLock {
		err = client.UnlockAPIForInstallation(installation.ID)
		if err != nil {
			return errors.Wrapf(err, "failed to unlock installation %s", installation.ID)
		}
		relock = true
	}

	installation, err = client.WakeupInstallation(installation.ID)
	if err != nil {
		return errors.Wrap(err, "failed to wake up installation")
	}

	if relock {
		err = client.LockAPIForInstallation(installation.ID)
		if err != nil {
			return errors.Wrapf(err, "failed to relock installation %s", installation.ID)
		}
	}

	return nil
}

func shouldWakeUp(installation *cmodel.InstallationDTO, unlock bool) error {
	if installation.State != cmodel.InstallationStateHibernating {
		return errors.Errorf("expected only hibernating installations (%s)", installation.State)
	}
	if installation.APISecurityLock && !unlock {
		return errors.New("installation is locked and fleet controller is not set to perform unlocks")
	}

	return nil
}
