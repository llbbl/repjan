// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

package db

import (
	"database/sql"
	"embed"
	"fmt"

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

	if err := goose.Up(db, "migrations"); err != nil {
		return fmt.Errorf("running migrations: %w", err)
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
