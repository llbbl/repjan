// SPDX-FileCopyrightText: 2026 Logan Lindquist Land
// SPDX-License-Identifier: FSL-1.1-MIT

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
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
		// TODO: Will be implemented later
		// 1. If no --owner, get authenticated user
		// 2. Fetch repositories
		// 3. Initialize and run TUI
		fmt.Println("repjan TUI starting...")
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
