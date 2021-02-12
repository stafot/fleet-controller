// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

type mockMetricsClient struct {
	finalUserMetrics map[string]int64
	userError        error

	newPostCount  float64
	newPostsError error
}

func newMockMetricsClient() *mockMetricsClient {
	return &mockMetricsClient{}
}

func (mc *mockMetricsClient) GetInstallationUserMetrics() (map[string]int64, error) {
	return mc.finalUserMetrics, mc.userError
}

func (mc *mockMetricsClient) GetInstallationNewPostCount(installationID string, days int) (float64, error) {
	return mc.newPostCount, mc.newPostsError
}
