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

// Tests for marked repos functionality

func TestSaveMarkedRepos_InsertsNewMarks(t *testing.T) {
	store := setupTestStore(t)
	owner := "testowner"

	err := store.SaveMarkedRepos(owner, []string{"repo1", "repo2", "repo3"})
	require.NoError(t, err)

	// Verify marks were saved
	marks, err := store.GetMarkedRepos(owner)
	require.NoError(t, err)
	assert.Len(t, marks, 3)
	assert.Contains(t, marks, "repo1")
	assert.Contains(t, marks, "repo2")
	assert.Contains(t, marks, "repo3")
}

func TestSaveMarkedRepos_ReplacesExistingMarks(t *testing.T) {
	store := setupTestStore(t)
	owner := "testowner"

	// Insert initial marks
	err := store.SaveMarkedRepos(owner, []string{"repo1", "repo2"})
	require.NoError(t, err)

	// Replace with new marks
	err = store.SaveMarkedRepos(owner, []string{"repo3", "repo4"})
	require.NoError(t, err)

	// Verify old marks are gone and new marks exist
	marks, err := store.GetMarkedRepos(owner)
	require.NoError(t, err)
	assert.Len(t, marks, 2)
	assert.NotContains(t, marks, "repo1")
	assert.NotContains(t, marks, "repo2")
	assert.Contains(t, marks, "repo3")
	assert.Contains(t, marks, "repo4")
}

func TestSaveMarkedRepos_EmptySliceClearsMarks(t *testing.T) {
	store := setupTestStore(t)
	owner := "testowner"

	// Insert marks
	err := store.SaveMarkedRepos(owner, []string{"repo1", "repo2"})
	require.NoError(t, err)

	// Clear marks by saving empty slice
	err = store.SaveMarkedRepos(owner, []string{})
	require.NoError(t, err)

	// Verify marks are gone
	marks, err := store.GetMarkedRepos(owner)
	require.NoError(t, err)
	assert.Empty(t, marks)
}

func TestGetMarkedRepos_ReturnsEmptyForNoMarks(t *testing.T) {
	store := setupTestStore(t)

	marks, err := store.GetMarkedRepos("nonexistent")
	require.NoError(t, err)
	assert.Empty(t, marks)
}

func TestGetMarkedRepos_FiltersByOwner(t *testing.T) {
	store := setupTestStore(t)

	// Save marks for two owners
	err := store.SaveMarkedRepos("owner1", []string{"repo1", "repo2"})
	require.NoError(t, err)

	err = store.SaveMarkedRepos("owner2", []string{"repo3"})
	require.NoError(t, err)

	// Query for owner1
	marks, err := store.GetMarkedRepos("owner1")
	require.NoError(t, err)
	assert.Len(t, marks, 2)

	// Query for owner2
	marks, err = store.GetMarkedRepos("owner2")
	require.NoError(t, err)
	assert.Len(t, marks, 1)
	assert.Contains(t, marks, "repo3")
}

func TestClearMarkedRepos_RemovesAllMarksForOwner(t *testing.T) {
	store := setupTestStore(t)
	owner := "testowner"

	// Save marks
	err := store.SaveMarkedRepos(owner, []string{"repo1", "repo2"})
	require.NoError(t, err)

	// Clear marks
	err = store.ClearMarkedRepos(owner)
	require.NoError(t, err)

	// Verify marks are gone
	marks, err := store.GetMarkedRepos(owner)
	require.NoError(t, err)
	assert.Empty(t, marks)
}

func TestClearMarkedRepos_DoesNotAffectOtherOwners(t *testing.T) {
	store := setupTestStore(t)

	// Save marks for two owners
	err := store.SaveMarkedRepos("owner1", []string{"repo1"})
	require.NoError(t, err)

	err = store.SaveMarkedRepos("owner2", []string{"repo2"})
	require.NoError(t, err)

	// Clear marks for owner1
	err = store.ClearMarkedRepos("owner1")
	require.NoError(t, err)

	// owner1 marks should be gone
	marks, err := store.GetMarkedRepos("owner1")
	require.NoError(t, err)
	assert.Empty(t, marks)

	// owner2 marks should still exist
	marks, err = store.GetMarkedRepos("owner2")
	require.NoError(t, err)
	assert.Len(t, marks, 1)
}

func TestAddMarkedRepo_AddsNewMark(t *testing.T) {
	store := setupTestStore(t)
	owner := "testowner"

	err := store.AddMarkedRepo(owner, "repo1")
	require.NoError(t, err)

	marks, err := store.GetMarkedRepos(owner)
	require.NoError(t, err)
	assert.Len(t, marks, 1)
	assert.Contains(t, marks, "repo1")
}

func TestAddMarkedRepo_IgnoresDuplicates(t *testing.T) {
	store := setupTestStore(t)
	owner := "testowner"

	// Add same repo twice
	err := store.AddMarkedRepo(owner, "repo1")
	require.NoError(t, err)

	err = store.AddMarkedRepo(owner, "repo1")
	require.NoError(t, err) // Should not error

	// Should only have one mark
	marks, err := store.GetMarkedRepos(owner)
	require.NoError(t, err)
	assert.Len(t, marks, 1)
}

func TestRemoveMarkedRepo_RemovesMark(t *testing.T) {
	store := setupTestStore(t)
	owner := "testowner"

	// Add marks
	err := store.AddMarkedRepo(owner, "repo1")
	require.NoError(t, err)
	err = store.AddMarkedRepo(owner, "repo2")
	require.NoError(t, err)

	// Remove one
	err = store.RemoveMarkedRepo(owner, "repo1")
	require.NoError(t, err)

	// Verify
	marks, err := store.GetMarkedRepos(owner)
	require.NoError(t, err)
	assert.Len(t, marks, 1)
	assert.NotContains(t, marks, "repo1")
	assert.Contains(t, marks, "repo2")
}

func TestRemoveMarkedRepo_NoErrorForNonexistent(t *testing.T) {
	store := setupTestStore(t)

	// Remove nonexistent mark - should not error
	err := store.RemoveMarkedRepo("owner", "nonexistent")
	require.NoError(t, err)
}

// Tests for repo changes audit functionality

func TestRecordRepoChange_StoresAllFieldsCorrectly(t *testing.T) {
	store := setupTestStore(t)
	owner := "testowner"

	prevState := map[string]bool{"is_archived": false}
	newState := map[string]bool{"is_archived": true}

	err := store.RecordRepoChange(owner, "myrepo", "archived", "user", prevState, newState, "Archiving old repo")
	require.NoError(t, err)

	changes, err := store.GetRepoHistory(owner, "myrepo", 10)
	require.NoError(t, err)
	require.Len(t, changes, 1)

	change := changes[0]
	assert.Equal(t, owner, change.Owner)
	assert.Equal(t, "myrepo", change.RepoName)
	assert.Equal(t, "archived", change.Action)
	assert.Equal(t, "user", change.PerformedBy)
	assert.Equal(t, "Archiving old repo", change.Notes)
	assert.False(t, change.PerformedAt.IsZero())
	assert.Greater(t, change.ID, int64(0))
}

func TestRecordRepoChange_SerializesJSON(t *testing.T) {
	store := setupTestStore(t)
	owner := "testowner"

	prevState := map[string]interface{}{
		"stars":       42,
		"is_archived": false,
		"name":        "myrepo",
	}
	newState := map[string]interface{}{
		"stars":       42,
		"is_archived": true,
		"name":        "myrepo",
	}

	err := store.RecordRepoChange(owner, "myrepo", "archived", "system", prevState, newState, "")
	require.NoError(t, err)

	changes, err := store.GetRepoHistory(owner, "myrepo", 10)
	require.NoError(t, err)
	require.Len(t, changes, 1)

	// Verify JSON was stored correctly
	assert.Contains(t, changes[0].PreviousState, `"is_archived":false`)
	assert.Contains(t, changes[0].NewState, `"is_archived":true`)
	assert.Contains(t, changes[0].PreviousState, `"stars":42`)
}

func TestRecordRepoChange_HandlesNilStates(t *testing.T) {
	store := setupTestStore(t)
	owner := "testowner"

	err := store.RecordRepoChange(owner, "myrepo", "deleted", "user", nil, nil, "Cleanup")
	require.NoError(t, err)

	changes, err := store.GetRepoHistory(owner, "myrepo", 10)
	require.NoError(t, err)
	require.Len(t, changes, 1)

	assert.Empty(t, changes[0].PreviousState)
	assert.Empty(t, changes[0].NewState)
	assert.Equal(t, "Cleanup", changes[0].Notes)
}

func TestRecordRepoChange_HandlesEmptyNotes(t *testing.T) {
	store := setupTestStore(t)
	owner := "testowner"

	err := store.RecordRepoChange(owner, "myrepo", "marked", "user", nil, nil, "")
	require.NoError(t, err)

	changes, err := store.GetRepoHistory(owner, "myrepo", 10)
	require.NoError(t, err)
	require.Len(t, changes, 1)

	assert.Empty(t, changes[0].Notes)
}

func TestGetRepoHistory_FiltersByOwnerAndRepoName(t *testing.T) {
	store := setupTestStore(t)

	// Record changes for different repos and owners
	err := store.RecordRepoChange("owner1", "repo1", "marked", "user", nil, nil, "")
	require.NoError(t, err)

	err = store.RecordRepoChange("owner1", "repo2", "archived", "user", nil, nil, "")
	require.NoError(t, err)

	err = store.RecordRepoChange("owner2", "repo1", "deleted", "user", nil, nil, "")
	require.NoError(t, err)

	// Get history for owner1/repo1 only
	changes, err := store.GetRepoHistory("owner1", "repo1", 10)
	require.NoError(t, err)
	require.Len(t, changes, 1)
	assert.Equal(t, "owner1", changes[0].Owner)
	assert.Equal(t, "repo1", changes[0].RepoName)
	assert.Equal(t, "marked", changes[0].Action)
}

func TestGetRepoHistory_RespectsLimit(t *testing.T) {
	store := setupTestStore(t)
	owner := "testowner"

	// Record multiple changes for the same repo
	for i := 0; i < 5; i++ {
		err := store.RecordRepoChange(owner, "myrepo", "synced", "sync", nil, nil, "")
		require.NoError(t, err)
	}

	// Get with limit 3
	changes, err := store.GetRepoHistory(owner, "myrepo", 3)
	require.NoError(t, err)
	assert.Len(t, changes, 3)
}

func TestGetRepoHistory_OrderedByPerformedAtDesc(t *testing.T) {
	store := setupTestStore(t)
	owner := "testowner"

	// Record changes with different actions to identify order
	err := store.RecordRepoChange(owner, "myrepo", "first", "user", nil, nil, "")
	require.NoError(t, err)

	err = store.RecordRepoChange(owner, "myrepo", "second", "user", nil, nil, "")
	require.NoError(t, err)

	err = store.RecordRepoChange(owner, "myrepo", "third", "user", nil, nil, "")
	require.NoError(t, err)

	changes, err := store.GetRepoHistory(owner, "myrepo", 10)
	require.NoError(t, err)
	require.Len(t, changes, 3)

	// Higher IDs indicate later inserts; verify ordering by ID is DESC
	// This is a proxy for performed_at DESC since all records have same second-level timestamp
	assert.Greater(t, changes[0].ID, changes[1].ID)
	assert.Greater(t, changes[1].ID, changes[2].ID)
}

func TestGetRepoHistory_EmptyForUnknownRepo(t *testing.T) {
	store := setupTestStore(t)

	changes, err := store.GetRepoHistory("unknown", "nonexistent", 10)
	require.NoError(t, err)
	assert.Empty(t, changes)
}

func TestGetRecentChanges_ReturnsAllReposForOwner(t *testing.T) {
	store := setupTestStore(t)
	owner := "testowner"

	// Record changes for multiple repos under the same owner
	err := store.RecordRepoChange(owner, "repo1", "marked", "user", nil, nil, "")
	require.NoError(t, err)

	err = store.RecordRepoChange(owner, "repo2", "archived", "system", nil, nil, "")
	require.NoError(t, err)

	err = store.RecordRepoChange(owner, "repo3", "deleted", "user", nil, nil, "")
	require.NoError(t, err)

	// Should get all 3 changes
	changes, err := store.GetRecentChanges(owner, 10)
	require.NoError(t, err)
	assert.Len(t, changes, 3)

	// Collect all repo names
	repoNames := make(map[string]bool)
	for _, c := range changes {
		repoNames[c.RepoName] = true
	}
	assert.True(t, repoNames["repo1"])
	assert.True(t, repoNames["repo2"])
	assert.True(t, repoNames["repo3"])
}

func TestGetRecentChanges_FiltersByOwner(t *testing.T) {
	store := setupTestStore(t)

	// Record changes for different owners
	err := store.RecordRepoChange("owner1", "repo1", "marked", "user", nil, nil, "")
	require.NoError(t, err)

	err = store.RecordRepoChange("owner2", "repo2", "archived", "user", nil, nil, "")
	require.NoError(t, err)

	// Get changes for owner1 only
	changes, err := store.GetRecentChanges("owner1", 10)
	require.NoError(t, err)
	require.Len(t, changes, 1)
	assert.Equal(t, "owner1", changes[0].Owner)
}

func TestGetRecentChanges_RespectsLimit(t *testing.T) {
	store := setupTestStore(t)
	owner := "testowner"

	// Record multiple changes
	for i := 0; i < 10; i++ {
		err := store.RecordRepoChange(owner, "repo1", "synced", "sync", nil, nil, "")
		require.NoError(t, err)
	}

	// Get with limit 5
	changes, err := store.GetRecentChanges(owner, 5)
	require.NoError(t, err)
	assert.Len(t, changes, 5)
}

func TestGetRecentChanges_OrderedByPerformedAtDesc(t *testing.T) {
	store := setupTestStore(t)
	owner := "testowner"

	// Record changes with different repos to identify order
	err := store.RecordRepoChange(owner, "first", "synced", "sync", nil, nil, "")
	require.NoError(t, err)

	err = store.RecordRepoChange(owner, "second", "synced", "sync", nil, nil, "")
	require.NoError(t, err)

	err = store.RecordRepoChange(owner, "third", "synced", "sync", nil, nil, "")
	require.NoError(t, err)

	changes, err := store.GetRecentChanges(owner, 10)
	require.NoError(t, err)
	require.Len(t, changes, 3)

	// Higher IDs indicate later inserts; verify ordering by ID is DESC
	// This is a proxy for performed_at DESC since all records have same second-level timestamp
	assert.Greater(t, changes[0].ID, changes[1].ID)
	assert.Greater(t, changes[1].ID, changes[2].ID)
}

func TestGetRecentChanges_EmptyForUnknownOwner(t *testing.T) {
	store := setupTestStore(t)

	changes, err := store.GetRecentChanges("unknown", 10)
	require.NoError(t, err)
	assert.Empty(t, changes)
}

func TestGetChangesByAction_FiltersCorrectly(t *testing.T) {
	store := setupTestStore(t)
	owner := "testowner"

	// Record changes with different actions
	err := store.RecordRepoChange(owner, "repo1", "archived", "user", nil, nil, "")
	require.NoError(t, err)

	err = store.RecordRepoChange(owner, "repo2", "marked", "user", nil, nil, "")
	require.NoError(t, err)

	err = store.RecordRepoChange(owner, "repo3", "archived", "system", nil, nil, "")
	require.NoError(t, err)

	err = store.RecordRepoChange(owner, "repo4", "deleted", "user", nil, nil, "")
	require.NoError(t, err)

	// Get only "archived" actions
	changes, err := store.GetChangesByAction(owner, "archived", 10)
	require.NoError(t, err)
	require.Len(t, changes, 2)

	for _, c := range changes {
		assert.Equal(t, "archived", c.Action)
	}
}

func TestGetChangesByAction_FiltersByOwnerAndAction(t *testing.T) {
	store := setupTestStore(t)

	// Record changes for different owners with same action
	err := store.RecordRepoChange("owner1", "repo1", "archived", "user", nil, nil, "")
	require.NoError(t, err)

	err = store.RecordRepoChange("owner2", "repo2", "archived", "user", nil, nil, "")
	require.NoError(t, err)

	// Get archived changes for owner1 only
	changes, err := store.GetChangesByAction("owner1", "archived", 10)
	require.NoError(t, err)
	require.Len(t, changes, 1)
	assert.Equal(t, "owner1", changes[0].Owner)
}

func TestGetChangesByAction_RespectsLimit(t *testing.T) {
	store := setupTestStore(t)
	owner := "testowner"

	// Record multiple changes with same action
	for i := 0; i < 10; i++ {
		err := store.RecordRepoChange(owner, "repo1", "synced", "sync", nil, nil, "")
		require.NoError(t, err)
	}

	// Get with limit 3
	changes, err := store.GetChangesByAction(owner, "synced", 3)
	require.NoError(t, err)
	assert.Len(t, changes, 3)
}

func TestGetChangesByAction_OrderedByPerformedAtDesc(t *testing.T) {
	store := setupTestStore(t)
	owner := "testowner"

	// Record changes with same action but different repos
	err := store.RecordRepoChange(owner, "first", "archived", "user", nil, nil, "")
	require.NoError(t, err)

	err = store.RecordRepoChange(owner, "second", "archived", "user", nil, nil, "")
	require.NoError(t, err)

	err = store.RecordRepoChange(owner, "third", "archived", "user", nil, nil, "")
	require.NoError(t, err)

	changes, err := store.GetChangesByAction(owner, "archived", 10)
	require.NoError(t, err)
	require.Len(t, changes, 3)

	// Higher IDs indicate later inserts; verify ordering by ID is DESC
	// This is a proxy for performed_at DESC since all records have same second-level timestamp
	assert.Greater(t, changes[0].ID, changes[1].ID)
	assert.Greater(t, changes[1].ID, changes[2].ID)
}

func TestGetChangesByAction_EmptyForNonexistentAction(t *testing.T) {
	store := setupTestStore(t)
	owner := "testowner"

	// Record a change with one action
	err := store.RecordRepoChange(owner, "repo1", "marked", "user", nil, nil, "")
	require.NoError(t, err)

	// Query for a different action
	changes, err := store.GetChangesByAction(owner, "archived", 10)
	require.NoError(t, err)
	assert.Empty(t, changes)
}

func TestRecordRepoChange_SerializesComplexStructs(t *testing.T) {
	store := setupTestStore(t)
	owner := "testowner"

	// Use a custom struct type as prev/new state
	type repoState struct {
		Name       string `json:"name"`
		Stars      int    `json:"stars"`
		IsArchived bool   `json:"is_archived"`
		Language   string `json:"language"`
	}

	prevState := repoState{
		Name:       "myrepo",
		Stars:      10,
		IsArchived: false,
		Language:   "Go",
	}
	newState := repoState{
		Name:       "myrepo",
		Stars:      10,
		IsArchived: true,
		Language:   "Go",
	}

	err := store.RecordRepoChange(owner, "myrepo", "archived", "user", prevState, newState, "Archiving inactive repo")
	require.NoError(t, err)

	changes, err := store.GetRepoHistory(owner, "myrepo", 10)
	require.NoError(t, err)
	require.Len(t, changes, 1)

	// Verify JSON contains expected data
	assert.Contains(t, changes[0].PreviousState, `"is_archived":false`)
	assert.Contains(t, changes[0].NewState, `"is_archived":true`)
	assert.Contains(t, changes[0].PreviousState, `"language":"Go"`)
}

// Tests for sync history functionality

func TestRecordSyncStart_CreatesRecordWithRunningStatus(t *testing.T) {
	store := setupTestStore(t)
	owner := "testowner"

	syncID, err := store.RecordSyncStart(owner)
	require.NoError(t, err)
	assert.Greater(t, syncID, int64(0))

	// Verify record was created with running status
	history, err := store.GetSyncHistory(owner, 10)
	require.NoError(t, err)
	require.Len(t, history, 1)

	record := history[0]
	assert.Equal(t, syncID, record.ID)
	assert.Equal(t, owner, record.Owner)
	assert.Equal(t, "running", record.Status)
	assert.Nil(t, record.CompletedAt)
	assert.WithinDuration(t, time.Now(), record.StartedAt, 1*time.Second)
}

func TestRecordSyncComplete_UpdatesAllFieldsCorrectly(t *testing.T) {
	store := setupTestStore(t)
	owner := "testowner"

	// Start a sync
	syncID, err := store.RecordSyncStart(owner)
	require.NoError(t, err)

	// Small delay to ensure duration is measurable
	time.Sleep(10 * time.Millisecond)

	// Complete the sync
	err = store.RecordSyncComplete(syncID, "success", 100, 50, 30, "")
	require.NoError(t, err)

	// Verify all fields were updated
	history, err := store.GetSyncHistory(owner, 10)
	require.NoError(t, err)
	require.Len(t, history, 1)

	record := history[0]
	assert.Equal(t, syncID, record.ID)
	assert.Equal(t, "success", record.Status)
	assert.Equal(t, 100, record.ReposFetched)
	assert.Equal(t, 50, record.ReposInserted)
	assert.Equal(t, 30, record.ReposUpdated)
	assert.Empty(t, record.ErrorMessage)
	assert.NotNil(t, record.CompletedAt)
	assert.Greater(t, record.DurationMs, int64(0))
}

func TestRecordSyncComplete_StoresErrorMessage(t *testing.T) {
	store := setupTestStore(t)
	owner := "testowner"

	syncID, err := store.RecordSyncStart(owner)
	require.NoError(t, err)

	errMsg := "API rate limit exceeded"
	err = store.RecordSyncComplete(syncID, "error", 10, 0, 0, errMsg)
	require.NoError(t, err)

	history, err := store.GetSyncHistory(owner, 10)
	require.NoError(t, err)
	require.Len(t, history, 1)

	assert.Equal(t, "error", history[0].Status)
	assert.Equal(t, errMsg, history[0].ErrorMessage)
}

func TestRecordSyncComplete_NotFoundForInvalidID(t *testing.T) {
	store := setupTestStore(t)

	err := store.RecordSyncComplete(999999, "success", 10, 5, 3, "")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestGetSyncHistory_ReturnsRecordsInDescendingOrderByStartedAt(t *testing.T) {
	store := setupTestStore(t)
	owner := "testowner"

	// Create multiple sync records with small delays to ensure ordering
	syncID1, err := store.RecordSyncStart(owner)
	require.NoError(t, err)
	time.Sleep(10 * time.Millisecond)

	syncID2, err := store.RecordSyncStart(owner)
	require.NoError(t, err)
	time.Sleep(10 * time.Millisecond)

	syncID3, err := store.RecordSyncStart(owner)
	require.NoError(t, err)

	// Get history
	history, err := store.GetSyncHistory(owner, 10)
	require.NoError(t, err)
	require.Len(t, history, 3)

	// Verify descending order (most recent first)
	assert.Equal(t, syncID3, history[0].ID)
	assert.Equal(t, syncID2, history[1].ID)
	assert.Equal(t, syncID1, history[2].ID)

	// Verify started_at times are in descending order
	assert.True(t, history[0].StartedAt.After(history[1].StartedAt) || history[0].StartedAt.Equal(history[1].StartedAt))
	assert.True(t, history[1].StartedAt.After(history[2].StartedAt) || history[1].StartedAt.Equal(history[2].StartedAt))
}

func TestGetSyncHistory_RespectsLimit(t *testing.T) {
	store := setupTestStore(t)
	owner := "testowner"

	// Create 5 sync records
	for i := 0; i < 5; i++ {
		_, err := store.RecordSyncStart(owner)
		require.NoError(t, err)
	}

	// Get only 3
	history, err := store.GetSyncHistory(owner, 3)
	require.NoError(t, err)
	assert.Len(t, history, 3)
}

func TestGetSyncHistory_FiltersByOwner(t *testing.T) {
	store := setupTestStore(t)

	// Create records for two owners
	_, err := store.RecordSyncStart("owner1")
	require.NoError(t, err)
	_, err = store.RecordSyncStart("owner1")
	require.NoError(t, err)

	_, err = store.RecordSyncStart("owner2")
	require.NoError(t, err)

	// Get history for owner1
	history, err := store.GetSyncHistory("owner1", 10)
	require.NoError(t, err)
	assert.Len(t, history, 2)
	for _, record := range history {
		assert.Equal(t, "owner1", record.Owner)
	}

	// Get history for owner2
	history, err = store.GetSyncHistory("owner2", 10)
	require.NoError(t, err)
	assert.Len(t, history, 1)
	assert.Equal(t, "owner2", history[0].Owner)
}

func TestGetSyncHistory_ReturnsEmptyForUnknownOwner(t *testing.T) {
	store := setupTestStore(t)

	history, err := store.GetSyncHistory("nonexistent", 10)
	require.NoError(t, err)
	assert.Empty(t, history)
}

func TestGetLastSuccessfulSync_ReturnsOnlySuccessStatusRecords(t *testing.T) {
	store := setupTestStore(t)
	owner := "testowner"

	// Create records with different statuses
	syncID1, err := store.RecordSyncStart(owner)
	require.NoError(t, err)
	err = store.RecordSyncComplete(syncID1, "error", 10, 0, 0, "failed")
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond)

	syncID2, err := store.RecordSyncStart(owner)
	require.NoError(t, err)
	err = store.RecordSyncComplete(syncID2, "success", 50, 25, 10, "")
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond)

	syncID3, err := store.RecordSyncStart(owner)
	require.NoError(t, err)
	err = store.RecordSyncComplete(syncID3, "partial", 30, 15, 5, "incomplete")
	require.NoError(t, err)

	// Get last successful sync
	record, err := store.GetLastSuccessfulSync(owner)
	require.NoError(t, err)
	require.NotNil(t, record)

	// Should return the success record, not the more recent partial or error
	assert.Equal(t, syncID2, record.ID)
	assert.Equal(t, "success", record.Status)
	assert.Equal(t, 50, record.ReposFetched)
}

func TestGetLastSuccessfulSync_ReturnsNilWhenNoSuccessfulSyncsExist(t *testing.T) {
	store := setupTestStore(t)
	owner := "testowner"

	// Create only error and partial records
	syncID1, err := store.RecordSyncStart(owner)
	require.NoError(t, err)
	err = store.RecordSyncComplete(syncID1, "error", 10, 0, 0, "failed")
	require.NoError(t, err)

	syncID2, err := store.RecordSyncStart(owner)
	require.NoError(t, err)
	err = store.RecordSyncComplete(syncID2, "partial", 30, 15, 5, "incomplete")
	require.NoError(t, err)

	// Get last successful sync
	record, err := store.GetLastSuccessfulSync(owner)
	require.NoError(t, err)
	assert.Nil(t, record)
}

func TestGetLastSuccessfulSync_ReturnsNilForNoSyncs(t *testing.T) {
	store := setupTestStore(t)

	record, err := store.GetLastSuccessfulSync("nonexistent")
	require.NoError(t, err)
	assert.Nil(t, record)
}

func TestGetLastSuccessfulSync_ReturnsMostRecentSuccess(t *testing.T) {
	store := setupTestStore(t)
	owner := "testowner"

	// Create two successful syncs
	syncID1, err := store.RecordSyncStart(owner)
	require.NoError(t, err)
	err = store.RecordSyncComplete(syncID1, "success", 50, 25, 10, "")
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond)

	syncID2, err := store.RecordSyncStart(owner)
	require.NoError(t, err)
	err = store.RecordSyncComplete(syncID2, "success", 60, 30, 15, "")
	require.NoError(t, err)

	// Get last successful sync - should be the more recent one
	record, err := store.GetLastSuccessfulSync(owner)
	require.NoError(t, err)
	require.NotNil(t, record)

	assert.Equal(t, syncID2, record.ID)
	assert.Equal(t, 60, record.ReposFetched)
}

func TestGetLastSuccessfulSync_FiltersByOwner(t *testing.T) {
	store := setupTestStore(t)

	// Create successful sync for owner1
	syncID1, err := store.RecordSyncStart("owner1")
	require.NoError(t, err)
	err = store.RecordSyncComplete(syncID1, "success", 50, 25, 10, "")
	require.NoError(t, err)

	// Create successful sync for owner2
	syncID2, err := store.RecordSyncStart("owner2")
	require.NoError(t, err)
	err = store.RecordSyncComplete(syncID2, "success", 100, 50, 20, "")
	require.NoError(t, err)

	// Get last successful for owner1
	record, err := store.GetLastSuccessfulSync("owner1")
	require.NoError(t, err)
	require.NotNil(t, record)
	assert.Equal(t, syncID1, record.ID)
	assert.Equal(t, "owner1", record.Owner)
	assert.Equal(t, 50, record.ReposFetched)

	// Get last successful for owner2
	record, err = store.GetLastSuccessfulSync("owner2")
	require.NoError(t, err)
	require.NotNil(t, record)
	assert.Equal(t, syncID2, record.ID)
	assert.Equal(t, "owner2", record.Owner)
	assert.Equal(t, 100, record.ReposFetched)
}
