// SPDX-FileCopyrightText: 2026 Logan Lindquist Land
// SPDX-License-Identifier: FSL-1.1-MIT

// Package store provides data access layer for repository storage.
package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/llbbl/repjan/internal/github"
)

// ErrNotFound is returned when a requested record does not exist.
var ErrNotFound = errors.New("not found")

// Store provides data access methods for repositories.
type Store struct {
	db *sql.DB
}

// New creates a new Store with the given database connection.
func New(db *sql.DB) *Store {
	return &Store{db: db}
}

// UpsertRepositories bulk upserts repos from GitHub fetch.
// Uses INSERT OR REPLACE for efficiency within a transaction.
func (s *Store) UpsertRepositories(owner string, repos []github.Repository) error {
	if len(repos) == 0 {
		return nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // Rollback is no-op after commit

	stmt, err := tx.Prepare(`
		INSERT OR REPLACE INTO repositories (
			owner, name, full_name, description, stars, forks,
			is_archived, is_fork, is_private, primary_language,
			pushed_at, created_at, days_since_activity, synced_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("preparing statement: %w", err)
	}
	defer stmt.Close()

	now := formatTimeForSQLite(time.Now())
	for _, repo := range repos {
		// Ensure owner is set (use provided owner if repo.Owner is empty)
		repoOwner := repo.Owner
		if repoOwner == "" {
			repoOwner = owner
		}

		// Calculate days since activity if not already set
		daysSinceActivity := repo.DaysSinceActivity
		if daysSinceActivity == 0 && !repo.PushedAt.IsZero() {
			daysSinceActivity = int(time.Since(repo.PushedAt).Hours() / 24)
		}

		fullName := repoOwner + "/" + repo.Name

		_, err := stmt.Exec(
			repoOwner,
			repo.Name,
			fullName,
			nullString(repo.Description),
			repo.StargazerCount,
			repo.ForkCount,
			repo.IsArchived,
			repo.IsFork,
			repo.IsPrivate,
			nullString(repo.PrimaryLanguage),
			formatTimeForSQLite(repo.PushedAt),
			formatTimeForSQLite(repo.CreatedAt),
			daysSinceActivity,
			now,
		)
		if err != nil {
			return fmt.Errorf("inserting repository %s: %w", fullName, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	return nil
}

// GetRepositories loads all repos for an owner from the database.
func (s *Store) GetRepositories(owner string) ([]github.Repository, error) {
	rows, err := s.db.Query(`
		SELECT owner, name, description, stars, forks,
			   is_archived, is_fork, is_private, primary_language,
			   pushed_at, created_at, days_since_activity
		FROM repositories
		WHERE owner = ?
		ORDER BY name
	`, owner)
	if err != nil {
		return nil, fmt.Errorf("querying repositories: %w", err)
	}
	defer rows.Close()

	var repos []github.Repository
	for rows.Next() {
		repo, err := scanRepository(rows)
		if err != nil {
			return nil, err
		}
		repos = append(repos, repo)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating rows: %w", err)
	}

	return repos, nil
}

// GetRepository loads a single repo by owner and name.
// Returns ErrNotFound if the repository does not exist.
func (s *Store) GetRepository(owner, name string) (*github.Repository, error) {
	row := s.db.QueryRow(`
		SELECT owner, name, description, stars, forks,
			   is_archived, is_fork, is_private, primary_language,
			   pushed_at, created_at, days_since_activity
		FROM repositories
		WHERE owner = ? AND name = ?
	`, owner, name)

	repo, err := scanRepositoryRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("querying repository: %w", err)
	}

	return &repo, nil
}

// UpdateRepository updates a single repository.
func (s *Store) UpdateRepository(repo github.Repository) error {
	result, err := s.db.Exec(`
		UPDATE repositories SET
			description = ?,
			stars = ?,
			forks = ?,
			is_archived = ?,
			is_fork = ?,
			is_private = ?,
			primary_language = ?,
			pushed_at = ?,
			created_at = ?,
			days_since_activity = ?,
			synced_at = ?
		WHERE owner = ? AND name = ?
	`,
		nullString(repo.Description),
		repo.StargazerCount,
		repo.ForkCount,
		repo.IsArchived,
		repo.IsFork,
		repo.IsPrivate,
		nullString(repo.PrimaryLanguage),
		formatTimeForSQLite(repo.PushedAt),
		formatTimeForSQLite(repo.CreatedAt),
		repo.DaysSinceActivity,
		formatTimeForSQLite(time.Now()),
		repo.Owner,
		repo.Name,
	)
	if err != nil {
		return fmt.Errorf("updating repository: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("getting rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

// DeleteStaleRepositories removes repos not seen since the given time.
// Returns the number of deleted repositories.
func (s *Store) DeleteStaleRepositories(owner string, olderThan time.Time) (int64, error) {
	result, err := s.db.Exec(`
		DELETE FROM repositories
		WHERE owner = ? AND synced_at < ?
	`, owner, olderThan.UTC().Format(sqliteTimeFormat))
	if err != nil {
		return 0, fmt.Errorf("deleting stale repositories: %w", err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("getting rows affected: %w", err)
	}

	return count, nil
}

// GetLastSyncTime returns when repos for this owner were last synced.
// Returns zero time if no repos exist for the owner.
func (s *Store) GetLastSyncTime(owner string) (time.Time, error) {
	var syncedAt sql.NullString
	err := s.db.QueryRow(`
		SELECT MAX(synced_at)
		FROM repositories
		WHERE owner = ?
	`, owner).Scan(&syncedAt)
	if err != nil {
		return time.Time{}, fmt.Errorf("querying last sync time: %w", err)
	}

	if !syncedAt.Valid || syncedAt.String == "" {
		return time.Time{}, nil
	}

	t, err := parseTimeFromSQLite(syncedAt.String)
	if err != nil {
		return time.Time{}, fmt.Errorf("parsing sync time: %w", err)
	}

	return t, nil
}

// scanner is an interface for both *sql.Row and *sql.Rows.
type scanner interface {
	Scan(dest ...any) error
}

// scanRepository scans a repository from sql.Rows.
func scanRepository(rows *sql.Rows) (github.Repository, error) {
	return scanRepo(rows)
}

// scanRepositoryRow scans a repository from sql.Row.
func scanRepositoryRow(row *sql.Row) (github.Repository, error) {
	return scanRepo(row)
}

// scanRepo handles the common scanning logic.
func scanRepo(s scanner) (github.Repository, error) {
	var repo github.Repository
	var description, primaryLanguage, pushedAt, createdAt sql.NullString

	err := s.Scan(
		&repo.Owner,
		&repo.Name,
		&description,
		&repo.StargazerCount,
		&repo.ForkCount,
		&repo.IsArchived,
		&repo.IsFork,
		&repo.IsPrivate,
		&primaryLanguage,
		&pushedAt,
		&createdAt,
		&repo.DaysSinceActivity,
	)
	if err != nil {
		return github.Repository{}, err
	}

	if description.Valid {
		repo.Description = description.String
	}
	if primaryLanguage.Valid {
		repo.PrimaryLanguage = primaryLanguage.String
	}
	if pushedAt.Valid && pushedAt.String != "" {
		t, err := parseTimeFromSQLite(pushedAt.String)
		if err != nil {
			return github.Repository{}, fmt.Errorf("parsing pushed_at: %w", err)
		}
		repo.PushedAt = t
	}
	if createdAt.Valid && createdAt.String != "" {
		t, err := parseTimeFromSQLite(createdAt.String)
		if err != nil {
			return github.Repository{}, fmt.Errorf("parsing created_at: %w", err)
		}
		repo.CreatedAt = t
	}

	return repo, nil
}

// nullString returns a sql.NullString for the given string.
func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

// sqliteTimeFormat is the standard SQLite datetime format.
const sqliteTimeFormat = "2006-01-02 15:04:05"

// formatTimeForSQLite converts a time to SQLite format string, or nil if zero.
func formatTimeForSQLite(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t.UTC().Format(sqliteTimeFormat)
}

// parseTimeFromSQLite parses a time string from SQLite, handling multiple formats.
func parseTimeFromSQLite(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}

	// Try common SQLite formats
	formats := []string{
		"2006-01-02 15:04:05",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05.999999999Z",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("cannot parse time: %s", s)
}
