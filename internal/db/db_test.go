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

	// Check version
	version, err := GetMigrationVersion(db)
	require.NoError(t, err)
	assert.Equal(t, int64(1), version, "migration version should be 1 after running 001_create_repositories.sql")
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
