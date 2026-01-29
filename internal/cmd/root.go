// SPDX-FileCopyrightText: 2026 Logan Lindquist Land
// SPDX-License-Identifier: FSL-1.1-MIT

package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/llbbl/repjan/internal/github"
	"github.com/llbbl/repjan/internal/tui"
)

// Version is set at build time with -ldflags
var Version = "dev"

// Flag variables
var (
	owner      string
	fabric     bool
	fabricPath string
)

var rootCmd = &cobra.Command{
	Use:   "repjan",
	Short: "Repository janitor - manage GitHub repos at scale",
	Long: `repjan is a TUI tool for auditing and archiving GitHub repositories.
It helps identify inactive repos and batch archive them.`,
	RunE: func(cmd *cobra.Command, args []string) error {
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

		// Fetch repositories
		fmt.Fprintf(os.Stderr, "Fetching repositories for %s...\n", targetOwner)
		repos, err := client.FetchRepositories(targetOwner)
		if err != nil {
			return fmt.Errorf("failed to fetch repositories: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Found %d repositories\n", len(repos))

		// Initialize TUI model
		model := tui.NewModel(repos, targetOwner, client, fabric, fabricPath)

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
	// Define flags on root command
	rootCmd.PersistentFlags().StringVarP(&owner, "owner", "o", "", "GitHub username or org to audit")
	rootCmd.PersistentFlags().BoolVarP(&fabric, "fabric", "f", false, "Enable Fabric AI integration")
	rootCmd.PersistentFlags().StringVar(&fabricPath, "fabric-path", "fabric", "Custom path to Fabric binary")

	// Add subcommands
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(dbCmd)
	rootCmd.AddCommand(syncCmd)
}

// Execute runs the root command.
func Execute() error {
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
