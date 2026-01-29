// SPDX-FileCopyrightText: 2026 Logan Lindquist Land
// SPDX-License-Identifier: FSL-1.1-MIT

package store

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/llbbl/repjan/internal/db"
	"github.com/llbbl/repjan/internal/github"
)

// setupTestStore creates an in-memory database and returns a Store for testing.
func setupTestStore(t *testing.T) *Store {
	t.Helper()

	database, err := db.Open(":memory:")
	require.NoError(t, err)

	t.Cleanup(func() {
		db.Close(database)
	})

	err = db.RunMigrations(database)
	require.NoError(t, err)

	return New(database)
}

// testRepo creates a test repository with the given name.
func testRepo(owner, name string) github.Repository {
	return github.Repository{
		Owner:           owner,
		Name:            name,
		Description:     "A test repository",
		StargazerCount:  42,
		ForkCount:       10,
		IsArchived:      false,
		IsFork:          false,
		IsPrivate:       false,
		PrimaryLanguage: "Go",
		PushedAt:        time.Now().Add(-24 * time.Hour),
		CreatedAt:       time.Now().Add(-30 * 24 * time.Hour),
	}
}

func TestUpsertRepositories_InsertsNewRepos(t *testing.T) {
	store := setupTestStore(t)
	owner := "testowner"

	repos := []github.Repository{
		testRepo(owner, "repo1"),
		testRepo(owner, "repo2"),
	}

	err := store.UpsertRepositories(owner, repos)
	require.NoError(t, err)

	// Verify repos were inserted
	result, err := store.GetRepositories(owner)
	require.NoError(t, err)
	assert.Len(t, result, 2)

	// Verify data was stored correctly
	repo1, err := store.GetRepository(owner, "repo1")
	require.NoError(t, err)
	assert.Equal(t, "repo1", repo1.Name)
	assert.Equal(t, owner, repo1.Owner)
	assert.Equal(t, "A test repository", repo1.Description)
	assert.Equal(t, 42, repo1.StargazerCount)
	assert.Equal(t, 10, repo1.ForkCount)
	assert.Equal(t, "Go", repo1.PrimaryLanguage)
}

func TestUpsertRepositories_UpdatesExistingRepos(t *testing.T) {
	store := setupTestStore(t)
	owner := "testowner"

	// Insert initial repo
	initialRepo := testRepo(owner, "myrepo")
	initialRepo.StargazerCount = 10
	initialRepo.Description = "Initial description"

	err := store.UpsertRepositories(owner, []github.Repository{initialRepo})
	require.NoError(t, err)

	// Update with new data
	updatedRepo := testRepo(owner, "myrepo")
	updatedRepo.StargazerCount = 100
	updatedRepo.Description = "Updated description"
	updatedRepo.IsArchived = true

	err = store.UpsertRepositories(owner, []github.Repository{updatedRepo})
	require.NoError(t, err)

	// Verify update
	result, err := store.GetRepository(owner, "myrepo")
	require.NoError(t, err)
	assert.Equal(t, 100, result.StargazerCount)
	assert.Equal(t, "Updated description", result.Description)
	assert.True(t, result.IsArchived)

	// Verify only one record exists
	allRepos, err := store.GetRepositories(owner)
	require.NoError(t, err)
	assert.Len(t, allRepos, 1)
}

func TestUpsertRepositories_EmptySlice(t *testing.T) {
	store := setupTestStore(t)

	err := store.UpsertRepositories("owner", []github.Repository{})
	require.NoError(t, err)
}

func TestUpsertRepositories_HandlesNullValues(t *testing.T) {
	store := setupTestStore(t)
	owner := "testowner"

	// Repo with empty optional fields
	repo := github.Repository{
		Owner:           owner,
		Name:            "minimal",
		Description:     "", // empty
		PrimaryLanguage: "", // empty
	}

	err := store.UpsertRepositories(owner, []github.Repository{repo})
	require.NoError(t, err)

	result, err := store.GetRepository(owner, "minimal")
	require.NoError(t, err)
	assert.Equal(t, "", result.Description)
	assert.Equal(t, "", result.PrimaryLanguage)
}

func TestUpsertRepositories_SetsOwnerFromParameter(t *testing.T) {
	store := setupTestStore(t)
	owner := "providedowner"

	// Repo without owner set
	repo := github.Repository{
		Name: "nowner",
	}

	err := store.UpsertRepositories(owner, []github.Repository{repo})
	require.NoError(t, err)

	result, err := store.GetRepository(owner, "nowner")
	require.NoError(t, err)
	assert.Equal(t, owner, result.Owner)
}

func TestGetRepositories_ReturnsCorrectData(t *testing.T) {
	store := setupTestStore(t)
	owner := "testowner"

	repos := []github.Repository{
		testRepo(owner, "alpha"),
		testRepo(owner, "beta"),
		testRepo(owner, "gamma"),
	}

	err := store.UpsertRepositories(owner, repos)
	require.NoError(t, err)

	result, err := store.GetRepositories(owner)
	require.NoError(t, err)
	assert.Len(t, result, 3)

	// Verify ordering (by name)
	assert.Equal(t, "alpha", result[0].Name)
	assert.Equal(t, "beta", result[1].Name)
	assert.Equal(t, "gamma", result[2].Name)
}

func TestGetRepositories_FiltersByOwner(t *testing.T) {
	store := setupTestStore(t)

	// Insert repos for two different owners
	err := store.UpsertRepositories("owner1", []github.Repository{
		testRepo("owner1", "repo1"),
		testRepo("owner1", "repo2"),
	})
	require.NoError(t, err)

	err = store.UpsertRepositories("owner2", []github.Repository{
		testRepo("owner2", "repo3"),
	})
	require.NoError(t, err)

	// Query for owner1
	result, err := store.GetRepositories("owner1")
	require.NoError(t, err)
	assert.Len(t, result, 2)

	// Query for owner2
	result, err = store.GetRepositories("owner2")
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "repo3", result[0].Name)
}

func TestGetRepositories_EmptyResultForUnknownOwner(t *testing.T) {
	store := setupTestStore(t)

	result, err := store.GetRepositories("nonexistent")
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestGetRepository_ExistingRepo(t *testing.T) {
	store := setupTestStore(t)
	owner := "testowner"

	repo := testRepo(owner, "myrepo")
	repo.IsPrivate = true
	repo.IsFork = true

	err := store.UpsertRepositories(owner, []github.Repository{repo})
	require.NoError(t, err)

	result, err := store.GetRepository(owner, "myrepo")
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "myrepo", result.Name)
	assert.Equal(t, owner, result.Owner)
	assert.True(t, result.IsPrivate)
	assert.True(t, result.IsFork)
}

func TestGetRepository_NonExisting(t *testing.T) {
	store := setupTestStore(t)

	result, err := store.GetRepository("owner", "nonexistent")
	assert.ErrorIs(t, err, ErrNotFound)
	assert.Nil(t, result)
}

func TestUpdateRepository_Success(t *testing.T) {
	store := setupTestStore(t)
	owner := "testowner"

	// Insert initial repo
	repo := testRepo(owner, "updateme")
	err := store.UpsertRepositories(owner, []github.Repository{repo})
	require.NoError(t, err)

	// Update it
	repo.StargazerCount = 999
	repo.Description = "Updated via UpdateRepository"
	repo.IsArchived = true

	err = store.UpdateRepository(repo)
	require.NoError(t, err)

	// Verify update
	result, err := store.GetRepository(owner, "updateme")
	require.NoError(t, err)
	assert.Equal(t, 999, result.StargazerCount)
	assert.Equal(t, "Updated via UpdateRepository", result.Description)
	assert.True(t, result.IsArchived)
}

func TestUpdateRepository_NotFound(t *testing.T) {
	store := setupTestStore(t)

	repo := testRepo("owner", "nonexistent")
	err := store.UpdateRepository(repo)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestDeleteStaleRepositories_RemovesOldEntries(t *testing.T) {
	store := setupTestStore(t)
	owner := "testowner"

	// Insert repos
	err := store.UpsertRepositories(owner, []github.Repository{
		testRepo(owner, "repo1"),
		testRepo(owner, "repo2"),
		testRepo(owner, "repo3"),
	})
	require.NoError(t, err)

	// All repos should exist
	repos, err := store.GetRepositories(owner)
	require.NoError(t, err)
	assert.Len(t, repos, 3)

	// Delete repos older than future time (should delete all)
	futureTime := time.Now().Add(1 * time.Hour)
	deleted, err := store.DeleteStaleRepositories(owner, futureTime)
	require.NoError(t, err)
	assert.Equal(t, int64(3), deleted)

	// Verify all deleted
	repos, err = store.GetRepositories(owner)
	require.NoError(t, err)
	assert.Empty(t, repos)
}

func TestDeleteStaleRepositories_PreservesRecentEntries(t *testing.T) {
	store := setupTestStore(t)
	owner := "testowner"

	// Insert repos
	err := store.UpsertRepositories(owner, []github.Repository{
		testRepo(owner, "repo1"),
	})
	require.NoError(t, err)

	// Delete repos older than past time (should delete none)
	pastTime := time.Now().Add(-1 * time.Hour)
	deleted, err := store.DeleteStaleRepositories(owner, pastTime)
	require.NoError(t, err)
	assert.Equal(t, int64(0), deleted)

	// Verify repo still exists
	repos, err := store.GetRepositories(owner)
	require.NoError(t, err)
	assert.Len(t, repos, 1)
}

func TestDeleteStaleRepositories_FiltersByOwner(t *testing.T) {
	store := setupTestStore(t)

	// Insert repos for two owners
	err := store.UpsertRepositories("owner1", []github.Repository{
		testRepo("owner1", "repo1"),
	})
	require.NoError(t, err)

	err = store.UpsertRepositories("owner2", []github.Repository{
		testRepo("owner2", "repo2"),
	})
	require.NoError(t, err)

	// Delete stale for owner1 only
	futureTime := time.Now().Add(1 * time.Hour)
	deleted, err := store.DeleteStaleRepositories("owner1", futureTime)
	require.NoError(t, err)
	assert.Equal(t, int64(1), deleted)

	// owner1 repos should be gone
	repos, err := store.GetRepositories("owner1")
	require.NoError(t, err)
	assert.Empty(t, repos)

	// owner2 repos should still exist
	repos, err = store.GetRepositories("owner2")
	require.NoError(t, err)
	assert.Len(t, repos, 1)
}

func TestGetLastSyncTime_ReturnsMaxSyncTime(t *testing.T) {
	store := setupTestStore(t)
	owner := "testowner"

	// Insert repos (they will all have the same sync time from the single upsert)
	err := store.UpsertRepositories(owner, []github.Repository{
		testRepo(owner, "repo1"),
		testRepo(owner, "repo2"),
	})
	require.NoError(t, err)

	syncTime, err := store.GetLastSyncTime(owner)
	require.NoError(t, err)

	// Sync time should be very recent (within last second)
	assert.WithinDuration(t, time.Now(), syncTime, 1*time.Second)
}

func TestGetLastSyncTime_ZeroForNoRepos(t *testing.T) {
	store := setupTestStore(t)

	syncTime, err := store.GetLastSyncTime("nonexistent")
	require.NoError(t, err)
	assert.True(t, syncTime.IsZero())
}

func TestGetLastSyncTime_FiltersByOwner(t *testing.T) {
	store := setupTestStore(t)

	// Insert repos for owner1
	err := store.UpsertRepositories("owner1", []github.Repository{
		testRepo("owner1", "repo1"),
	})
	require.NoError(t, err)

	// owner2 should have zero sync time
	syncTime, err := store.GetLastSyncTime("owner2")
	require.NoError(t, err)
	assert.True(t, syncTime.IsZero())

	// owner1 should have recent sync time
	syncTime, err = store.GetLastSyncTime("owner1")
	require.NoError(t, err)
	assert.False(t, syncTime.IsZero())
}

func TestUpsertRepositories_UpdatesSyncedAt(t *testing.T) {
	store := setupTestStore(t)
	owner := "testowner"

	// Insert repo
	err := store.UpsertRepositories(owner, []github.Repository{
		testRepo(owner, "repo1"),
	})
	require.NoError(t, err)

	firstSyncTime, err := store.GetLastSyncTime(owner)
	require.NoError(t, err)

	// Small delay to ensure time difference
	time.Sleep(10 * time.Millisecond)

	// Upsert same repo again
	err = store.UpsertRepositories(owner, []github.Repository{
		testRepo(owner, "repo1"),
	})
	require.NoError(t, err)

	secondSyncTime, err := store.GetLastSyncTime(owner)
	require.NoError(t, err)

	// Second sync time should be after first
	assert.True(t, secondSyncTime.After(firstSyncTime) || secondSyncTime.Equal(firstSyncTime))
}

func TestStore_TimeFieldsPreserved(t *testing.T) {
	store := setupTestStore(t)
	owner := "testowner"

	pushedAt := time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC)
	createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

	repo := github.Repository{
		Owner:     owner,
		Name:      "timerepo",
		PushedAt:  pushedAt,
		CreatedAt: createdAt,
	}

	err := store.UpsertRepositories(owner, []github.Repository{repo})
	require.NoError(t, err)

	result, err := store.GetRepository(owner, "timerepo")
	require.NoError(t, err)

	// Times should be preserved (comparing in UTC)
	assert.True(t, result.PushedAt.Equal(pushedAt))
	assert.True(t, result.CreatedAt.Equal(createdAt))
}

func TestStore_DaysSinceActivityCalculated(t *testing.T) {
	store := setupTestStore(t)
	owner := "testowner"

	// Repo with PushedAt 30 days ago and no DaysSinceActivity set
	repo := github.Repository{
		Owner:    owner,
		Name:     "oldrepo",
		PushedAt: time.Now().Add(-30 * 24 * time.Hour),
	}

	err := store.UpsertRepositories(owner, []github.Repository{repo})
	require.NoError(t, err)

	result, err := store.GetRepository(owner, "oldrepo")
	require.NoError(t, err)

	// Should have calculated ~30 days
	assert.InDelta(t, 30, result.DaysSinceActivity, 1)
}
