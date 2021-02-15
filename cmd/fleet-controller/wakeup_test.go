// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"testing"

	cmodel "github.com/mattermost/mattermost-cloud/model"
	"github.com/stretchr/testify/assert"
)

func TestSholdWakeUp(t *testing.T) {
	installation := &cmodel.InstallationDTO{
		Installation: &cmodel.Installation{
			ID:              cmodel.NewID(),
			State:           cmodel.InstallationStateHibernating,
			APISecurityLock: true,
		},
	}

	t.Run("fleet controller can't unlock", func(t *testing.T) {
		err := shouldWakeUp(installation, false)
		assert.Error(t, err)
	})

	t.Run("installation is hibernating and can unlock", func(t *testing.T) {
		err := shouldWakeUp(installation, true)
		assert.NoError(t, err)
	})

	t.Run("installation not hibernating", func(t *testing.T) {
		installation.State = cmodel.InstallationStateStable
		err := shouldWakeUp(installation, true)
		assert.Error(t, err)
	})
}
