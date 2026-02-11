// SPDX-FileCopyrightText: 2026 Logan Lindquist Land
// SPDX-License-Identifier: FSL-1.1-MIT

package cmd

import (
	"fmt"
	"log/slog"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/llbbl/repjan/internal/config"
	"github.com/llbbl/repjan/internal/db"
	"github.com/llbbl/repjan/internal/github"
	"github.com/llbbl/repjan/internal/logging"
	"github.com/llbbl/repjan/internal/store"
	"github.com/llbbl/repjan/internal/sync"
	"github.com/llbbl/repjan/internal/tui"
)

// Version is set at build time with -ldflags
var Version = "dev"

// cfg holds the loaded configuration
var cfg *config.Config

// Flag variables
var (
	owner        string
	fabric       bool
	fabricPath   string
	syncInterval time.Duration
	logLevel     string
	logFormat    string
)

var rootCmd = &cobra.Command{
	Use:   "repjan",
	Short: "Repository janitor - manage GitHub repos at scale",
	Long: `repjan is a TUI tool for auditing and archiving GitHub repositories.
It helps identify inactive repos and batch archive them.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Resolve effective sync interval (CLI flag overrides config)
		effectiveSyncInterval := cfg.SyncInterval
		if syncInterval > 0 {
			effectiveSyncInterval = syncInterval
		}

		// Create GitHub client
		client := github.NewDefaultClient()

		// Determine owner
		targetOwner := owner
		if targetOwner == "" {
			user, err := client.GetAuthenticatedUser()
			if err != nil {
				return fmt.Errorf("failed to get authenticated user: %w\nMake sure you're logged in with 'gh auth login'", err)
			}
			targetOwner = user
		}

		// Open database (use config path if set, otherwise use default)
		var dbPath string
		if cfg.DBPath != "" {
			dbPath = cfg.DBPath
		} else {
			var err error
			dbPath, err = db.GetDefaultDBPath()
			if err != nil {
				return fmt.Errorf("getting database path: %w", err)
			}
		}

		database, err := db.Open(dbPath)
		if err != nil {
			return fmt.Errorf("opening database: %w", err)
		}
		defer db.Close(database)

		// Ensure migrations are run
		if err := db.RunMigrations(database); err != nil {
			return fmt.Errorf("running migrations: %w", err)
		}

		// Create store
		repoStore := store.New(database)

		// Try to load cached data first
		var repos []github.Repository
		var lastSyncTime time.Time
		var usingCache bool

		lastSyncTime, _ = repoStore.GetLastSyncTime(targetOwner)
		cachedRepos, cacheErr := repoStore.GetRepositories(targetOwner)

		// If we have cached data and it's recent (within sync interval), use it
		if cacheErr == nil && len(cachedRepos) > 0 && !lastSyncTime.IsZero() && time.Since(lastSyncTime) < effectiveSyncInterval {
			slog.Info("loading from cache", "last_synced_ago", time.Since(lastSyncTime).Round(time.Second))
			repos = cachedRepos
			usingCache = true
		} else {
			// Fetch fresh data from GitHub
			slog.Info("fetching repositories", "owner", targetOwner, "source", "github")
			freshRepos, fetchErr := client.FetchRepositories(targetOwner)
			if fetchErr != nil {
				// If fetch fails but we have cached data, use it with a warning
				if cacheErr == nil && len(cachedRepos) > 0 {
					slog.Warn("github fetch failed, using cached data", "error", fetchErr, "last_synced_ago", time.Since(lastSyncTime).Round(time.Second))
					repos = cachedRepos
					usingCache = true
				} else {
					return fmt.Errorf("failed to fetch repositories: %w", fetchErr)
				}
			} else {
				repos = freshRepos
				lastSyncTime = time.Now()
				slog.Info("found repositories", "count", len(repos))

				// Upsert fresh repos to database
				if err := repoStore.UpsertRepositories(targetOwner, repos); err != nil {
					return fmt.Errorf("storing repositories: %w", err)
				}
			}
		}

		// Create and start background syncer
		syncer := sync.New(repoStore, client, targetOwner, effectiveSyncInterval)
		syncCh := syncer.Start()
		defer syncer.Stop()

		// Initialize TUI model with store and sync channel
		model := tui.NewModelWithOptions(repos, targetOwner, client, repoStore, fabric, fabricPath, lastSyncTime, usingCache, syncCh)

		// Load marked repos from database
		if err := model.LoadMarkedRepos(); err != nil {
			slog.Warn("failed to load marked repos", "error", err)
		}

		// Run the TUI
		p := tea.NewProgram(model, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("error running TUI: %w", err)
		}

		return nil
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("repjan version %s\n", Version)
	},
}

func init() {
	// Enable --version flag
	rootCmd.Version = Version

	// Define flags on root command
	rootCmd.PersistentFlags().StringVarP(&owner, "owner", "o", "", "GitHub username or org to audit")
	rootCmd.PersistentFlags().BoolVarP(&fabric, "fabric", "f", false, "Enable Fabric AI integration")
	rootCmd.PersistentFlags().StringVar(&fabricPath, "fabric-path", "fabric", "Custom path to Fabric binary")
	rootCmd.PersistentFlags().DurationVar(&syncInterval, "sync-interval", 0, "Interval for background repository sync (overrides env)")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "", "Log level: debug, info, warn, error (overrides env)")
	rootCmd.PersistentFlags().StringVar(&logFormat, "log-format", "", "Log format: text, json (overrides env)")

	// Add subcommands
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(dbCmd)
	rootCmd.AddCommand(syncCmd)
}

// Execute runs the root command.
func Execute() error {
	// Load configuration from env vars and .env file
	var err error
	cfg, err = config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Setup structured logging with configured level and format
	// CLI flags override config values if provided
	effectiveLogLevel := cfg.LogLevel
	if logLevel != "" {
		effectiveLogLevel = logLevel
	}
	effectiveLogFormat := cfg.LogFormat
	if logFormat != "" {
		effectiveLogFormat = logFormat
	}
	logging.SetupLogger(effectiveLogLevel, effectiveLogFormat)

	return rootCmd.Execute()
}

// GetOwner returns the --owner flag value.
func GetOwner() string {
	return owner
}

// IsFabricEnabled returns the --fabric flag value.
func IsFabricEnabled() bool {
	return fabric
}

// GetFabricPath returns the --fabric-path flag value.
func GetFabricPath() string {
	return fabricPath
}

// GetConfig returns the loaded configuration.
func GetConfig() *config.Config {
	return cfg
}
