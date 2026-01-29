// SPDX-FileCopyrightText: 2026 Logan Lindquist Land
// SPDX-License-Identifier: FSL-1.1-MIT

package tui

import (
	"fmt"
	"log/slog"
	"os/exec"
	"runtime"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/llbbl/repjan/internal/analyze"
	"github.com/llbbl/repjan/internal/github"
)

// maxReposToShow is the maximum number of repo names to display in the confirm modal.
const maxReposToShow = 5

// archiveState holds the state for an ongoing archive operation.
type archiveState struct {
	repos     []github.Repository
	succeeded int
	failed    int
	errors    []error
}

// renderConfirmModal renders the archive confirmation modal.
func (m Model) renderConfirmModal() string {
	// Get marked repos
	var markedRepos []github.Repository
	for _, repo := range m.repos {
		if m.marked[repo.FullName()] {
			markedRepos = append(markedRepos, repo)
		}
	}

	count := len(markedRepos)
	if count == 0 {
		return m.styles.ModalBorder.Render("No repositories marked for archive")
	}

	// Build modal content
	var lines []string

	// Header
	lines = append(lines, m.styles.ModalTitle.Render("Archive Confirmation"))
	lines = append(lines, strings.Repeat("-", 40))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("You are about to archive %d repo%s:", count, pluralize(count)))
	lines = append(lines, "")

	// List first N repos
	showCount := count
	if showCount > maxReposToShow {
		showCount = maxReposToShow
	}

	for i := 0; i < showCount; i++ {
		lines = append(lines, fmt.Sprintf("  * %s", markedRepos[i].FullName()))
	}

	// Show remaining count if there are more
	if count > maxReposToShow {
		remaining := count - maxReposToShow
		lines = append(lines, fmt.Sprintf("  ... (%d more)", remaining))
	}

	lines = append(lines, "")
	lines = append(lines, "This action is reversible via GitHub")
	lines = append(lines, "web UI or gh CLI.")
	lines = append(lines, "")
	lines = append(lines, m.styles.HelpKey.Render("Continue? [Y/n]"))

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)

	return m.styles.ModalBorder.Render(content)
}

// pluralize returns "s" if count != 1, empty string otherwise.
func pluralize(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}

// startArchiveOperation returns a command to begin the archive operation.
func (m Model) startArchiveOperation() tea.Cmd {
	return func() tea.Msg {
		// Get marked repos
		var toArchive []github.Repository
		for _, repo := range m.repos {
			if m.marked[repo.FullName()] {
				toArchive = append(toArchive, repo)
			}
		}

		if len(toArchive) == 0 {
			return ArchiveCompleteMsg{
				Succeeded: 0,
				Failed:    0,
				Errors:    nil,
			}
		}

		// Return first progress message
		return ArchiveProgressMsg{
			Current: 0,
			Total:   len(toArchive),
		}
	}
}

// archiveNextRepo returns a command to archive the next repository in the queue.
func archiveNextRepo(client *github.Client, repos []github.Repository, current int, state *archiveState) tea.Cmd {
	slog.Debug("archiveNextRepo called",
		"component", "tui",
		"current", current,
		"totalRepos", len(repos),
	)

	return func() tea.Msg {
		if current >= len(repos) {
			slog.Debug("archive queue exhausted, returning ArchiveCompleteMsg",
				"component", "tui",
				"succeeded", state.succeeded,
				"failed", state.failed,
			)
			return ArchiveCompleteMsg{
				Succeeded: state.succeeded,
				Failed:    state.failed,
				Errors:    state.errors,
			}
		}

		repo := repos[current]
		slog.Debug("archiving repository",
			"component", "tui",
			"repo", repo.FullName(),
			"index", current,
		)

		err := client.ArchiveRepository(repo.Owner, repo.Name)

		if err != nil {
			slog.Debug("archive failed",
				"component", "tui",
				"repo", repo.FullName(),
				"err", err,
			)
			state.failed++
			state.errors = append(state.errors, fmt.Errorf("%s: %w", repo.FullName(), err))
		} else {
			slog.Debug("archive succeeded",
				"component", "tui",
				"repo", repo.FullName(),
			)
			state.succeeded++
		}

		return ArchiveProgressMsg{
			Current:  current + 1,
			Total:    len(repos),
			RepoName: repo.FullName(),
			Err:      err,
		}
	}
}

// getMarkedRepos returns a slice of all marked repositories.
func (m Model) getMarkedRepos() []github.Repository {
	var marked []github.Repository
	for _, repo := range m.repos {
		if m.marked[repo.FullName()] {
			marked = append(marked, repo)
		}
	}
	return marked
}

// renderDetailModal renders the repository detail modal.
func (m Model) renderDetailModal() string {
	if m.selectedRepo == nil {
		return ""
	}

	repo := m.selectedRepo

	// Build the modal content
	var content strings.Builder

	// Description
	description := repo.Description
	if description == "" {
		description = "No description"
	}
	content.WriteString(fmt.Sprintf("Description: %s\n\n", description))

	// Stats section
	content.WriteString("Stats:\n")

	language := repo.PrimaryLanguage
	if language == "" {
		language = "None"
	}

	visibility := "Public"
	if repo.IsPrivate {
		visibility = "Private"
	}

	content.WriteString(fmt.Sprintf("  Stars:         %d\n", repo.StargazerCount))
	content.WriteString(fmt.Sprintf("  Forks:         %d\n", repo.ForkCount))
	content.WriteString(fmt.Sprintf("  Language:      %s\n", language))
	content.WriteString(fmt.Sprintf("  Visibility:    %s\n\n", visibility))

	// Activity section
	content.WriteString("Activity:\n")

	lastPush := "Never"
	lastPushRelative := ""
	if !repo.PushedAt.IsZero() {
		lastPush = repo.PushedAt.Format("2006-01-02")
		lastPushRelative = formatDaysAgo(repo.DaysSinceActivity)
	}

	createdAt := "Unknown"
	if !repo.CreatedAt.IsZero() {
		createdAt = repo.CreatedAt.Format("2006-01-02")
	}

	if lastPushRelative != "" {
		content.WriteString(fmt.Sprintf("  Last Push:     %s (%s)\n", lastPushRelative, lastPush))
	} else {
		content.WriteString(fmt.Sprintf("  Last Push:     %s\n", lastPush))
	}
	content.WriteString(fmt.Sprintf("  Created:       %s\n\n", createdAt))

	// Archive Analysis section
	content.WriteString("Archive Analysis:\n")

	isCandidate, reasons := analyze.IsArchiveCandidate(*repo)
	status := "Active"
	if repo.IsArchived {
		status = "Archived"
	} else if isCandidate {
		status = "Candidate"
	}

	reasonsDisplay := "None"
	if reasons != "" {
		reasonsDisplay = reasons
	}

	content.WriteString(fmt.Sprintf("  Status:        %s\n", status))
	content.WriteString(fmt.Sprintf("  Reasons:       %s\n\n", reasonsDisplay))

	// Actions section
	content.WriteString("Actions:\n")
	content.WriteString("  [Space] Mark/unmark for archiving\n")
	content.WriteString("  [o]     Open in browser\n")
	if m.fabricEnabled {
		content.WriteString("  [i]     Analyze with Fabric\n")
	} else {
		content.WriteString("  [i]     Analyze with Fabric (disabled)\n")
	}
	content.WriteString("  [Esc]   Close\n")

	// Build the modal with title and content
	title := m.styles.ModalTitle.Render(fmt.Sprintf("Repository Details: %s/%s", repo.Owner, repo.Name))
	body := m.styles.ModalContent.Render(content.String())

	modalContent := lipgloss.JoinVertical(lipgloss.Left, title, body)

	// Apply border and padding
	modal := m.styles.ModalBorder.Render(modalContent)

	// Calculate modal dimensions for centering
	modalWidth := lipgloss.Width(modal)
	modalHeight := lipgloss.Height(modal)

	// Center the modal
	horizontalPadding := (m.width - modalWidth) / 2
	verticalPadding := (m.height - modalHeight) / 2

	if horizontalPadding < 0 {
		horizontalPadding = 0
	}
	if verticalPadding < 0 {
		verticalPadding = 0
	}

	// Create centered modal using lipgloss positioning
	centeredModal := lipgloss.NewStyle().
		MarginLeft(horizontalPadding).
		MarginTop(verticalPadding).
		Render(modal)

	return centeredModal
}

// formatDaysAgo formats days since activity into a human-readable string.
func formatDaysAgo(days int) string {
	if days == 0 {
		return "today"
	} else if days == 1 {
		return "yesterday"
	} else if days < 7 {
		return fmt.Sprintf("%d days ago", days)
	} else if days < 30 {
		weeks := days / 7
		if weeks == 1 {
			return "1 week ago"
		}
		return fmt.Sprintf("%d weeks ago", weeks)
	} else if days < 365 {
		months := days / 30
		if months == 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", months)
	}

	years := days / 365
	if years == 1 {
		return "1 year ago"
	}
	return fmt.Sprintf("%d years ago", years)
}

// openInBrowser opens the given URL in the default browser.
func openInBrowser(url string) tea.Cmd {
	return func() tea.Msg {
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command("open", url)
		default:
			cmd = exec.Command("xdg-open", url)
		}
		_ = cmd.Run()
		return nil
	}
}

// renderHelpModal renders the help modal with keybinding information.
func (m Model) renderHelpModal() string {
	var lines []string

	// Title
	lines = append(lines, m.styles.ModalTitle.Render("Help - Keyboard Shortcuts"))
	lines = append(lines, strings.Repeat("-", 55))

	// Category header style (dimmed)
	categoryStyle := lipgloss.NewStyle().Foreground(ColorMuted)

	// Helper to format a keybinding line
	formatBinding := func(key, desc string) string {
		return fmt.Sprintf("  %s  %s",
			m.styles.HelpKey.Render(fmt.Sprintf("%-13s", key)),
			m.styles.HelpDesc.Render(desc))
	}

	// Navigation section
	lines = append(lines, categoryStyle.Render("Navigation:"))
	lines = append(lines, formatBinding("j/k or Up/Dn", "Navigate list"))
	lines = append(lines, formatBinding("g/G", "Go to top/bottom"))
	lines = append(lines, formatBinding("/", "Search by name"))
	lines = append(lines, "")

	// Filtering section
	lines = append(lines, categoryStyle.Render("Filtering:"))
	lines = append(lines, formatBinding("a", "Show all"))
	lines = append(lines, formatBinding("o", "Show old (365+ days)"))
	lines = append(lines, formatBinding("n", "Show no stars"))
	lines = append(lines, formatBinding("f", "Show only forks"))
	lines = append(lines, formatBinding("l", "Language filter"))
	lines = append(lines, formatBinding("p", "Toggle private/public"))
	lines = append(lines, "")

	// Sorting section
	lines = append(lines, categoryStyle.Render("Sorting:"))
	lines = append(lines, formatBinding("1-4", "Sort by Name/Activity/Stars/Language"))
	lines = append(lines, "")

	// Actions section
	lines = append(lines, categoryStyle.Render("Actions:"))
	lines = append(lines, formatBinding("Space", "Mark/unmark for archiving"))
	lines = append(lines, formatBinding("Shift+A/U", "Mark/unmark all visible"))
	lines = append(lines, formatBinding("Enter", "View details"))
	lines = append(lines, formatBinding("a", "Archive marked repos"))
	lines = append(lines, formatBinding("e", "Export marked to JSON"))
	lines = append(lines, "")

	// Fabric section (conditional)
	if m.fabricEnabled {
		lines = append(lines, categoryStyle.Render("Fabric (enabled):"))
	} else {
		lines = append(lines, categoryStyle.Render("Fabric (if enabled):"))
	}
	lines = append(lines, formatBinding("i", "Analyze repo"))
	lines = append(lines, formatBinding("Shift+I", "Batch analyze"))
	lines = append(lines, "")

	// Close hint
	lines = append(lines, m.styles.HelpKey.Render("[Esc] Close"))

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	modal := m.styles.ModalBorder.Render(content)

	// Calculate modal dimensions for centering
	modalWidth := lipgloss.Width(modal)
	modalHeight := lipgloss.Height(modal)

	// Center the modal
	horizontalPadding := (m.width - modalWidth) / 2
	verticalPadding := (m.height - modalHeight) / 2

	if horizontalPadding < 0 {
		horizontalPadding = 0
	}
	if verticalPadding < 0 {
		verticalPadding = 0
	}

	// Create centered modal using lipgloss positioning
	centeredModal := lipgloss.NewStyle().
		MarginLeft(horizontalPadding).
		MarginTop(verticalPadding).
		Render(modal)

	return centeredModal
}

// renderLanguageModal renders the language filter selection modal.
func (m Model) renderLanguageModal() string {
	var lines []string

	lines = append(lines, m.styles.ModalTitle.Render("Filter by Language"))
	lines = append(lines, strings.Repeat("-", 30))

	// Render language options
	maxVisible := 15
	startIdx := 0
	if m.languageCursor >= maxVisible {
		startIdx = m.languageCursor - maxVisible + 1
	}

	for i, opt := range m.languages {
		if i < startIdx {
			continue
		}
		if i >= startIdx+maxVisible {
			lines = append(lines, m.styles.HelpDesc.Render("  ..."))
			break
		}

		// Build the line with cursor indicator
		cursor := "  "
		if i == m.languageCursor {
			cursor = "> "
		}

		// Format: cursor + name + count (right-aligned)
		countStr := fmt.Sprintf("(%d)", opt.count)
		nameWidth := 20
		name := opt.name
		if len(name) > nameWidth {
			name = name[:nameWidth-3] + "..."
		}

		lineText := fmt.Sprintf("%s%-*s %6s", cursor, nameWidth, name, countStr)

		// Highlight current selection
		if i == m.languageCursor {
			lines = append(lines, m.styles.ActiveFilter.Render(lineText))
		} else if m.languageFilter != "" && opt.name == m.languageFilter {
			// Highlight currently applied filter
			lines = append(lines, m.styles.HelpKey.Render(lineText))
		} else {
			lines = append(lines, m.styles.ModalContent.Render(lineText))
		}
	}

	lines = append(lines, strings.Repeat("-", 30))
	lines = append(lines, m.styles.HelpDesc.Render("j/k: Navigate  Enter: Select  Esc: Cancel"))

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return m.styles.ModalBorder.Render(content)
}

// populateLanguages builds the language options list from the current repos.
func (m *Model) populateLanguages() {
	// Count repos per language
	counts := make(map[string]int)
	for _, repo := range m.repos {
		// Skip archived repos to match filter behavior
		if repo.IsArchived {
			continue
		}
		lang := repo.PrimaryLanguage
		if lang == "" {
			lang = "None"
		}
		counts[lang]++
	}

	// Calculate total non-archived repos
	totalNonArchived := 0
	for _, count := range counts {
		totalNonArchived += count
	}

	// Build sorted list starting with "All Languages"
	m.languages = []languageOption{{name: "All Languages", count: totalNonArchived}}

	// Get sorted language names (excluding empty since we've mapped it to "None")
	var names []string
	for name := range counts {
		names = append(names, name)
	}

	// Sort by count descending, then by name ascending
	sortLanguagesByCount(names, counts)

	// Add languages to the list
	for _, name := range names {
		m.languages = append(m.languages, languageOption{
			name:  name,
			count: counts[name],
		})
	}
}

// sortLanguagesByCount sorts language names by their count (descending), then alphabetically.
func sortLanguagesByCount(names []string, counts map[string]int) {
	// Simple bubble sort for clarity - typically small number of languages
	for i := 0; i < len(names)-1; i++ {
		for j := 0; j < len(names)-i-1; j++ {
			// Sort by count descending
			if counts[names[j]] < counts[names[j+1]] {
				names[j], names[j+1] = names[j+1], names[j]
			} else if counts[names[j]] == counts[names[j+1]] {
				// If counts are equal, sort alphabetically ascending
				if names[j] > names[j+1] {
					names[j], names[j+1] = names[j+1], names[j]
				}
			}
		}
	}
}
