// SPDX-FileCopyrightText: 2026 Logan Lindquist Land
// SPDX-License-Identifier: FSL-1.1-MIT

package db

import (
	"database/sql"
	"embed"
	"fmt"
	"log/slog"

	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

// RunMigrations runs all pending database migrations.
func RunMigrations(db *sql.DB) error {
	goose.SetBaseFS(embedMigrations)

	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("setting goose dialect: %w", err)
	}

	// Get version before migrations
	versionBefore, err := goose.GetDBVersion(db)
	if err != nil {
		slog.Debug("could not get migration version before", "component", "db", "error", err)
		versionBefore = 0
	}

	slog.Info("running migrations", "component", "db", "version_before", versionBefore)

	if err := goose.Up(db, "migrations"); err != nil {
		slog.Error("failed to run migrations", "component", "db", "error", err)
		return fmt.Errorf("running migrations: %w", err)
	}

	// Get version after migrations
	versionAfter, err := goose.GetDBVersion(db)
	if err != nil {
		slog.Debug("could not get migration version after", "component", "db", "error", err)
	} else {
		slog.Info("migrations complete", "component", "db", "version_after", versionAfter)
	}

	return nil
}

// GetMigrationVersion returns the current migration version.
func GetMigrationVersion(db *sql.DB) (int64, error) {
	goose.SetBaseFS(embedMigrations)

	if err := goose.SetDialect("sqlite3"); err != nil {
		return 0, fmt.Errorf("setting goose dialect: %w", err)
	}

	version, err := goose.GetDBVersion(db)
	if err != nil {
		return 0, fmt.Errorf("getting migration version: %w", err)
	}

	return version, nil
}
