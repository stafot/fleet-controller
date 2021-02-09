// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"os"

	cmodel "github.com/mattermost/mattermost-cloud/model"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

var logger *log.Logger

func init() {
	logger = log.New()
	logger.SetLevel(log.DebugLevel)
	log.SetOutput(os.Stdout)
	logger.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})
}

func setupProductionLogging(l log.FieldLogger) (*log.Entry, string) {
	logger.SetFormatter(&logrus.JSONFormatter{})

	runID := cmodel.NewID()
	return l.WithField("run", runID), runID
}
