// SPDX-FileCopyrightText: 2026 Logan Lindquist Land
// SPDX-License-Identifier: FSL-1.1-MIT

package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// View implements tea.Model and renders the complete TUI.
func (m Model) View() string {
	// Handle loading state
	if m.loading {
		return m.styles.HeaderInfo.Render("Loading repositories...")
	}

	// Handle archiving state - show full UI with progress indicator instead of blank screen
	// The progress indicator will be shown in the status bar

	// Build main view
	// Note: renderSortBar() includes header, warning banner, and filter info
	// as a workaround for rendering issues with separate sections
	sections := []string{
		m.renderSortBar(),
		m.renderTableHeader(),
		m.renderTableBody(),
		m.renderFooter(),
	}

	// Add error message if present
	if m.lastError != nil {
		errMsg := m.styles.Error.Render(fmt.Sprintf("Error: %s", m.lastError.Error()))
		sections = append([]string{errMsg}, sections...)
	}

	// Add search input if in search mode
	if m.searchMode {
		matchCount := len(m.filteredRepos)
		searchBar := m.styles.FilterBar.Render(
			fmt.Sprintf("/ %s (%d matches)_", m.searchQuery, matchCount),
		)
		sections = append([]string{sections[0], searchBar}, sections[1:]...)
	}

	// Join with explicit newlines to ensure all sections show
	view := strings.Join(sections, "\n")

	// Render modal overlay if active
	if m.activeModal != ModalNone {
		var modalContent string
		switch m.activeModal {
		case ModalDetail:
			modalContent = m.renderDetailModal()
		case ModalConfirm:
			modalContent = m.renderConfirmModal()
		case ModalHelp:
			modalContent = m.renderHelpModal()
		case ModalLanguage:
			modalContent = m.renderLanguageModal()
		default:
			modalContent = m.styles.ModalBorder.Render("Unknown modal")
		}
		view = lipgloss.JoinVertical(lipgloss.Left, view, modalContent)
	}

	return view
}

// getVisibilityLabel returns a human-readable label for the current visibility state.
func (m Model) getVisibilityLabel() string {
	switch {
	case !m.showPrivate && !m.showArchived:
		return "Public Active"
	case !m.showPrivate && m.showArchived:
		return "Public All"
	case m.showPrivate && !m.showArchived:
		return "Including Private"
	default: // m.showPrivate && m.showArchived
		return "All Repos"
	}
}

// renderSortBar renders the sort bar with available sort options.
// Also includes header, filter, and warning info since separate sections weren't rendering
func (m Model) renderSortBar() string {
	markedCount := len(m.marked)
	totalCount := len(m.filteredRepos)

	// Build header line
	headerLine := fmt.Sprintf("repjan - %s (%d repos, %d marked)  [? Help]", m.owner, totalCount, markedCount)

	// Build warning line if private repos visible - with prominent styling
	warningLine := ""
	if m.showPrivate {
		warningText := "  ⚠️  PRIVATE REPOS VISIBLE - Screenshots may expose sensitive data  ⚠️  "
		warningLine = m.styles.PrivateWarningBanner.Width(m.width).Render(warningText) + "\n"
	}

	// Build filter line
	filterNames := map[Filter]string{FilterAll: "All", FilterOld: "Old", FilterNoStars: "NoStars", FilterForks: "Forks"}
	privateStr := "[P]rivate"
	if m.showPrivate {
		// Style the PRIVATE indicator with bold red to make it obvious
		privateStr = m.styles.PrivateIndicator.Render("[P]+PRIVATE!")
	}
	archivedStr := "[X]Archived"
	if m.showArchived {
		// Style with light blue to indicate archived repos are visible
		archivedStr = lipgloss.NewStyle().Foreground(lipgloss.Color("#87CEEB")).Bold(true).Render("[X]+Archived")
	}
	filterLine := fmt.Sprintf("%s | Filter: %s | %s %s", m.getVisibilityLabel(), filterNames[m.currentFilter], privateStr, archivedStr)

	sortOptions := []struct {
		key   string
		label string
		field SortField
	}{
		{"1", "Name", SortName},
		{"2", "Activity", SortActivity},
		{"3", "Stars", SortStars},
		{"4", "Language", SortLanguage},
	}

	var parts []string
	parts = append(parts, "Sort: ")

	for i, s := range sortOptions {
		var rendered string
		keyLabel := fmt.Sprintf("[%s]%s", s.key, s.label)

		if m.sortField == s.field {
			// Add direction indicator
			arrow := "↑"
			if !m.sortAscending {
				arrow = "↓"
			}
			keyLabel = fmt.Sprintf("[%s]%s %s", s.key, s.label, arrow)
			rendered = m.styles.ActiveFilter.Render(keyLabel)
		} else {
			rendered = m.styles.HelpDesc.Render(keyLabel)
		}

		parts = append(parts, rendered)
		if i < len(sortOptions)-1 {
			parts = append(parts, " ")
		}
	}

	sortLine := strings.Join(parts, "")

	// Combine all lines: header, warning (if any), filter, sort
	return headerLine + "\n" + warningLine + filterLine + "\n" + sortLine
}

// renderTableHeader renders the table column headers.
func (m Model) renderTableHeader() string {
	// Column widths (approximate percentages of terminal width)
	nameWidth := 30
	starsWidth := 8
	langWidth := 12
	pushWidth := 12
	statusWidth := 10
	markWidth := 6

	header := fmt.Sprintf("%-*s | %*s | %-*s | %-*s | %-*s | %-*s",
		nameWidth, "NAME",
		starsWidth, "STARS",
		langWidth, "LANG",
		pushWidth, "LAST PUSH",
		statusWidth, "STATUS",
		markWidth, "MARK",
	)

	return m.styles.TableHeader.Width(m.width).Render(header)
}

// renderTableBody renders the repository table body.
func (m Model) renderTableBody() string {
	if len(m.filteredRepos) == 0 {
		return m.styles.HeaderInfo.Render("No repositories to display")
	}

	// Calculate visible rows based on height using the package-level constant
	visibleRows := m.height - reservedRows
	if visibleRows < 1 {
		visibleRows = 5
	}

	// Calculate start and end indices based on viewport offset
	startIdx := m.viewportOffset
	endIdx := min(startIdx+visibleRows, len(m.filteredRepos))

	var rows []string
	for i := startIdx; i < endIdx; i++ {
		repo := m.filteredRepos[i]

		// Determine row style
		style := m.styles.TableRow
		if i == m.cursor {
			style = m.styles.SelectedRow
		}

		repoKey := fmt.Sprintf("%s/%s", repo.Owner, repo.Name)
		if m.marked[repoKey] {
			// When archiving is in progress, show archiving style for marked repos
			if m.archiving {
				if i == m.cursor {
					// Combine selected and archiving styling
					style = m.styles.ArchivingRow.Background(ColorPrimary)
				} else {
					style = m.styles.ArchivingRow
				}
			} else if i == m.cursor {
				// Combine selected and marked styling
				style = m.styles.SelectedRow.Foreground(ColorMarked)
			} else {
				style = m.styles.MarkedRow
			}
		}

		// Format row - placeholder values for now
		markIndicator := " "
		if m.marked[repoKey] {
			markIndicator = "*"
		}

		// Get language, defaulting to empty if not set
		lang := repo.PrimaryLanguage
		if lang == "" {
			lang = "-"
		}

		// Format last push date
		lastPush := "-"
		if !repo.PushedAt.IsZero() {
			lastPush = repo.PushedAt.Format("2006-01-02")
		}

		// Determine status based on repo state
		status := "active"
		if repo.IsArchived {
			status = "archived"
		}

		row := fmt.Sprintf("%-30s | %8d | %-12s | %-12s | %-10s | %-6s",
			truncateString(repo.Name, 30),
			repo.StargazerCount,
			truncateString(lang, 12),
			lastPush,
			status,
			markIndicator,
		)

		rows = append(rows, style.Width(m.width).Render(row))
	}

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

// renderFooter renders the footer with keybinding hints and sync status.
func (m Model) renderFooter() string {
	bindings := []struct {
		key  string
		desc string
	}{
		{"j/k", "navigate"},
		{"pgup/pgdn", "page"},
		{"space", "mark"},
		{"enter", "details"},
		{"/", "search"},
		{"p", "private"},
		{"x", "archived"},
		{"a", "archive marked"},
		{"q", "quit"},
	}

	var parts []string
	for i, b := range bindings {
		key := m.styles.HelpKey.Render(b.key)
		desc := m.styles.HelpDesc.Render(b.desc)
		parts = append(parts, fmt.Sprintf("%s %s", key, desc))
		if i < len(bindings)-1 {
			parts = append(parts, m.styles.HelpDesc.Render(" | "))
		}
	}

	keybindings := strings.Join(parts, "")

	// Build status bar with sync info
	statusBar := m.renderStatusBar()

	// Combine keybindings and status bar
	footer := lipgloss.JoinVertical(lipgloss.Left,
		m.styles.FilterBar.Render(keybindings),
		statusBar,
	)

	return footer
}

// renderStatusBar renders the status bar showing sync status and messages.
func (m Model) renderStatusBar() string {
	var parts []string

	// Show archiving/unarchiving progress if in progress (takes priority)
	if m.archiving {
		action := "Archiving"
		if m.archiveMode == "unarchive" {
			action = "Unarchiving"
		}
		archiveStatus := fmt.Sprintf("%s %d/%d repositories...", action, m.archiveProgress, m.archiveTotal)
		parts = append(parts, m.styles.Warning.Render(archiveStatus))
	} else if m.syncing {
		// Show syncing indicator
		parts = append(parts, m.styles.HelpKey.Render("Syncing..."))
	} else {
		// Show last sync time
		syncStatus := m.formatSyncStatus()
		parts = append(parts, m.styles.HelpDesc.Render(syncStatus))
	}

	// Show cached data warning if applicable
	if m.usingCache {
		parts = append(parts, m.styles.Error.Render(" (cached data)"))
	}

	// Show status message if present
	if m.statusMessage != "" {
		parts = append(parts, m.styles.HelpDesc.Render(" | "))
		parts = append(parts, m.styles.HelpKey.Render(m.statusMessage))
	}

	return m.styles.FilterBar.Render(strings.Join(parts, ""))
}

// formatSyncStatus formats the last sync time into a human-readable string.
func (m Model) formatSyncStatus() string {
	if m.lastSyncTime.IsZero() {
		return "Not synced"
	}

	elapsed := time.Since(m.lastSyncTime)

	if elapsed < time.Minute {
		return "Last synced: just now"
	} else if elapsed < time.Hour {
		mins := int(elapsed.Minutes())
		if mins == 1 {
			return "Last synced: 1 minute ago"
		}
		return fmt.Sprintf("Last synced: %d minutes ago", mins)
	} else if elapsed < 24*time.Hour {
		hours := int(elapsed.Hours())
		if hours == 1 {
			return "Last synced: 1 hour ago"
		}
		return fmt.Sprintf("Last synced: %d hours ago", hours)
	}

	// For older syncs, show the date/time
	return fmt.Sprintf("Last synced: %s", m.lastSyncTime.Format("Jan 2 15:04"))
}

// truncateString truncates a string to the specified length, adding "..." if truncated.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
