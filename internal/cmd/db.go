// SPDX-FileCopyrightText: 2026 Logan Lindquist Land
// SPDX-License-Identifier: FSL-1.1-MIT

package cmd

import (
	"bufio"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/llbbl/repjan/internal/db"
)

var (
	forceReset bool
)

var dbCmd = &cobra.Command{
	Use:   "db",
	Short: "Database management commands",
	Long:  `Commands for managing the repjan SQLite database.`,
}

var dbMigrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Run pending database migrations",
	Long:  `Run all pending database migrations to update the schema.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dbPath, err := db.GetDefaultDBPath()
		if err != nil {
			slog.Error("failed to get database path", "component", "cmd", "error", err)
			return fmt.Errorf("getting database path: %w", err)
		}

		slog.Debug("opening database for migration", "component", "cmd", "path", dbPath)
		database, err := db.Open(dbPath)
		if err != nil {
			slog.Error("failed to open database", "component", "cmd", "path", dbPath, "error", err)
			return fmt.Errorf("opening database: %w", err)
		}
		defer db.Close(database)

		// Get version before migration
		versionBefore, _ := db.GetMigrationVersion(database)
		slog.Debug("current migration version", "component", "cmd", "version", versionBefore)

		if err := db.RunMigrations(database); err != nil {
			slog.Error("migration failed", "component", "cmd", "error", err)
			return fmt.Errorf("running migrations: %w", err)
		}

		versionAfter, err := db.GetMigrationVersion(database)
		if err != nil {
			slog.Error("failed to get migration version", "component", "cmd", "error", err)
			return fmt.Errorf("getting migration version: %w", err)
		}

		if versionBefore == versionAfter {
			fmt.Printf("Database is already at version %d (no migrations needed)\n", versionAfter)
			slog.Debug("no migrations needed", "component", "cmd", "version", versionAfter)
		} else {
			fmt.Printf("Migrations complete: version %d -> %d\n", versionBefore, versionAfter)
			slog.Debug("migrations completed", "component", "cmd", "version_before", versionBefore, "version_after", versionAfter)
		}

		return nil
	},
}

var dbStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show database status and statistics",
	Long:  `Display database location, migration version, and repository statistics.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dbPath, err := db.GetDefaultDBPath()
		if err != nil {
			slog.Error("failed to get database path", "component", "cmd", "error", err)
			return fmt.Errorf("getting database path: %w", err)
		}

		slog.Debug("checking database status", "component", "cmd", "path", dbPath)
		fmt.Printf("Database path: %s\n", dbPath)

		// Check if database file exists
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			slog.Debug("database does not exist", "component", "cmd", "path", dbPath)
			fmt.Println("Status: Database does not exist (run 'repjan db migrate' to create)")
			return nil
		}

		database, err := db.Open(dbPath)
		if err != nil {
			slog.Error("failed to open database", "component", "cmd", "path", dbPath, "error", err)
			return fmt.Errorf("opening database: %w", err)
		}
		defer db.Close(database)

		// Get migration version
		version, err := db.GetMigrationVersion(database)
		if err != nil {
			slog.Error("failed to get migration version", "component", "cmd", "error", err)
			fmt.Printf("Migration version: unknown (error: %v)\n", err)
		} else {
			fmt.Printf("Migration version: %d\n", version)
		}

		// Get repository count
		repoCount, err := getRepositoryCount(database)
		if err != nil {
			slog.Error("failed to get repository count", "component", "cmd", "error", err)
			fmt.Printf("Repository count: unknown (error: %v)\n", err)
		} else {
			fmt.Printf("Repository count: %d\n", repoCount)
		}

		// Get last sync time
		lastSync, err := getLastSyncTime(database)
		if err != nil {
			slog.Error("failed to get last sync time", "component", "cmd", "error", err)
			fmt.Printf("Last sync: unknown (error: %v)\n", err)
		} else if lastSync == "" {
			fmt.Println("Last sync: never")
		} else {
			fmt.Printf("Last sync: %s\n", lastSync)
		}

		slog.Debug("status check complete", "component", "cmd", "version", version, "repo_count", repoCount)
		return nil
	},
}

var dbPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Print database file location",
	Long:  `Print the path to the repjan database file.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dbPath, err := db.GetDefaultDBPath()
		if err != nil {
			slog.Error("failed to get database path", "component", "cmd", "error", err)
			return fmt.Errorf("getting database path: %w", err)
		}
		slog.Debug("resolved database path", "component", "cmd", "path", dbPath)
		fmt.Println(dbPath)
		return nil
	},
}

var dbResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset database (destructive)",
	Long: `Delete the database file and recreate it with fresh migrations.
This is a destructive operation that will delete all stored data.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dbPath, err := db.GetDefaultDBPath()
		if err != nil {
			slog.Error("failed to get database path", "component", "cmd", "error", err)
			return fmt.Errorf("getting database path: %w", err)
		}

		slog.Debug("initiating database reset", "component", "cmd", "path", dbPath, "force", forceReset)

		// Check if database exists
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			slog.Debug("database does not exist, will create fresh", "component", "cmd", "path", dbPath)
			fmt.Println("Database does not exist, creating fresh database...")
		} else {
			// Require confirmation
			if !forceReset {
				fmt.Printf("WARNING: This will delete all data in %s\n", dbPath)
				fmt.Print("Type 'yes' to confirm: ")

				reader := bufio.NewReader(os.Stdin)
				confirmation, err := reader.ReadString('\n')
				if err != nil {
					slog.Error("failed to read confirmation", "component", "cmd", "error", err)
					return fmt.Errorf("reading confirmation: %w", err)
				}

				if strings.TrimSpace(strings.ToLower(confirmation)) != "yes" {
					slog.Debug("reset aborted by user", "component", "cmd")
					fmt.Println("Aborted.")
					return nil
				}
			}

			// Delete the database file
			slog.Debug("deleting database file", "component", "cmd", "path", dbPath)
			if err := os.Remove(dbPath); err != nil {
				slog.Error("failed to delete database", "component", "cmd", "path", dbPath, "error", err)
				return fmt.Errorf("deleting database: %w", err)
			}
			fmt.Printf("Deleted: %s\n", dbPath)
		}

		// Create fresh database with migrations
		slog.Debug("creating fresh database", "component", "cmd", "path", dbPath)
		database, err := db.Open(dbPath)
		if err != nil {
			slog.Error("failed to create database", "component", "cmd", "path", dbPath, "error", err)
			return fmt.Errorf("creating database: %w", err)
		}
		defer db.Close(database)

		if err := db.RunMigrations(database); err != nil {
			slog.Error("migration failed", "component", "cmd", "error", err)
			return fmt.Errorf("running migrations: %w", err)
		}

		version, err := db.GetMigrationVersion(database)
		if err != nil {
			slog.Error("failed to get migration version", "component", "cmd", "error", err)
			return fmt.Errorf("getting migration version: %w", err)
		}

		fmt.Printf("Created fresh database at version %d\n", version)
		slog.Debug("database reset complete", "component", "cmd", "version", version)
		return nil
	},
}

func init() {
	// Add flags
	dbResetCmd.Flags().BoolVar(&forceReset, "force", false, "Skip confirmation prompt")

	// Add subcommands to db command
	dbCmd.AddCommand(dbMigrateCmd)
	dbCmd.AddCommand(dbStatusCmd)
	dbCmd.AddCommand(dbPathCmd)
	dbCmd.AddCommand(dbResetCmd)
}

// getRepositoryCount returns the number of repositories in the database.
func getRepositoryCount(database *sql.DB) (int, error) {
	var count int
	err := database.QueryRow("SELECT COUNT(*) FROM repositories").Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// getLastSyncTime returns the most recent synced_at time from the repositories table.
func getLastSyncTime(database *sql.DB) (string, error) {
	var lastSync sql.NullString
	err := database.QueryRow("SELECT MAX(synced_at) FROM repositories").Scan(&lastSync)
	if err != nil {
		return "", err
	}
	if !lastSync.Valid {
		return "", nil
	}
	return lastSync.String, nil
}
