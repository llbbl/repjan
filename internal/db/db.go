// SPDX-FileCopyrightText: 2026 Logan Lindquist Land
// SPDX-License-Identifier: FSL-1.1-MIT

// Package db provides SQLite database access for repjan.
package db

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite" // Pure Go SQLite driver
)

// buildDSN appends the foreign_keys pragma to the given SQLite path so it is
// applied to every connection the database/sql pool opens. Running
// "PRAGMA foreign_keys = ON" on the *sql.DB only configures the single
// connection that happens to execute it; new pool connections would silently
// run with FK enforcement disabled. Passing the pragma via the DSN ensures
// modernc.org/sqlite sets it on each underlying connection.
//
// The function is robust to paths that already contain a query string
// (e.g. "file:foo.db?cache=shared") and to bare paths or ":memory:".
func buildDSN(path string) string {
	const pragma = "_pragma=foreign_keys(1)"

	// Split on the first '?' so we don't disturb any existing query.
	if i := strings.IndexByte(path, '?'); i >= 0 {
		base, query := path[:i], path[i+1:]
		if query == "" {
			return base + "?" + pragma
		}
		return base + "?" + query + "&" + pragma
	}
	return path + "?" + pragma
}

// GetDefaultDBPath returns the default database path (~/.repjan/repjan.db).
// It creates the directory if it doesn't exist.
func GetDefaultDBPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home directory: %w", err)
	}

	repjanDir := filepath.Join(homeDir, ".repjan")
	if err := os.MkdirAll(repjanDir, 0750); err != nil {
		return "", fmt.Errorf("creating repjan directory: %w", err)
	}

	return filepath.Join(repjanDir, "repjan.db"), nil
}

// Open opens or creates a SQLite database at the specified path.
// Use ":memory:" for an in-memory database (useful for testing).
func Open(dbPath string) (*sql.DB, error) {
	slog.Info("opening database", "component", "db", "path", dbPath)

	db, err := sql.Open("sqlite", buildDSN(dbPath))
	if err != nil {
		slog.Error("failed to open database", "component", "db", "path", dbPath, "error", err)
		return nil, fmt.Errorf("opening database: %w", err)
	}

	// Verify connection works
	if err := db.Ping(); err != nil {
		if closeErr := db.Close(); closeErr != nil {
			slog.Warn("failed to close database after ping failure", "component", "db", "error", closeErr)
		}
		return nil, fmt.Errorf("connecting to database: %w", err)
	}

	return db, nil
}

// Close closes the database connection.
func Close(db *sql.DB) error {
	if db == nil {
		return nil
	}
	slog.Debug("closing database", "component", "db")
	if err := db.Close(); err != nil {
		slog.Error("failed to close database", "component", "db", "error", err)
		return fmt.Errorf("closing database: %w", err)
	}
	return nil
}
