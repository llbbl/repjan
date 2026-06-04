// SPDX-FileCopyrightText: 2026 Logan Lindquist Land
// SPDX-License-Identifier: FSL-1.1-MIT

package cmd

import (
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/llbbl/repjan/internal/db"
	"github.com/llbbl/repjan/internal/github"
	"github.com/llbbl/repjan/internal/store"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync repositories from GitHub to database",
	Long: `Fetch repositories from GitHub and store them in the local database.
If --owner is not specified, uses the authenticated GitHub user.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Create GitHub client
		client := github.NewDefaultClient()

		// Determine owner
		targetOwner := owner
		if targetOwner == "" {
			slog.Debug("no owner specified, getting authenticated user", "component", "cmd")
			user, err := client.GetAuthenticatedUser()
			if err != nil {
				slog.Error("failed to get authenticated user", "component", "cmd", "error", err)
				return fmt.Errorf("failed to get authenticated user: %w\nMake sure you're logged in with 'gh auth login'", err)
			}
			targetOwner = user
			slog.Debug("using authenticated user", "component", "cmd", "owner", targetOwner)
		}

		// Open database
		dbPath, err := db.GetDefaultDBPath()
		if err != nil {
			slog.Error("failed to get database path", "component", "cmd", "error", err)
			return fmt.Errorf("getting database path: %w", err)
		}

		slog.Debug("opening database", "component", "cmd", "path", dbPath)
		database, err := db.Open(dbPath)
		if err != nil {
			slog.Error("failed to open database", "component", "cmd", "path", dbPath, "error", err)
			return fmt.Errorf("opening database: %w", err)
		}
		defer func() { _ = db.Close(database) }()

		// Ensure migrations are run
		slog.Debug("running migrations", "component", "cmd")
		if err := db.RunMigrations(database); err != nil {
			slog.Error("migration failed", "component", "cmd", "error", err)
			return fmt.Errorf("running migrations: %w", err)
		}

		// Fetch repositories from GitHub
		fmt.Printf("Fetching repositories for %s...\n", targetOwner)
		slog.Debug("fetching repositories from GitHub", "component", "cmd", "owner", targetOwner)
		repos, err := client.FetchRepositories(targetOwner)
		if err != nil {
			slog.Error("failed to fetch repositories", "component", "cmd", "owner", targetOwner, "error", err)
			return fmt.Errorf("fetching repositories: %w", err)
		}
		fmt.Printf("Found %d repositories\n", len(repos))
		slog.Debug("fetched repositories", "component", "cmd", "count", len(repos))

		// Upsert repositories to database via the store layer so writes use the
		// same time format the store reads back (see store.formatTimeForSQLite).
		slog.Debug("upserting repositories to database", "component", "cmd", "count", len(repos))
		s := store.New(database)
		if err := s.UpsertRepositories(targetOwner, repos); err != nil {
			slog.Error("failed to upsert repositories", "component", "cmd", "error", err)
			return fmt.Errorf("upserting repositories: %w", err)
		}

		fmt.Printf("Sync complete: %d repositories synced\n", len(repos))
		slog.Debug("sync completed", "component", "cmd", "synced", len(repos))
		return nil
	},
}

func init() {
	// The --owner flag is already defined on rootCmd as a persistent flag
	// so it's inherited by all subcommands including sync
}
