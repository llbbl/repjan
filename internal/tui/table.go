// SPDX-FileCopyrightText: 2026 Logan Lindquist Land
// SPDX-License-Identifier: FSL-1.1-MIT

package tui

import (
	"fmt"
	"time"

	"github.com/llbbl/repjan/internal/analyze"
	"github.com/llbbl/repjan/internal/github"
)

// Status icons for repository states.
const (
	iconActive    = "●" // ● (filled circle)
	iconCandidate = "⚠" // ⚠ (warning)
	iconArchived  = "□" // □ (empty square)
)

// formatRelativeTime formats a time.Time as a relative time string (e.g., "3 days ago").
func formatRelativeTime(t time.Time) string {
	if t.IsZero() {
		return "unknown"
	}

	duration := time.Since(t)
	days := int(duration.Hours() / 24)

	if days < 0 {
		return "in the future"
	}
	if days == 0 {
		return "today"
	}
	if days == 1 {
		return "1 day ago"
	}
	return fmt.Sprintf("%d days ago", days)
}

// truncateWithEllipsis truncates a string to the specified maximum length,
// adding ellipsis if truncation occurs. This is an internal helper for table rendering.
func truncateWithEllipsis(s string, max int) string {
	if max < 4 {
		max = 4
	}
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

// getStatusIcon returns the appropriate status icon for a repository.
// Returns ● for active, ⚠ for archive candidate, □ for archived.
func getStatusIcon(repo github.Repository) string {
	if repo.IsArchived {
		return iconArchived
	}
	isCandidate, _ := analyze.IsArchiveCandidate(repo)
	if isCandidate {
		return iconCandidate
	}
	return iconActive
}

// getStatusText returns the status text for a repository.
// Returns "Active", "Candidate", or "Archived".
func getStatusText(repo github.Repository) string {
	if repo.IsArchived {
		return "Archived"
	}
	isCandidate, _ := analyze.IsArchiveCandidate(repo)
	if isCandidate {
		return "Candidate"
	}
	return "Active"
}
