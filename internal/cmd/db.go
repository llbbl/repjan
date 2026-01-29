// SPDX-FileCopyrightText: 2026 Logan Lindquist Land
// SPDX-License-Identifier: FSL-1.1-MIT

package cmd

import (
	"bufio"
	"database/sql"
	"fmt"
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
			return fmt.Errorf("getting database path: %w", err)
		}

		database, err := db.Open(dbPath)
		if err != nil {
			return fmt.Errorf("opening database: %w", err)
		}
		defer db.Close(database)

		// Get version before migration
		versionBefore, _ := db.GetMigrationVersion(database)

		if err := db.RunMigrations(database); err != nil {
			return fmt.Errorf("running migrations: %w", err)
		}

		versionAfter, err := db.GetMigrationVersion(database)
		if err != nil {
			return fmt.Errorf("getting migration version: %w", err)
		}

		if versionBefore == versionAfter {
			fmt.Printf("Database is already at version %d (no migrations needed)\n", versionAfter)
		} else {
			fmt.Printf("Migrations complete: version %d -> %d\n", versionBefore, versionAfter)
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
			return fmt.Errorf("getting database path: %w", err)
		}

		fmt.Printf("Database path: %s\n", dbPath)

		// Check if database file exists
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			fmt.Println("Status: Database does not exist (run 'repjan db migrate' to create)")
			return nil
		}

		database, err := db.Open(dbPath)
		if err != nil {
			return fmt.Errorf("opening database: %w", err)
		}
		defer db.Close(database)

		// Get migration version
		version, err := db.GetMigrationVersion(database)
		if err != nil {
			fmt.Printf("Migration version: unknown (error: %v)\n", err)
		} else {
			fmt.Printf("Migration version: %d\n", version)
		}

		// Get repository count
		repoCount, err := getRepositoryCount(database)
		if err != nil {
			fmt.Printf("Repository count: unknown (error: %v)\n", err)
		} else {
			fmt.Printf("Repository count: %d\n", repoCount)
		}

		// Get last sync time
		lastSync, err := getLastSyncTime(database)
		if err != nil {
			fmt.Printf("Last sync: unknown (error: %v)\n", err)
		} else if lastSync == "" {
			fmt.Println("Last sync: never")
		} else {
			fmt.Printf("Last sync: %s\n", lastSync)
		}

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
			return fmt.Errorf("getting database path: %w", err)
		}
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
			return fmt.Errorf("getting database path: %w", err)
		}

		// Check if database exists
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			fmt.Println("Database does not exist, creating fresh database...")
		} else {
			// Require confirmation
			if !forceReset {
				fmt.Printf("WARNING: This will delete all data in %s\n", dbPath)
				fmt.Print("Type 'yes' to confirm: ")

				reader := bufio.NewReader(os.Stdin)
				confirmation, err := reader.ReadString('\n')
				if err != nil {
					return fmt.Errorf("reading confirmation: %w", err)
				}

				if strings.TrimSpace(strings.ToLower(confirmation)) != "yes" {
					fmt.Println("Aborted.")
					return nil
				}
			}

			// Delete the database file
			if err := os.Remove(dbPath); err != nil {
				return fmt.Errorf("deleting database: %w", err)
			}
			fmt.Printf("Deleted: %s\n", dbPath)
		}

		// Create fresh database with migrations
		database, err := db.Open(dbPath)
		if err != nil {
			return fmt.Errorf("creating database: %w", err)
		}
		defer db.Close(database)

		if err := db.RunMigrations(database); err != nil {
			return fmt.Errorf("running migrations: %w", err)
		}

		version, err := db.GetMigrationVersion(database)
		if err != nil {
			return fmt.Errorf("getting migration version: %w", err)
		}

		fmt.Printf("Created fresh database at version %d\n", version)
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
