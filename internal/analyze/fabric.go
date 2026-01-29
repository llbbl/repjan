// SPDX-FileCopyrightText: 2026 Logan Lindquist Land
// SPDX-License-Identifier: FSL-1.1-MIT

package analyze

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/llbbl/repjan/internal/github"
)

// CheckFabricAvailable checks if the fabric CLI is available in PATH.
func CheckFabricAvailable(fabricPath string) bool {
	if fabricPath == "" {
		fabricPath = "fabric"
	}
	_, err := exec.LookPath(fabricPath)
	return err == nil
}

// AnalyzeRepo runs fabric analysis on a single repository.
func AnalyzeRepo(repo github.Repository, readme string, fabricPath string) (string, error) {
	if fabricPath == "" {
		fabricPath = "fabric"
	}

	// Build context
	repoContext := fmt.Sprintf(`Repository: %s
Description: %s
Language: %s
Stars: %d, Forks: %d
Last Activity: %d days ago

README:
%s`,
		repo.FullName(),
		repo.Description,
		repo.PrimaryLanguage,
		repo.StargazerCount,
		repo.ForkCount,
		repo.DaysSinceActivity,
		truncateReadme(readme, 2000),
	)

	// Execute fabric with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, fabricPath, "--pattern", "analyze-repo")
	cmd.Stdin = strings.NewReader(repoContext)

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("fabric analysis failed: %w", err)
	}

	return string(output), nil
}

// AnalyzeRepos runs fabric analysis on multiple repositories.
func AnalyzeRepos(repos []github.Repository, fabricPath string) (string, error) {
	if fabricPath == "" {
		fabricPath = "fabric"
	}

	if len(repos) == 0 {
		return "", fmt.Errorf("no repositories provided for analysis")
	}

	// Build combined context for all repos
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("Analyzing %d repositories:\n\n", len(repos)))

	for i, repo := range repos {
		builder.WriteString(fmt.Sprintf("--- Repository %d ---\n", i+1))
		builder.WriteString(fmt.Sprintf("Name: %s\n", repo.FullName()))
		builder.WriteString(fmt.Sprintf("Description: %s\n", repo.Description))
		builder.WriteString(fmt.Sprintf("Language: %s\n", repo.PrimaryLanguage))
		builder.WriteString(fmt.Sprintf("Stars: %d, Forks: %d\n", repo.StargazerCount, repo.ForkCount))
		builder.WriteString(fmt.Sprintf("Last Activity: %d days ago\n", repo.DaysSinceActivity))
		builder.WriteString("\n")
	}

	// Execute fabric with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, fabricPath, "--pattern", "analyze-repo")
	cmd.Stdin = strings.NewReader(builder.String())

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("fabric batch analysis failed: %w", err)
	}

	return string(output), nil
}

// truncateReadme truncates a README to the specified maximum length.
func truncateReadme(readme string, maxLen int) string {
	if len(readme) <= maxLen {
		return readme
	}
	return readme[:maxLen] + "\n... [truncated]"
}
