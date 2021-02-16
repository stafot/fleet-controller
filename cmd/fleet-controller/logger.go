// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"os"

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

func setupLogger(cmd string, production bool) *log.Entry {
	l := logger.WithField("fleet-controller", cmd)
	if production {
		logger.SetFormatter(&logrus.JSONFormatter{})
		l = l.WithField("run", runID)
	}

	return l
}
