// SPDX-FileCopyrightText: 2026 Logan Lindquist Land
// SPDX-License-Identifier: FSL-1.1-MIT

// Package export provides export functionality for repository data.
package export

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/llbbl/repjan/internal/analyze"
	"github.com/llbbl/repjan/internal/github"
)

// ExportData represents the complete export structure.
type ExportData struct {
	ExportedAt   time.Time      `json:"exported_at"`
	Owner        string         `json:"owner"`
	TotalMarked  int            `json:"total_marked"`
	Repositories []ExportedRepo `json:"repositories"`
}

// ExportedRepo represents a single repository in the export.
type ExportedRepo struct {
	Name              string    `json:"name"`
	FullName          string    `json:"full_name"`
	Stars             int       `json:"stars"`
	Forks             int       `json:"forks"`
	DaysSinceActivity int       `json:"days_since_activity"`
	Reason            string    `json:"reason"`
	Language          string    `json:"language"`
	LastPush          time.Time `json:"last_push"`
	IsFork            bool      `json:"is_fork"`
	IsPrivate         bool      `json:"is_private"`
}

// Export writes marked repositories as JSON to a timestamped file.
// Returns the filename on success or an empty string with an error on failure.
func Export(repos []github.Repository, owner string) (string, error) {
	exportedRepos := make([]ExportedRepo, 0, len(repos))

	for _, repo := range repos {
		_, reason := analyze.IsArchiveCandidate(repo)

		exportedRepos = append(exportedRepos, ExportedRepo{
			Name:              repo.Name,
			FullName:          repo.FullName(),
			Stars:             repo.StargazerCount,
			Forks:             repo.ForkCount,
			DaysSinceActivity: repo.DaysSinceActivity,
			Reason:            reason,
			Language:          repo.PrimaryLanguage,
			LastPush:          repo.PushedAt,
			IsFork:            repo.IsFork,
			IsPrivate:         repo.IsPrivate,
		})
	}

	data := ExportData{
		ExportedAt:   time.Now(),
		Owner:        owner,
		TotalMarked:  len(repos),
		Repositories: exportedRepos,
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}

	filename := fmt.Sprintf("archived-repos-%s.json", time.Now().Format("2006-01-02-150405"))

	if err := os.WriteFile(filename, jsonData, 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return filename, nil
}
