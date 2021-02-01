// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	viper.SetEnvPrefix("FC")
	viper.AutomaticEnv()

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
