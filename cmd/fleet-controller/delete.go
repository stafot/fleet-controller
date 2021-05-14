// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"bufio"
	"fmt"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	cmodel "github.com/mattermost/mattermost-cloud/model"
)

func init() {
	deleteCmd.PersistentFlags().String("server", "http://localhost:8075", "The provisioning server whose API will be queried.")
	deleteCmd.PersistentFlags().String("file", "installations.txt", "Location of file containing installation IDs to be deleted. File should contain only IDs separated by a newline.")
	deleteCmd.PersistentFlags().Bool("dry-run", true, "Whether the autoscaler will perform scaling actions or just print actions that would be taken.")
	deleteCmd.PersistentFlags().Bool("unlock", false, "Whether the autoscaler will unlock installations to update their size or not.")
}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete hibernating installations",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		productionLogs, _ := command.Flags().GetBool("production-logs")
		logger := setupLogger("delete", productionLogs)

		logger.Info("Starting installation deletion")

		start := time.Now()

		serverAddress, _ := command.Flags().GetString("server")
		file, _ := command.Flags().GetString("file")
		dryrun, _ := command.Flags().GetBool("dry-run")
		unlock, _ := command.Flags().GetBool("unlock")

		if len(serverAddress) == 0 {
			return errors.New("server value must be defined")
		}

		client := cmodel.NewClient(serverAddress)

		installationIDs, err := readInInstallationIDs(file)
		if err != nil {
			return err
		}

		logger.Infof("Deleting %d installations", len(installationIDs))

		var deletedInstallations []string

		timer := time.NewTimer(3 * time.Hour)
		maxUpdating := int64(25)
		var installationToDeleteIndex int
		for {
			updating, err := getCurrentInstallationUpdatingCount(client)
			if err != nil {
				// TODO: maybe allow for a few retries before giving up, but
				// let's play it safe for now.
				return errors.Wrap(err, "failed to get current updating count")
			}

			logger.Debugf("%d installations are currently updating (max %d)", updating, maxUpdating)
			if updating < maxUpdating {
				// Delete up to 5 installations at a time.
				for i := 1; i <= 5 && installationToDeleteIndex < len(installationIDs); i++ {
					installation, err := client.GetInstallation(installationIDs[installationToDeleteIndex], &cmodel.GetInstallationRequest{})
					if err != nil {
						return errors.Wrap(err, "failed to get installation")
					}
					if installation == nil {
						logger.Info("Could not find installation")
						installationToDeleteIndex++
						continue
					}
					err = ensureSafeToDelete(installation, unlock)
					if err != nil {
						logger.WithError(err).Warn("Skipping installation deletion")
						installationToDeleteIndex++
						continue
					}

					logger.WithField("installation", installation.ID).Infof("Deleting installation %d/%d", installationToDeleteIndex+1, len(installationIDs))

					if !dryrun {
						err = deleteInstallation(installation, client)
						if err != nil {
							return errors.Wrap(err, "failed to delete installation")
						}
						deletedInstallations = append(deletedInstallations, installation.ID)
					}
					installationToDeleteIndex++

					// Another sleep to slow the API calls to the provisioner.
					time.Sleep(100 * time.Millisecond)
				}
			}

			if installationToDeleteIndex >= len(installationIDs) {
				break
			}

			select {
			case <-time.After(3 * time.Second):
				continue
			case <-timer.C:
				return errors.Errorf("timed out after 3 hours trying to delete %d installations", len(installationIDs))
			}
		}

		runtime := fmt.Sprintf("%s", time.Now().Sub(start))

		logger.WithField("runtime", runtime).Info("Instalaltion deletion complete")

		return nil
	},
}

func deleteInstallation(installation *cmodel.InstallationDTO, client *cmodel.Client) error {
	var err error

	if installation.APISecurityLock {
		err = client.UnlockAPIForInstallation(installation.ID)
		if err != nil {
			return errors.Wrapf(err, "failed to unlock installation %s", installation.ID)
		}
	}

	err = client.DeleteInstallation(installation.ID)
	if err != nil {
		return errors.Wrap(err, "failed to delete installation")
	}

	return nil
}

func ensureSafeToDelete(installation *cmodel.InstallationDTO, unlock bool) error {
	if installation.State != cmodel.InstallationStateHibernating {
		return errors.Errorf("expected only hibernating installations (%s)", installation.State)
	}
	if installation.APISecurityLock && !unlock {
		return errors.New("installation is locked and fleet controller is not set to perform unlocks")
	}

	return nil
}

func readInInstallationIDs(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var installationIDs []string

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		installationIDs = append(installationIDs, scanner.Text())
	}

	err = scanner.Err()
	if err != nil {
		return nil, err
	}

	return installationIDs, nil
}
