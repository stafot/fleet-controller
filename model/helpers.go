package model

import (
	cmodel "github.com/mattermost/mattermost-cloud/model"
	log "github.com/sirupsen/logrus"
)

// InstallationsUpdatingIsBelowMax whether the number of installations updating
// on a given cloud server is below a maximum value or not.
func InstallationsUpdatingIsBelowMax(max int64, client *cmodel.Client, logger log.FieldLogger) bool {
	status, err := client.GetInstallationsStatus()
	if err != nil {
		logger.WithError(err).Error("Failed to get updating installation count")
		return false
	}

	logger.Debugf("%d installations are currently updating (max %d)", status.InstallationsUpdating, max)

	return status.InstallationsUpdating < max
}
