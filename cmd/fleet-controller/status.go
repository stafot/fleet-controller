package main

import (
	cmodel "github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
)

// getCurrentInstallationUpdatingCount returns the number of installations that
// are in all groups that are updating.
// TODO: add new provisioner status endpoint to give stats on all installations
// and use that instead.
func getCurrentInstallationUpdatingCount(client *cmodel.Client) (int64, error) {
	statuses, err := client.GetGroupsStatus()
	if err != nil {
		return 0, errors.Wrap(err, "failed to get group statuses")
	}

	var totalUpdating int64
	for _, status := range *statuses {
		totalUpdating += status.Status.InstallationsUpdating
	}

	return totalUpdating, nil
}
