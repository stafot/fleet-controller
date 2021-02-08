// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

type mockMetricsClient struct {
	finalUserMetrics map[string]int64
	userError        error

	installationHasNoNewPosts bool
	newPostsError             error
}

func newMockMetricsClient() *mockMetricsClient {
	return &mockMetricsClient{}
}

func (mc *mockMetricsClient) GetInstallationUserMetrics() (map[string]int64, error) {
	return mc.finalUserMetrics, mc.userError
}

func (mc *mockMetricsClient) DetermineIfInstallationHasNoNewPosts(installationID string, days int) (bool, error) {
	return mc.installationHasNoNewPosts, mc.newPostsError
}
