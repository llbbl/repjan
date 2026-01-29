// SPDX-FileCopyrightText: 2026 Logan Lindquist Land
// SPDX-License-Identifier: FSL-1.1-MIT

package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// View implements tea.Model and renders the complete TUI.
func (m Model) View() string {
	// Handle loading state
	if m.loading {
		return m.styles.HeaderInfo.Render("Loading repositories...")
	}

	// Handle archiving state
	if m.archiving {
		return m.styles.HeaderInfo.Render(
			fmt.Sprintf("Archiving %d/%d...", m.archiveProgress, m.archiveTotal),
		)
	}

	// Build main view
	sections := []string{
		m.renderHeader(),
		m.renderFilters(),
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
		searchBar := m.styles.FilterBar.Render(
			fmt.Sprintf("Search: %s_", m.searchQuery),
		)
		sections = append([]string{sections[0], searchBar}, sections[1:]...)
	}

	view := lipgloss.JoinVertical(lipgloss.Left, sections...)

	// TODO: Modal overlay will be added here
	if m.activeModal != ModalNone {
		// Placeholder for modal rendering
		view = lipgloss.JoinVertical(lipgloss.Left,
			view,
			m.styles.ModalBorder.Render("Modal placeholder"),
		)
	}

	return view
}

// renderHeader renders the header bar with title and stats.
func (m Model) renderHeader() string {
	markedCount := len(m.marked)
	totalCount := len(m.filteredRepos)

	title := m.styles.HeaderTitle.Render("repjan")
	info := m.styles.HeaderInfo.Render(
		fmt.Sprintf(" - %s (%d repositories, %d marked)", m.owner, totalCount, markedCount),
	)
	helpHint := m.styles.HelpKey.Render("[? Help]")

	headerContent := lipgloss.JoinHorizontal(lipgloss.Center, title, info)

	// Add spacing between header content and help hint
	availableWidth := m.width - lipgloss.Width(headerContent) - lipgloss.Width(helpHint) - 4
	if availableWidth < 0 {
		availableWidth = 1
	}
	spacing := strings.Repeat(" ", availableWidth)

	fullHeader := lipgloss.JoinHorizontal(lipgloss.Center, headerContent, spacing, helpHint)

	return m.styles.Header.Width(m.width).Render(fullHeader)
}

// renderFilters renders the filter bar with available filter options.
func (m Model) renderFilters() string {
	filters := []struct {
		key    string
		label  string
		filter Filter
	}{
		{"A", "All", FilterAll},
		{"O", "Old", FilterOld},
		{"N", "No Stars", FilterNoStars},
		{"F", "Forks", FilterForks},
		{"P", "Private", FilterPrivate},
	}

	var parts []string
	parts = append(parts, "Filters: ")

	for i, f := range filters {
		var rendered string
		keyLabel := fmt.Sprintf("[%s]%s", f.key, f.label)

		if m.currentFilter == f.filter {
			rendered = m.styles.ActiveFilter.Render(keyLabel)
		} else {
			rendered = m.styles.HelpDesc.Render(keyLabel)
		}

		parts = append(parts, rendered)
		if i < len(filters)-1 {
			parts = append(parts, " ")
		}
	}

	// Add language filter if set
	if m.languageFilter != "" {
		langLabel := fmt.Sprintf(" [L]anguage:%s", m.languageFilter)
		parts = append(parts, m.styles.ActiveFilter.Render(langLabel))
	} else {
		parts = append(parts, m.styles.HelpDesc.Render(" [L]anguage"))
	}

	return m.styles.FilterBar.Render(strings.Join(parts, ""))
}

// renderSortBar renders the sort bar with available sort options.
func (m Model) renderSortBar() string {
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

	return m.styles.FilterBar.Render(strings.Join(parts, ""))
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
	// Placeholder - will call table rendering in future task
	if len(m.filteredRepos) == 0 {
		return m.styles.HeaderInfo.Render("No repositories to display")
	}

	// Calculate visible rows based on height
	// Reserve space for: header, filters, sort bar, table header, footer, margins
	reservedRows := 8
	visibleRows := m.height - reservedRows
	if visibleRows < 1 {
		visibleRows = 5
	}

	var rows []string
	for i, repo := range m.filteredRepos {
		if i >= visibleRows {
			break
		}

		// Determine row style
		style := m.styles.TableRow
		if i == m.cursor {
			style = m.styles.SelectedRow
		}

		repoKey := fmt.Sprintf("%s/%s", repo.Owner, repo.Name)
		if m.marked[repoKey] {
			if i == m.cursor {
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

// renderFooter renders the footer with keybinding hints.
func (m Model) renderFooter() string {
	bindings := []struct {
		key  string
		desc string
	}{
		{"j/k", "navigate"},
		{"space", "mark"},
		{"enter", "details"},
		{"/", "search"},
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

	footer := strings.Join(parts, "")
	return m.styles.FilterBar.Render(footer)
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
