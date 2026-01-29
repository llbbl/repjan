// SPDX-FileCopyrightText: 2026 Logan Lindquist Land
// SPDX-License-Identifier: FSL-1.1-MIT

package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpen_InMemory(t *testing.T) {
	db, err := Open(":memory:")
	require.NoError(t, err)
	defer Close(db)

	// Verify connection works
	err = db.Ping()
	assert.NoError(t, err)
}

func TestRunMigrations_CreatesRepositoriesTable(t *testing.T) {
	db, err := Open(":memory:")
	require.NoError(t, err)
	defer Close(db)

	// Run migrations
	err = RunMigrations(db)
	require.NoError(t, err)

	// Verify repositories table exists by querying it
	_, err = db.Exec("SELECT id, owner, name, full_name, description, stars, forks, is_archived, is_fork, is_private, primary_language, pushed_at, created_at, days_since_activity, synced_at FROM repositories LIMIT 1")
	assert.NoError(t, err, "repositories table should exist with expected columns")
}

func TestRunMigrations_Idempotent(t *testing.T) {
	db, err := Open(":memory:")
	require.NoError(t, err)
	defer Close(db)

	// Run migrations twice - should not error
	err = RunMigrations(db)
	require.NoError(t, err)

	err = RunMigrations(db)
	assert.NoError(t, err, "running migrations twice should be idempotent")
}

func TestGetMigrationVersion(t *testing.T) {
	db, err := Open(":memory:")
	require.NoError(t, err)
	defer Close(db)

	// Run migrations
	err = RunMigrations(db)
	require.NoError(t, err)

	// Check version - should be 4 after running all migrations
	version, err := GetMigrationVersion(db)
	require.NoError(t, err)
	assert.Equal(t, int64(4), version, "migration version should be 4 after running all migrations")
}

func TestClose_NilDB(t *testing.T) {
	// Should not panic or error
	err := Close(nil)
	assert.NoError(t, err)
}

func TestGetDefaultDBPath(t *testing.T) {
	path, err := GetDefaultDBPath()
	require.NoError(t, err)
	assert.Contains(t, path, ".repjan")
	assert.Contains(t, path, "repjan.db")
}

func TestRepositoriesTable_Indexes(t *testing.T) {
	db, err := Open(":memory:")
	require.NoError(t, err)
	defer Close(db)

	err = RunMigrations(db)
	require.NoError(t, err)

	// Verify indexes exist by checking sqlite_master
	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='index' AND tbl_name='repositories'")
	require.NoError(t, err)
	defer rows.Close()

	var indexes []string
	for rows.Next() {
		var name string
		err := rows.Scan(&name)
		require.NoError(t, err)
		indexes = append(indexes, name)
	}

	assert.Contains(t, indexes, "idx_repositories_owner")
	assert.Contains(t, indexes, "idx_repositories_synced_at")
}

func TestRepositoriesTable_UniqueConstraints(t *testing.T) {
	db, err := Open(":memory:")
	require.NoError(t, err)
	defer Close(db)

	err = RunMigrations(db)
	require.NoError(t, err)

	// Insert a repository
	_, err = db.Exec(`INSERT INTO repositories (owner, name, full_name) VALUES ('test-owner', 'test-repo', 'test-owner/test-repo')`)
	require.NoError(t, err)

	// Try to insert duplicate full_name - should fail
	_, err = db.Exec(`INSERT INTO repositories (owner, name, full_name) VALUES ('other-owner', 'other-repo', 'test-owner/test-repo')`)
	assert.Error(t, err, "duplicate full_name should violate unique constraint")

	// Try to insert duplicate owner+name - should fail
	_, err = db.Exec(`INSERT INTO repositories (owner, name, full_name) VALUES ('test-owner', 'test-repo', 'different/full-name')`)
	assert.Error(t, err, "duplicate owner+name should violate unique constraint")
}

func TestMarkedReposTable_Exists(t *testing.T) {
	db, err := Open(":memory:")
	require.NoError(t, err)
	defer Close(db)

	err = RunMigrations(db)
	require.NoError(t, err)

	// Verify marked_repos table exists by querying it
	_, err = db.Exec("SELECT id, owner, repo_name, marked_at FROM marked_repos LIMIT 1")
	assert.NoError(t, err, "marked_repos table should exist with expected columns")
}

func TestMarkedReposTable_UniqueConstraint(t *testing.T) {
	db, err := Open(":memory:")
	require.NoError(t, err)
	defer Close(db)

	err = RunMigrations(db)
	require.NoError(t, err)

	// Insert a marked repo
	_, err = db.Exec(`INSERT INTO marked_repos (owner, repo_name) VALUES ('test-owner', 'test-repo')`)
	require.NoError(t, err)

	// Try to insert duplicate owner+repo_name - should fail
	_, err = db.Exec(`INSERT INTO marked_repos (owner, repo_name) VALUES ('test-owner', 'test-repo')`)
	assert.Error(t, err, "duplicate owner+repo_name should violate unique constraint")

	// Same repo for different owner should succeed
	_, err = db.Exec(`INSERT INTO marked_repos (owner, repo_name) VALUES ('other-owner', 'test-repo')`)
	assert.NoError(t, err, "same repo_name for different owner should succeed")
}

func TestMarkedReposTable_Index(t *testing.T) {
	db, err := Open(":memory:")
	require.NoError(t, err)
	defer Close(db)

	err = RunMigrations(db)
	require.NoError(t, err)

	// Verify index exists by checking sqlite_master
	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='index' AND tbl_name='marked_repos'")
	require.NoError(t, err)
	defer rows.Close()

	var indexes []string
	for rows.Next() {
		var name string
		err := rows.Scan(&name)
		require.NoError(t, err)
		indexes = append(indexes, name)
	}

	assert.Contains(t, indexes, "idx_marked_repos_owner")
}

func TestSyncHistoryTable_Exists(t *testing.T) {
	db, err := Open(":memory:")
	require.NoError(t, err)
	defer Close(db)

	err = RunMigrations(db)
	require.NoError(t, err)

	// Verify sync_history table exists by querying all expected columns
	_, err = db.Exec(`SELECT id, owner, started_at, completed_at, status,
		repos_fetched, repos_inserted, repos_updated, error_message, duration_ms
		FROM sync_history LIMIT 1`)
	assert.NoError(t, err, "sync_history table should exist with expected columns")
}

func TestSyncHistoryTable_Indexes(t *testing.T) {
	db, err := Open(":memory:")
	require.NoError(t, err)
	defer Close(db)

	err = RunMigrations(db)
	require.NoError(t, err)

	// Verify indexes exist by checking sqlite_master
	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='index' AND tbl_name='sync_history'")
	require.NoError(t, err)
	defer rows.Close()

	var indexes []string
	for rows.Next() {
		var name string
		err := rows.Scan(&name)
		require.NoError(t, err)
		indexes = append(indexes, name)
	}

	assert.Contains(t, indexes, "idx_sync_history_owner")
	assert.Contains(t, indexes, "idx_sync_history_started_at")
	assert.Contains(t, indexes, "idx_sync_history_owner_status")
}

func TestSyncHistoryTable_DefaultValues(t *testing.T) {
	db, err := Open(":memory:")
	require.NoError(t, err)
	defer Close(db)

	err = RunMigrations(db)
	require.NoError(t, err)

	// Insert with minimal fields to test defaults
	result, err := db.Exec(`INSERT INTO sync_history (owner) VALUES ('test-owner')`)
	require.NoError(t, err)

	id, err := result.LastInsertId()
	require.NoError(t, err)

	// Verify default values
	var status string
	var reposFetched, reposInserted, reposUpdated int
	err = db.QueryRow(`SELECT status, repos_fetched, repos_inserted, repos_updated FROM sync_history WHERE id = ?`, id).
		Scan(&status, &reposFetched, &reposInserted, &reposUpdated)
	require.NoError(t, err)

	assert.Equal(t, "running", status, "default status should be 'running'")
	assert.Equal(t, 0, reposFetched, "default repos_fetched should be 0")
	assert.Equal(t, 0, reposInserted, "default repos_inserted should be 0")
	assert.Equal(t, 0, reposUpdated, "default repos_updated should be 0")
}

func TestRepoChangesTable_Exists(t *testing.T) {
	db, err := Open(":memory:")
	require.NoError(t, err)
	defer Close(db)

	err = RunMigrations(db)
	require.NoError(t, err)

	// Verify repo_changes table exists by querying all expected columns
	_, err = db.Exec(`SELECT id, owner, repo_name, action, performed_at, performed_by,
		previous_state, new_state, notes
		FROM repo_changes LIMIT 1`)
	assert.NoError(t, err, "repo_changes table should exist with expected columns")
}

func TestRepoChangesTable_Indexes(t *testing.T) {
	db, err := Open(":memory:")
	require.NoError(t, err)
	defer Close(db)

	err = RunMigrations(db)
	require.NoError(t, err)

	// Verify indexes exist by checking sqlite_master
	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='index' AND tbl_name='repo_changes'")
	require.NoError(t, err)
	defer rows.Close()

	var indexes []string
	for rows.Next() {
		var name string
		err := rows.Scan(&name)
		require.NoError(t, err)
		indexes = append(indexes, name)
	}

	assert.Contains(t, indexes, "idx_repo_changes_owner")
	assert.Contains(t, indexes, "idx_repo_changes_repo")
	assert.Contains(t, indexes, "idx_repo_changes_action")
	assert.Contains(t, indexes, "idx_repo_changes_performed_at")
}

func TestRepoChangesTable_DefaultValues(t *testing.T) {
	db, err := Open(":memory:")
	require.NoError(t, err)
	defer Close(db)

	err = RunMigrations(db)
	require.NoError(t, err)

	// Insert with minimal required fields to test defaults
	result, err := db.Exec(`INSERT INTO repo_changes (owner, repo_name, action) VALUES ('test-owner', 'test-repo', 'archived')`)
	require.NoError(t, err)

	id, err := result.LastInsertId()
	require.NoError(t, err)

	// Verify default values
	var performedBy string
	err = db.QueryRow(`SELECT performed_by FROM repo_changes WHERE id = ?`, id).Scan(&performedBy)
	require.NoError(t, err)

	assert.Equal(t, "user", performedBy, "default performed_by should be 'user'")
}

func TestRepoChangesTable_NullableFields(t *testing.T) {
	db, err := Open(":memory:")
	require.NoError(t, err)
	defer Close(db)

	err = RunMigrations(db)
	require.NoError(t, err)

	// Insert without optional fields - should succeed
	_, err = db.Exec(`INSERT INTO repo_changes (owner, repo_name, action) VALUES ('test-owner', 'test-repo', 'marked')`)
	require.NoError(t, err, "insert without nullable fields should succeed")

	// Insert with all fields including nullable ones
	_, err = db.Exec(`INSERT INTO repo_changes (owner, repo_name, action, performed_by, previous_state, new_state, notes)
		VALUES ('test-owner', 'test-repo', 'archived', 'system', '{"is_archived": false}', '{"is_archived": true}', 'User requested archive')`)
	assert.NoError(t, err, "insert with all fields should succeed")
}
