// SPDX-FileCopyrightText: 2026 Logan Lindquist Land
// SPDX-License-Identifier: FSL-1.1-MIT

package sync

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSyncMsgType_Constants(t *testing.T) {
	// Verify the enum values are distinct and start at 0
	assert.Equal(t, SyncMsgType(0), SyncStarted)
	assert.Equal(t, SyncMsgType(1), SyncCompleted)
	assert.Equal(t, SyncMsgType(2), SyncError)

	// Verify they are not equal to each other
	assert.NotEqual(t, SyncStarted, SyncCompleted)
	assert.NotEqual(t, SyncCompleted, SyncError)
	assert.NotEqual(t, SyncStarted, SyncError)
}

func TestSyncMsg_DefaultValues(t *testing.T) {
	msg := SyncMsg{}

	// Default type should be SyncStarted (0)
	assert.Equal(t, SyncStarted, msg.Type)
	// Default repos should be nil
	assert.Nil(t, msg.Repos)
	// Default error should be nil
	assert.Nil(t, msg.Error)
}

func TestSyncResult_DefaultValues(t *testing.T) {
	result := SyncResult{}

	// Default repos should be nil
	assert.Nil(t, result.Repos)
	// Default error should be nil
	assert.Nil(t, result.Error)
}
