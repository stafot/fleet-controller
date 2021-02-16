// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"os"

	"github.com/mattermost/mattermost-cloud/model"
	"github.com/ory/viper"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var runID string

func init() {
	runID = model.NewID()

	viper.SetEnvPrefix("FC")
	viper.AutomaticEnv()

	rootCmd.PersistentFlags().Bool("production-logs", viper.GetBool("PRODUCTION_LOGS"), "Set log output with production settings | ENV: FC_PRODUCTION_LOGS")
	rootCmd.PersistentFlags().String("mm-webhook-url", viper.GetString("MM_WEBHOOK_URL"), "Optional Mattmost incoming webhook URL to send information on actions taken by fleet controller | ENV: FC_MM_WEBHOOK_URL")

	rootCmd.AddCommand(scaleCmd)
	rootCmd.AddCommand(hibernate)
	rootCmd.AddCommand(wakeupCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		logger.Error(errors.Wrap(err, "Command failed").Error())
		webhookURL, _ := rootCmd.Flags().GetString("mm-webhook-url")
		sendErrorWebhook(webhookURL, runID, err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:           "fleet-controller",
	Short:         "The fleet controller manages configuration of the fleet of Mattermost Cloud installations.",
	SilenceErrors: true,
}
