// SPDX-FileCopyrightText: 2026 Logan Lindquist Land
// SPDX-License-Identifier: FSL-1.1-MIT

package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/llbbl/repjan/internal/analyze"
	"github.com/llbbl/repjan/internal/github"
)

// Column widths for the table layout.
const (
	colWidthStatus   = 3
	colWidthName     = 25
	colWidthStars    = 7
	colWidthLang     = 12
	colWidthLastPush = 14
	colWidthState    = 10
	colWidthMark     = 5
)

// Status icons for repository states.
const (
	iconActive    = "\u25cf" // ● (filled circle)
	iconCandidate = "\u26a0" // ⚠ (warning)
	iconArchived  = "\u25a1" // □ (empty square)
)

// renderTable renders the repository table with virtual scrolling.
func (m Model) renderTable() string {
	if len(m.filteredRepos) == 0 {
		return m.styles.TableRow.Render("No repositories to display")
	}

	var b strings.Builder

	// Render header row
	b.WriteString(m.styles.TableHeader.Render(buildTableHeader()))
	b.WriteString("\n")

	// Calculate visible rows for virtual scrolling
	// Reserve space for header (2 lines: header + border)
	headerHeight := 2
	availableHeight := m.height - headerHeight
	if availableHeight < 1 {
		availableHeight = 10 // Default minimum
	}

	// Calculate viewport
	startIdx, endIdx := m.calculateViewport(availableHeight)

	// Render visible rows
	for i := startIdx; i < endIdx; i++ {
		if i >= len(m.filteredRepos) {
			break
		}
		repo := m.filteredRepos[i]
		isSelected := i == m.cursor
		b.WriteString(m.renderTableRow(repo, isSelected))
		if i < endIdx-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

// buildTableHeader builds the table header row content string.
func buildTableHeader() string {
	return fmt.Sprintf("%-*s %-*s %*s %-*s %-*s %-*s %-*s",
		colWidthStatus, " ",
		colWidthName, "NAME",
		colWidthStars, "STARS",
		colWidthLang, "LANG",
		colWidthLastPush, "LAST PUSH",
		colWidthState, "STATUS",
		colWidthMark, "MARK",
	)
}

// renderTableRow renders a single table row for a repository.
func (m Model) renderTableRow(repo github.Repository, isSelected bool) string {
	// Get status icon with color
	statusIcon := getStatusIcon(repo)
	statusStyle := m.getStatusStyle(repo)
	styledIcon := statusStyle.Render(statusIcon)

	// Format fields
	name := truncateWithEllipsis(repo.Name, colWidthName)
	stars := fmt.Sprintf("%*d", colWidthStars, repo.StargazerCount)
	lang := truncateWithEllipsis(repo.PrimaryLanguage, colWidthLang)
	lastPush := formatRelativeTime(repo.PushedAt)
	status := getStatusText(repo)

	// Mark indicator
	mark := ""
	repoKey := repo.FullName()
	if m.marked[repoKey] {
		mark = "[\u2713]" // [✓]
	}

	// Build the row content (without status icon, which has its own styling)
	rowContent := fmt.Sprintf(" %-*s %*s %-*s %-*s %-*s %-*s",
		colWidthName, name,
		colWidthStars, stars,
		colWidthLang, lang,
		colWidthLastPush, lastPush,
		colWidthState, status,
		colWidthMark, mark,
	)

	// Apply row styling based on state
	var rowStyle lipgloss.Style
	if isSelected {
		rowStyle = m.styles.SelectedRow
	} else if m.marked[repoKey] {
		rowStyle = m.styles.MarkedRow
	} else {
		rowStyle = m.styles.TableRow
	}

	// Combine styled icon with styled row content
	return styledIcon + rowStyle.Render(rowContent)
}

// calculateViewport calculates the start and end indices for visible rows.
func (m Model) calculateViewport(visibleRows int) (startIdx, endIdx int) {
	totalRows := len(m.filteredRepos)

	if totalRows <= visibleRows {
		// All rows fit on screen
		return 0, totalRows
	}

	// Keep cursor visible by adjusting viewport
	// Center cursor in viewport when possible
	halfVisible := visibleRows / 2

	startIdx = m.cursor - halfVisible
	if startIdx < 0 {
		startIdx = 0
	}

	endIdx = startIdx + visibleRows
	if endIdx > totalRows {
		endIdx = totalRows
		startIdx = endIdx - visibleRows
		if startIdx < 0 {
			startIdx = 0
		}
	}

	return startIdx, endIdx
}

// getStatusStyle returns the appropriate style for a repository's status.
func (m Model) getStatusStyle(repo github.Repository) lipgloss.Style {
	if repo.IsArchived {
		return m.styles.StatusArchived
	}
	isCandidate, _ := analyze.IsArchiveCandidate(repo)
	if isCandidate {
		return m.styles.StatusCandidate
	}
	return m.styles.StatusActive
}

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
