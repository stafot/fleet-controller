// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"os"

	"github.com/ory/viper"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func init() {
	viper.SetEnvPrefix("FC")
	viper.AutomaticEnv()

	rootCmd.PersistentFlags().Bool("production-logs", viper.GetBool("PRODUCTION_LOGS"), "Set log output with production settings | ENV: FC_PRODUCTION_LOGS")

	rootCmd.AddCommand(scaleCmd)
	rootCmd.AddCommand(hibernate)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		logger.Error(errors.Wrap(err, "Command failed").Error())
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:           "fleet-controller",
	Short:         "The fleet controller manages configuration of the fleet of Mattermost Cloud installations.",
	SilenceErrors: true,
}
