// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"errors"
	"testing"

	cmodel "github.com/mattermost/mattermost-cloud/model"
	"github.com/stretchr/testify/assert"
)

func TestShouldHibernate(t *testing.T) {
	mc := newMockMetricsClient()
	mc.newPostCount = 10
	installation := &cmodel.InstallationDTO{
		Installation: &cmodel.Installation{
			ID:              cmodel.NewID(),
			State:           cmodel.InstallationStateStable,
			APISecurityLock: true,
		},
	}
	userMetrics := map[string]int64{
		installation.ID: 5,
	}

	creationCutoff := int64(999999999999999999)

	logger := logger.WithField("fleet-controller", "hibernate")

	t.Run("hibernator can't unlock", func(t *testing.T) {
		shouldHibernate, err := shouldHibernate(installation, userMetrics, mc, false, 7, 100, creationCutoff, logger)
		assert.False(t, shouldHibernate)
		assert.Error(t, err)
	})

	t.Run("installation has new posts", func(t *testing.T) {
		shouldHibernate, err := shouldHibernate(installation, userMetrics, mc, true, 7, 100, creationCutoff, logger)
		assert.False(t, shouldHibernate)
		assert.NoError(t, err)
	})

	t.Run("installation has no new posts", func(t *testing.T) {
		mc.newPostCount = 0
		shouldHibernate, err := shouldHibernate(installation, userMetrics, mc, true, 7, 100, creationCutoff, logger)
		assert.True(t, shouldHibernate)
		assert.NoError(t, err)
		mc.newPostCount = 10
	})

	t.Run("installation has no user metrics", func(t *testing.T) {
		mc.newPostCount = 0
		shouldHibernate, err := shouldHibernate(installation, make(map[string]int64), mc, true, 7, 100, creationCutoff, logger)
		assert.False(t, shouldHibernate)
		assert.Error(t, err)
		mc.newPostCount = 10
	})

	t.Run("installation no new posts, but more than maxUsers", func(t *testing.T) {
		mc.newPostCount = 0
		shouldHibernate, err := shouldHibernate(installation, userMetrics, mc, true, 7, 4, creationCutoff, logger)
		assert.True(t, shouldHibernate)
		assert.Error(t, err)
		mc.newPostCount = 10
	})

	t.Run("installation has a user metric count of 0", func(t *testing.T) {
		mc.newPostCount = 0
		userMetrics[installation.ID] = 0
		shouldHibernate, err := shouldHibernate(installation, userMetrics, mc, true, 7, 100, creationCutoff, logger)
		assert.False(t, shouldHibernate)
		assert.Error(t, err)
		mc.newPostCount = 10
		userMetrics[installation.ID] = 5
	})

	t.Run("error getting post metrics", func(t *testing.T) {
		mc.newPostsError = errors.New("test")
		shouldHibernate, err := shouldHibernate(installation, userMetrics, mc, true, 7, 4, creationCutoff, logger)
		assert.False(t, shouldHibernate)
		assert.Error(t, err)
		mc.newPostsError = nil
	})

	t.Run("installation not stable", func(t *testing.T) {
		installation.State = cmodel.InstallationStateUpdateInProgress
		shouldHibernate, err := shouldHibernate(installation, userMetrics, mc, true, 7, 100, creationCutoff, logger)
		assert.False(t, shouldHibernate)
		assert.Error(t, err)
	})

	t.Run("installation was created recently", func(t *testing.T) {
		installation.State = cmodel.ClusterInstallationStateStable
		shouldHibernate, err := shouldHibernate(installation, userMetrics, mc, true, 7, 100, 0, logger)
		assert.False(t, shouldHibernate)
		assert.NoError(t, err)
	})
}
