// SPDX-FileCopyrightText: 2026 Logan Lindquist Land
// SPDX-License-Identifier: FSL-1.1-MIT

package testutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewTestRepo_DefaultValues(t *testing.T) {
	repo := NewTestRepo()

	assert.Equal(t, "testowner", repo.Owner)
	assert.Equal(t, "testrepo", repo.Name)
	assert.Equal(t, "A test repository", repo.Description)
	assert.Equal(t, 5, repo.StargazerCount)
	assert.Equal(t, 2, repo.ForkCount)
	assert.False(t, repo.IsArchived)
	assert.False(t, repo.IsFork)
	assert.False(t, repo.IsPrivate)
	assert.Equal(t, "Go", repo.PrimaryLanguage)
}

func TestNewTestRepo_WithOptions(t *testing.T) {
	repo := NewTestRepo(
		WithOwner("customowner"),
		WithName("customrepo"),
		WithStars(100),
		WithForks(50),
		WithArchived(true),
	)

	assert.Equal(t, "customowner", repo.Owner)
	assert.Equal(t, "customrepo", repo.Name)
	assert.Equal(t, 100, repo.StargazerCount)
	assert.Equal(t, 50, repo.ForkCount)
	assert.True(t, repo.IsArchived)
}

func TestNewTestRepo_CalculatesDaysInactive(t *testing.T) {
	repo := NewTestRepo(WithDaysInactive(100))

	// The DaysSinceActivity should be approximately 100
	// Allow for slight timing variations
	assert.InDelta(t, 100, repo.DaysSinceActivity, 1)
}
