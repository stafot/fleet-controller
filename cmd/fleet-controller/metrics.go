// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

type metricsClient interface {
	GetInstallationUserMetrics() (map[string]int64, error)
	DetermineIfInstallationHasNoNewPosts(installationID string, days int) (bool, error)
}
