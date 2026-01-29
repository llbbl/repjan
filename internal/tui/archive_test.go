// SPDX-FileCopyrightText: 2026 Logan Lindquist Land
// SPDX-License-Identifier: FSL-1.1-MIT

package tui

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/llbbl/repjan/internal/github"
	"github.com/llbbl/repjan/internal/testutil"
)

// TestArchiveMarkedRepos_NoMarkedRepos verifies that archiveMarkedRepos returns nil
// when no repositories are marked for archiving.
func TestArchiveMarkedRepos_NoMarkedRepos(t *testing.T) {
	m := &Model{
		repos:  []github.Repository{testutil.NewTestRepo()},
		marked: make(map[string]bool), // Empty marked map
	}

	cmd := m.archiveMarkedRepos()
	assert.Nil(t, cmd, "expected nil command when no repos are marked")
}

// TestArchiveMarkedRepos_InitializesState verifies that archiveMarkedRepos correctly
// initializes the archive state when repos are marked.
func TestArchiveMarkedRepos_InitializesState(t *testing.T) {
	repo1 := testutil.NewTestRepo(testutil.WithOwner("owner1"), testutil.WithName("repo1"))
	repo2 := testutil.NewTestRepo(testutil.WithOwner("owner2"), testutil.WithName("repo2"))

	mockExec := testutil.NewMockExecutor()
	mockExec.ExecuteFunc = func(name string, args ...string) ([]byte, error) {
		return nil, nil // Success
	}
	client := github.NewClient(mockExec)

	m := &Model{
		repos:  []github.Repository{repo1, repo2},
		marked: map[string]bool{"owner1/repo1": true, "owner2/repo2": true},
		client: client,
	}

	cmd := m.archiveMarkedRepos()

	// Verify state is initialized
	assert.True(t, m.archiving, "archiving should be true")
	assert.Equal(t, 2, m.archiveTotal, "archiveTotal should be 2")
	assert.Equal(t, 0, m.archiveProgress, "archiveProgress should be 0")
	assert.NotNil(t, m.archiveState, "archiveState should not be nil")
	assert.Len(t, m.archiveState.repos, 2, "archiveState.repos should have 2 repos")
	assert.NotNil(t, cmd, "expected a command to be returned")
}

// TestArchiveNextRepo_Success verifies that archiveNextRepo archives a repository
// and sends a progress message on success.
func TestArchiveNextRepo_Success(t *testing.T) {
	repo := testutil.NewTestRepo(testutil.WithOwner("testowner"), testutil.WithName("testrepo"))

	mockExec := testutil.NewMockExecutor()
	mockExec.ExecuteFunc = func(name string, args ...string) ([]byte, error) {
		return nil, nil // Success
	}
	client := github.NewClient(mockExec)

	state := &archiveState{
		repos:     []github.Repository{repo},
		succeeded: 0,
		failed:    0,
		errors:    nil,
	}

	cmd := archiveNextRepo(client, state.repos, 0, state)
	require.NotNil(t, cmd, "expected a command")

	// Execute the command
	msg := cmd()

	// Verify progress message
	progressMsg, ok := msg.(ArchiveProgressMsg)
	require.True(t, ok, "expected ArchiveProgressMsg, got %T", msg)
	assert.Equal(t, 1, progressMsg.Current, "progress current should be 1")
	assert.Equal(t, 1, progressMsg.Total, "progress total should be 1")
	assert.Equal(t, "testowner/testrepo", progressMsg.RepoName)
	assert.Nil(t, progressMsg.Err, "expected no error")

	// Verify state was updated
	assert.Equal(t, 1, state.succeeded, "succeeded should be 1")
	assert.Equal(t, 0, state.failed, "failed should be 0")

	// Verify the executor was called with correct args
	assert.Equal(t, 1, mockExec.CallCount(), "expected 1 call to executor")
	call := mockExec.GetCall(0)
	assert.Equal(t, []string{"gh", "repo", "archive", "testowner/testrepo", "--yes"}, call)
}

// TestArchiveNextRepo_Error verifies that archiveNextRepo handles errors
// and continues to track them in state.
func TestArchiveNextRepo_Error(t *testing.T) {
	repo := testutil.NewTestRepo(testutil.WithOwner("testowner"), testutil.WithName("testrepo"))

	archiveErr := errors.New("archive failed")
	mockExec := testutil.NewMockExecutor()
	mockExec.ExecuteFunc = func(name string, args ...string) ([]byte, error) {
		return []byte("error output"), archiveErr
	}
	client := github.NewClient(mockExec)

	state := &archiveState{
		repos:     []github.Repository{repo},
		succeeded: 0,
		failed:    0,
		errors:    nil,
	}

	cmd := archiveNextRepo(client, state.repos, 0, state)
	require.NotNil(t, cmd, "expected a command")

	// Execute the command
	msg := cmd()

	// Verify progress message has error
	progressMsg, ok := msg.(ArchiveProgressMsg)
	require.True(t, ok, "expected ArchiveProgressMsg, got %T", msg)
	assert.Equal(t, 1, progressMsg.Current)
	assert.Equal(t, 1, progressMsg.Total)
	assert.Equal(t, "testowner/testrepo", progressMsg.RepoName)
	assert.NotNil(t, progressMsg.Err, "expected error in progress message")

	// Verify state was updated with failure
	assert.Equal(t, 0, state.succeeded, "succeeded should be 0")
	assert.Equal(t, 1, state.failed, "failed should be 1")
	assert.Len(t, state.errors, 1, "should have 1 error recorded")
}

// TestArchiveComplete_Counts verifies that ArchiveCompleteMsg has correct
// success/failure counts after archiving multiple repos.
func TestArchiveComplete_Counts(t *testing.T) {
	repo1 := testutil.NewTestRepo(testutil.WithOwner("owner1"), testutil.WithName("repo1"))
	repo2 := testutil.NewTestRepo(testutil.WithOwner("owner2"), testutil.WithName("repo2"))
	repo3 := testutil.NewTestRepo(testutil.WithOwner("owner3"), testutil.WithName("repo3"))

	callCount := 0
	mockExec := testutil.NewMockExecutor()
	mockExec.ExecuteFunc = func(name string, args ...string) ([]byte, error) {
		callCount++
		// First and third succeed, second fails
		if callCount == 2 {
			return nil, errors.New("archive failed")
		}
		return nil, nil
	}
	client := github.NewClient(mockExec)

	repos := []github.Repository{repo1, repo2, repo3}
	state := &archiveState{
		repos:     repos,
		succeeded: 0,
		failed:    0,
		errors:    nil,
	}

	// Archive all three repos
	for i := 0; i < len(repos); i++ {
		cmd := archiveNextRepo(client, repos, i, state)
		cmd() // Execute the command
	}

	// Now get the complete message (when current >= len(repos))
	cmd := archiveNextRepo(client, repos, 3, state)
	msg := cmd()

	completeMsg, ok := msg.(ArchiveCompleteMsg)
	require.True(t, ok, "expected ArchiveCompleteMsg, got %T", msg)
	assert.Equal(t, 2, completeMsg.Succeeded, "expected 2 successful")
	assert.Equal(t, 1, completeMsg.Failed, "expected 1 failed")
	assert.Len(t, completeMsg.Errors, 1, "expected 1 error in list")
}

// TestMarkRepoAsArchived verifies that markRepoAsArchived correctly updates
// the repo's IsArchived field in the model.
func TestMarkRepoAsArchived(t *testing.T) {
	repo1 := testutil.NewTestRepo(testutil.WithOwner("owner1"), testutil.WithName("repo1"))
	repo2 := testutil.NewTestRepo(testutil.WithOwner("owner2"), testutil.WithName("repo2"))

	m := &Model{
		repos: []github.Repository{repo1, repo2},
	}

	// Verify initial state
	assert.False(t, m.repos[0].IsArchived)
	assert.False(t, m.repos[1].IsArchived)

	// Mark first repo as archived
	m.markRepoAsArchived("owner1/repo1")

	// Verify only first repo is marked
	assert.True(t, m.repos[0].IsArchived, "repo1 should be archived")
	assert.False(t, m.repos[1].IsArchived, "repo2 should not be archived")

	// Mark second repo as archived
	m.markRepoAsArchived("owner2/repo2")

	// Verify both are now archived
	assert.True(t, m.repos[0].IsArchived)
	assert.True(t, m.repos[1].IsArchived)
}

// TestMarkRepoAsArchived_NonexistentRepo verifies that markRepoAsArchived
// handles nonexistent repos gracefully (no panic).
func TestMarkRepoAsArchived_NonexistentRepo(t *testing.T) {
	repo := testutil.NewTestRepo(testutil.WithOwner("owner"), testutil.WithName("repo"))
	m := &Model{repos: []github.Repository{repo}}

	// Should not panic
	m.markRepoAsArchived("nonexistent/repo")

	// Original repo should be unchanged
	assert.False(t, m.repos[0].IsArchived)
}

// TestClearArchivedMarks verifies that clearArchivedMarks removes marks
// from repos that have been archived.
func TestClearArchivedMarks(t *testing.T) {
	repo1 := testutil.NewTestRepo(testutil.WithOwner("owner1"), testutil.WithName("repo1"), testutil.WithArchived(true))
	repo2 := testutil.NewTestRepo(testutil.WithOwner("owner2"), testutil.WithName("repo2"), testutil.WithArchived(false))
	repo3 := testutil.NewTestRepo(testutil.WithOwner("owner3"), testutil.WithName("repo3"), testutil.WithArchived(true))

	m := &Model{
		repos: []github.Repository{repo1, repo2, repo3},
		marked: map[string]bool{
			"owner1/repo1": true, // archived - should be cleared
			"owner2/repo2": true, // not archived - should remain
			"owner3/repo3": true, // archived - should be cleared
		},
	}

	m.clearArchivedMarks()

	// Only the non-archived repo should still be marked
	assert.False(t, m.marked["owner1/repo1"], "archived repo1 mark should be cleared")
	assert.True(t, m.marked["owner2/repo2"], "non-archived repo2 mark should remain")
	assert.False(t, m.marked["owner3/repo3"], "archived repo3 mark should be cleared")
	assert.Len(t, m.marked, 1, "only 1 mark should remain")
}

// TestClearArchivedMarks_NoMarks verifies clearArchivedMarks handles empty
// marked map gracefully.
func TestClearArchivedMarks_NoMarks(t *testing.T) {
	repo := testutil.NewTestRepo(testutil.WithArchived(true))
	m := &Model{
		repos:  []github.Repository{repo},
		marked: make(map[string]bool),
	}

	// Should not panic
	m.clearArchivedMarks()
	assert.Empty(t, m.marked)
}

// TestArchiveProgressMsg_UpdatesModel verifies that the Update function
// correctly handles ArchiveProgressMsg.
func TestArchiveProgressMsg_UpdatesModel(t *testing.T) {
	repo := testutil.NewTestRepo(testutil.WithOwner("owner"), testutil.WithName("repo"))
	m := Model{
		repos:           []github.Repository{repo},
		marked:          map[string]bool{"owner/repo": true},
		archiveProgress: 0,
		archiveTotal:    1,
		archiveState: &archiveState{
			repos:     []github.Repository{repo},
			succeeded: 1,
			failed:    0,
		},
		styles: DefaultStyles(),
	}

	// Send a successful progress message
	progressMsg := ArchiveProgressMsg{
		Current:  1,
		Total:    1,
		RepoName: "owner/repo",
		Err:      nil,
	}

	newModel, _ := m.Update(progressMsg)
	updated := newModel.(Model)

	// Verify progress was updated
	assert.Equal(t, 1, updated.archiveProgress)
	assert.Equal(t, 1, updated.archiveTotal)

	// Verify repo was marked as archived
	assert.True(t, updated.repos[0].IsArchived)
}

// TestArchiveCompleteMsg_UpdatesModel verifies that the Update function
// correctly handles ArchiveCompleteMsg.
func TestArchiveCompleteMsg_UpdatesModel(t *testing.T) {
	repo := testutil.NewTestRepo(testutil.WithOwner("owner"), testutil.WithName("repo"), testutil.WithArchived(true))
	m := Model{
		repos:           []github.Repository{repo},
		marked:          map[string]bool{"owner/repo": true},
		archiving:       true,
		archiveProgress: 1,
		archiveTotal:    1,
		archiveState: &archiveState{
			repos:     []github.Repository{repo},
			succeeded: 1,
			failed:    0,
		},
		styles: DefaultStyles(),
	}

	completeMsg := ArchiveCompleteMsg{
		Succeeded: 1,
		Failed:    0,
		Errors:    nil,
	}

	newModel, _ := m.Update(completeMsg)
	updated := newModel.(Model)

	// Verify archiving state was reset
	assert.False(t, updated.archiving)
	assert.Equal(t, 0, updated.archiveProgress)
	assert.Equal(t, 0, updated.archiveTotal)
	assert.Nil(t, updated.archiveState)

	// Verify mark was cleared (repo is archived)
	assert.Empty(t, updated.marked)

	// Verify status message
	assert.Equal(t, "Successfully archived 1 repo", updated.statusMessage)
}

// TestArchiveCompleteMsg_WithFailures verifies the status message when
// some archives fail.
func TestArchiveCompleteMsg_WithFailures(t *testing.T) {
	m := Model{
		repos:     []github.Repository{},
		marked:    make(map[string]bool),
		archiving: true,
		archiveState: &archiveState{
			repos:     []github.Repository{},
			succeeded: 2,
			failed:    1,
		},
		styles: DefaultStyles(),
	}

	completeMsg := ArchiveCompleteMsg{
		Succeeded: 2,
		Failed:    1,
		Errors:    []error{errors.New("test error")},
	}

	newModel, _ := m.Update(completeMsg)
	updated := newModel.(Model)

	// Verify status message includes failure count
	assert.Equal(t, "Archive completed: 2 succeeded, 1 failed", updated.statusMessage)
}

// TestArchiveNextRepo_CompletesWhenAllDone verifies that archiveNextRepo
// returns an ArchiveCompleteMsg when all repos have been processed.
func TestArchiveNextRepo_CompletesWhenAllDone(t *testing.T) {
	state := &archiveState{
		repos:     []github.Repository{},
		succeeded: 3,
		failed:    1,
		errors:    []error{errors.New("error1")},
	}

	// Call with current >= len(repos)
	cmd := archiveNextRepo(nil, state.repos, 0, state)
	msg := cmd()

	completeMsg, ok := msg.(ArchiveCompleteMsg)
	require.True(t, ok, "expected ArchiveCompleteMsg, got %T", msg)
	assert.Equal(t, 3, completeMsg.Succeeded)
	assert.Equal(t, 1, completeMsg.Failed)
	assert.Len(t, completeMsg.Errors, 1)
}

// TestArchiveMarkedRepos_MarkedRepoNotInList verifies behavior when a marked
// repo key doesn't correspond to any repo in the list.
func TestArchiveMarkedRepos_MarkedRepoNotInList(t *testing.T) {
	repo := testutil.NewTestRepo(testutil.WithOwner("owner"), testutil.WithName("repo"))
	m := &Model{
		repos:  []github.Repository{repo},
		marked: map[string]bool{"nonexistent/repo": true}, // Marked repo not in list
		client: github.NewClient(testutil.NewMockExecutor()),
	}

	cmd := m.archiveMarkedRepos()

	// Should return nil because getMarkedRepos returns empty
	assert.Nil(t, cmd, "expected nil command when marked repo not found in list")
}

// TestGetMarkedRepos verifies that getMarkedRepos returns only repos that
// are both in the repos list and in the marked map.
func TestGetMarkedRepos(t *testing.T) {
	repo1 := testutil.NewTestRepo(testutil.WithOwner("owner1"), testutil.WithName("repo1"))
	repo2 := testutil.NewTestRepo(testutil.WithOwner("owner2"), testutil.WithName("repo2"))
	repo3 := testutil.NewTestRepo(testutil.WithOwner("owner3"), testutil.WithName("repo3"))

	m := Model{
		repos: []github.Repository{repo1, repo2, repo3},
		marked: map[string]bool{
			"owner1/repo1": true,
			"owner3/repo3": true,
			// repo2 not marked
		},
	}

	markedRepos := m.getMarkedRepos()

	assert.Len(t, markedRepos, 2)
	// Verify the correct repos are returned
	names := make([]string, len(markedRepos))
	for i, r := range markedRepos {
		names[i] = r.FullName()
	}
	assert.Contains(t, names, "owner1/repo1")
	assert.Contains(t, names, "owner3/repo3")
	assert.NotContains(t, names, "owner2/repo2")
}

// TestArchiveProgressMsg_WithError verifies that progress messages with errors
// update lastError but don't mark the repo as archived.
func TestArchiveProgressMsg_WithError(t *testing.T) {
	repo := testutil.NewTestRepo(testutil.WithOwner("owner"), testutil.WithName("repo"))
	m := Model{
		repos:           []github.Repository{repo},
		marked:          map[string]bool{"owner/repo": true},
		archiveProgress: 0,
		archiveTotal:    1,
		archiveState: &archiveState{
			repos:     []github.Repository{repo},
			succeeded: 0,
			failed:    1,
		},
		styles: DefaultStyles(),
	}

	archiveErr := errors.New("archive failed")
	progressMsg := ArchiveProgressMsg{
		Current:  1,
		Total:    1,
		RepoName: "owner/repo",
		Err:      archiveErr,
	}

	newModel, _ := m.Update(progressMsg)
	updated := newModel.(Model)

	// Verify error was recorded
	assert.Equal(t, archiveErr, updated.lastError)

	// Verify repo was NOT marked as archived (because it failed)
	assert.False(t, updated.repos[0].IsArchived)
}

// TestPluralize verifies the pluralize helper function.
func TestPluralize(t *testing.T) {
	tests := []struct {
		count int
		want  string
	}{
		{0, "s"},
		{1, ""},
		{2, "s"},
		{10, "s"},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := pluralize(tt.count)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestArchiveWorkflow_MultipleRepos tests archiving multiple repos
// through the archiveNextRepo function directly.
func TestArchiveWorkflow_MultipleRepos(t *testing.T) {
	repo1 := testutil.NewTestRepo(testutil.WithOwner("owner1"), testutil.WithName("repo1"))
	repo2 := testutil.NewTestRepo(testutil.WithOwner("owner2"), testutil.WithName("repo2"))
	repos := []github.Repository{repo1, repo2}

	callCount := 0
	mockExec := testutil.NewMockExecutor()
	mockExec.ExecuteFunc = func(name string, args ...string) ([]byte, error) {
		callCount++
		if callCount == 2 {
			return nil, errors.New("archive failed for repo2")
		}
		return nil, nil
	}
	client := github.NewClient(mockExec)

	state := &archiveState{
		repos:     repos,
		succeeded: 0,
		failed:    0,
		errors:    nil,
	}

	// Archive first repo - should succeed
	cmd1 := archiveNextRepo(client, repos, 0, state)
	msg1 := cmd1()
	progress1, ok := msg1.(ArchiveProgressMsg)
	require.True(t, ok)
	assert.Equal(t, 1, progress1.Current)
	assert.Nil(t, progress1.Err)
	assert.Equal(t, 1, state.succeeded)
	assert.Equal(t, 0, state.failed)

	// Archive second repo - should fail
	cmd2 := archiveNextRepo(client, repos, 1, state)
	msg2 := cmd2()
	progress2, ok := msg2.(ArchiveProgressMsg)
	require.True(t, ok)
	assert.Equal(t, 2, progress2.Current)
	assert.NotNil(t, progress2.Err)
	assert.Equal(t, 1, state.succeeded)
	assert.Equal(t, 1, state.failed)

	// Call once more to get completion message
	cmd3 := archiveNextRepo(client, repos, 2, state)
	msg3 := cmd3()
	complete, ok := msg3.(ArchiveCompleteMsg)
	require.True(t, ok)
	assert.Equal(t, 1, complete.Succeeded)
	assert.Equal(t, 1, complete.Failed)
	assert.Len(t, complete.Errors, 1)

	// Verify executor was called for both repos
	assert.Equal(t, 2, mockExec.CallCount())
}
