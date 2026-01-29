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

	_ "modernc.org/sqlite" // Pure Go SQLite driver
)

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

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		slog.Error("failed to open database", "component", "db", "path", dbPath, "error", err)
		return nil, fmt.Errorf("opening database: %w", err)
	}

	// Verify connection works
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("connecting to database: %w", err)
	}

	// Enable foreign keys for SQLite
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enabling foreign keys: %w", err)
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
