// SPDX-FileCopyrightText: 2026 Logan Lindquist Land
// SPDX-License-Identifier: FSL-1.1-MIT

// Package store provides data access layer for repository storage.
package store

import (
	"database/sql"
	"encoding/json"
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

// SaveMarkedRepos saves the marked repos for an owner, replacing any existing marks.
// This uses a transaction to ensure atomicity: clear old marks, insert new ones.
func (s *Store) SaveMarkedRepos(owner string, repoNames []string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // Rollback is no-op after commit

	// Clear existing marks for this owner
	_, err = tx.Exec(`DELETE FROM marked_repos WHERE owner = ?`, owner)
	if err != nil {
		return fmt.Errorf("clearing existing marks: %w", err)
	}

	// Insert new marks
	if len(repoNames) > 0 {
		stmt, err := tx.Prepare(`
			INSERT INTO marked_repos (owner, repo_name) VALUES (?, ?)
		`)
		if err != nil {
			return fmt.Errorf("preparing statement: %w", err)
		}
		defer stmt.Close()

		for _, name := range repoNames {
			_, err := stmt.Exec(owner, name)
			if err != nil {
				return fmt.Errorf("inserting marked repo %s: %w", name, err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	return nil
}

// GetMarkedRepos returns the list of marked repo names for an owner.
func (s *Store) GetMarkedRepos(owner string) ([]string, error) {
	rows, err := s.db.Query(`
		SELECT repo_name FROM marked_repos WHERE owner = ? ORDER BY repo_name
	`, owner)
	if err != nil {
		return nil, fmt.Errorf("querying marked repos: %w", err)
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scanning marked repo: %w", err)
		}
		names = append(names, name)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating rows: %w", err)
	}

	return names, nil
}

// ClearMarkedRepos removes all marks for an owner.
func (s *Store) ClearMarkedRepos(owner string) error {
	_, err := s.db.Exec(`DELETE FROM marked_repos WHERE owner = ?`, owner)
	if err != nil {
		return fmt.Errorf("clearing marked repos: %w", err)
	}
	return nil
}

// RemoveMarkedRepo removes a single marked repo by name.
func (s *Store) RemoveMarkedRepo(owner, repoName string) error {
	_, err := s.db.Exec(`DELETE FROM marked_repos WHERE owner = ? AND repo_name = ?`, owner, repoName)
	if err != nil {
		return fmt.Errorf("removing marked repo: %w", err)
	}
	return nil
}

// AddMarkedRepo adds a single marked repo.
func (s *Store) AddMarkedRepo(owner, repoName string) error {
	_, err := s.db.Exec(`
		INSERT OR IGNORE INTO marked_repos (owner, repo_name) VALUES (?, ?)
	`, owner, repoName)
	if err != nil {
		return fmt.Errorf("adding marked repo: %w", err)
	}
	return nil
}

// RepoChange represents an audit record of a repository change.
type RepoChange struct {
	ID            int64
	Owner         string
	RepoName      string
	Action        string    // archived, marked, unmarked, deleted, synced
	PerformedAt   time.Time
	PerformedBy   string // user, system, sync
	PreviousState string // JSON string
	NewState      string // JSON string
	Notes         string
}

// RecordRepoChange records a modification to a repository.
// prevState and newState can be any JSON-serializable value (or nil).
func (s *Store) RecordRepoChange(owner, repoName, action, performedBy string, prevState, newState interface{}, notes string) error {
	var prevJSON, newJSON string
	if prevState != nil {
		if data, err := json.Marshal(prevState); err == nil {
			prevJSON = string(data)
		}
	}
	if newState != nil {
		if data, err := json.Marshal(newState); err == nil {
			newJSON = string(data)
		}
	}

	_, err := s.db.Exec(`
		INSERT INTO repo_changes (owner, repo_name, action, performed_by, previous_state, new_state, notes)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, owner, repoName, action, performedBy, nullString(prevJSON), nullString(newJSON), nullString(notes))
	if err != nil {
		return fmt.Errorf("recording repo change: %w", err)
	}

	return nil
}

// GetRepoHistory returns change history for a specific repository.
func (s *Store) GetRepoHistory(owner, repoName string, limit int) ([]RepoChange, error) {
	rows, err := s.db.Query(`
		SELECT id, owner, repo_name, action, performed_at, performed_by, previous_state, new_state, notes
		FROM repo_changes
		WHERE owner = ? AND repo_name = ?
		ORDER BY performed_at DESC, id DESC
		LIMIT ?
	`, owner, repoName, limit)
	if err != nil {
		return nil, fmt.Errorf("querying repo history: %w", err)
	}
	defer rows.Close()

	return scanRepoChanges(rows)
}

// GetRecentChanges returns recent changes across all repos for an owner.
func (s *Store) GetRecentChanges(owner string, limit int) ([]RepoChange, error) {
	rows, err := s.db.Query(`
		SELECT id, owner, repo_name, action, performed_at, performed_by, previous_state, new_state, notes
		FROM repo_changes
		WHERE owner = ?
		ORDER BY performed_at DESC, id DESC
		LIMIT ?
	`, owner, limit)
	if err != nil {
		return nil, fmt.Errorf("querying recent changes: %w", err)
	}
	defer rows.Close()

	return scanRepoChanges(rows)
}

// GetChangesByAction returns changes filtered by action type.
func (s *Store) GetChangesByAction(owner, action string, limit int) ([]RepoChange, error) {
	rows, err := s.db.Query(`
		SELECT id, owner, repo_name, action, performed_at, performed_by, previous_state, new_state, notes
		FROM repo_changes
		WHERE owner = ? AND action = ?
		ORDER BY performed_at DESC, id DESC
		LIMIT ?
	`, owner, action, limit)
	if err != nil {
		return nil, fmt.Errorf("querying changes by action: %w", err)
	}
	defer rows.Close()

	return scanRepoChanges(rows)
}

// scanRepoChanges scans rows into a slice of RepoChange.
func scanRepoChanges(rows *sql.Rows) ([]RepoChange, error) {
	var changes []RepoChange
	for rows.Next() {
		var change RepoChange
		var performedAt, prevState, newState, notes sql.NullString

		err := rows.Scan(
			&change.ID,
			&change.Owner,
			&change.RepoName,
			&change.Action,
			&performedAt,
			&change.PerformedBy,
			&prevState,
			&newState,
			&notes,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning repo change: %w", err)
		}

		if performedAt.Valid && performedAt.String != "" {
			t, err := parseTimeFromSQLite(performedAt.String)
			if err != nil {
				return nil, fmt.Errorf("parsing performed_at: %w", err)
			}
			change.PerformedAt = t
		}
		if prevState.Valid {
			change.PreviousState = prevState.String
		}
		if newState.Valid {
			change.NewState = newState.String
		}
		if notes.Valid {
			change.Notes = notes.String
		}

		changes = append(changes, change)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating rows: %w", err)
	}

	return changes, nil
}

// SyncRecord represents a sync history entry.
type SyncRecord struct {
	ID            int64
	Owner         string
	StartedAt     time.Time
	CompletedAt   *time.Time // nullable
	Status        string     // running, success, error, partial
	ReposFetched  int
	ReposInserted int
	ReposUpdated  int
	ErrorMessage  string
	DurationMs    int64
}

// RecordSyncStart creates a new sync record and returns its ID.
func (s *Store) RecordSyncStart(owner string) (int64, error) {
	result, err := s.db.Exec(`
		INSERT INTO sync_history (owner, started_at, status)
		VALUES (?, ?, 'running')
	`, owner, formatTimeForSQLite(time.Now()))
	if err != nil {
		return 0, fmt.Errorf("inserting sync record: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("getting last insert id: %w", err)
	}

	return id, nil
}

// RecordSyncComplete updates a sync record with completion data.
func (s *Store) RecordSyncComplete(syncID int64, status string, fetched, inserted, updated int, errMsg string) error {
	now := time.Now()
	startedAt, err := s.getSyncStartedAt(syncID)
	if err != nil {
		return fmt.Errorf("getting sync started_at: %w", err)
	}

	durationMs := now.Sub(startedAt).Milliseconds()

	result, err := s.db.Exec(`
		UPDATE sync_history SET
			completed_at = ?,
			status = ?,
			repos_fetched = ?,
			repos_inserted = ?,
			repos_updated = ?,
			error_message = ?,
			duration_ms = ?
		WHERE id = ?
	`,
		formatTimeForSQLite(now),
		status,
		fetched,
		inserted,
		updated,
		nullString(errMsg),
		durationMs,
		syncID,
	)
	if err != nil {
		return fmt.Errorf("updating sync record: %w", err)
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

// getSyncStartedAt retrieves the started_at time for a sync record.
func (s *Store) getSyncStartedAt(syncID int64) (time.Time, error) {
	var startedAt string
	err := s.db.QueryRow(`SELECT started_at FROM sync_history WHERE id = ?`, syncID).Scan(&startedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return time.Time{}, ErrNotFound
		}
		return time.Time{}, err
	}

	return parseTimeFromSQLite(startedAt)
}

// GetSyncHistory returns recent sync records for an owner.
func (s *Store) GetSyncHistory(owner string, limit int) ([]SyncRecord, error) {
	rows, err := s.db.Query(`
		SELECT id, owner, started_at, completed_at, status,
		       repos_fetched, repos_inserted, repos_updated,
		       error_message, duration_ms
		FROM sync_history
		WHERE owner = ?
		ORDER BY started_at DESC, id DESC
		LIMIT ?
	`, owner, limit)
	if err != nil {
		return nil, fmt.Errorf("querying sync history: %w", err)
	}
	defer rows.Close()

	var records []SyncRecord
	for rows.Next() {
		record, err := scanSyncRecord(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating rows: %w", err)
	}

	return records, nil
}

// GetLastSuccessfulSync returns the most recent successful sync for an owner.
func (s *Store) GetLastSuccessfulSync(owner string) (*SyncRecord, error) {
	row := s.db.QueryRow(`
		SELECT id, owner, started_at, completed_at, status,
		       repos_fetched, repos_inserted, repos_updated,
		       error_message, duration_ms
		FROM sync_history
		WHERE owner = ? AND status = 'success'
		ORDER BY started_at DESC, id DESC
		LIMIT 1
	`, owner)

	record, err := scanSyncRecordRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("querying last successful sync: %w", err)
	}

	return &record, nil
}

// scanSyncRecord scans a sync record from sql.Rows.
func scanSyncRecord(rows *sql.Rows) (SyncRecord, error) {
	return scanSync(rows)
}

// scanSyncRecordRow scans a sync record from sql.Row.
func scanSyncRecordRow(row *sql.Row) (SyncRecord, error) {
	return scanSync(row)
}

// scanSync handles the common scanning logic for sync records.
func scanSync(s scanner) (SyncRecord, error) {
	var record SyncRecord
	var startedAt, completedAt, errorMessage sql.NullString
	var durationMs sql.NullInt64
	var reposFetched, reposInserted, reposUpdated sql.NullInt64

	err := s.Scan(
		&record.ID,
		&record.Owner,
		&startedAt,
		&completedAt,
		&record.Status,
		&reposFetched,
		&reposInserted,
		&reposUpdated,
		&errorMessage,
		&durationMs,
	)
	if err != nil {
		return SyncRecord{}, err
	}

	if startedAt.Valid && startedAt.String != "" {
		t, err := parseTimeFromSQLite(startedAt.String)
		if err != nil {
			return SyncRecord{}, fmt.Errorf("parsing started_at: %w", err)
		}
		record.StartedAt = t
	}

	if completedAt.Valid && completedAt.String != "" {
		t, err := parseTimeFromSQLite(completedAt.String)
		if err != nil {
			return SyncRecord{}, fmt.Errorf("parsing completed_at: %w", err)
		}
		record.CompletedAt = &t
	}

	if errorMessage.Valid {
		record.ErrorMessage = errorMessage.String
	}

	if durationMs.Valid {
		record.DurationMs = durationMs.Int64
	}

	if reposFetched.Valid {
		record.ReposFetched = int(reposFetched.Int64)
	}

	if reposInserted.Valid {
		record.ReposInserted = int(reposInserted.Int64)
	}

	if reposUpdated.Valid {
		record.ReposUpdated = int(reposUpdated.Int64)
	}

	return record, nil
}
