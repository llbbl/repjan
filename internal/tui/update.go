// SPDX-FileCopyrightText: 2026 Logan Lindquist Land
// SPDX-License-Identifier: FSL-1.1-MIT

package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case ReposFetchedMsg:
		m.loading = false
		if msg.Err != nil {
			m.lastError = msg.Err
		} else {
			m.repos = msg.Repos
			m.RefreshFilteredRepos()
		}
	case ArchiveProgressMsg:
		m.archiveProgress = msg.Current
		m.archiveTotal = msg.Total
		if msg.Err != nil {
			m.lastError = msg.Err
		}
	case ArchiveCompleteMsg:
		m.archiving = false
		m.archiveProgress = 0
		m.archiveTotal = 0
		if msg.Failed > 0 {
			m.statusMessage = "Archive completed with errors"
		} else {
			m.statusMessage = "Archive completed successfully"
		}
	case FabricResultMsg:
		if msg.Err != nil {
			m.lastError = msg.Err
		}
	case ErrorMsg:
		m.lastError = msg.Err
	}
	return m, nil
}

// handleKeyMsg routes key messages to the appropriate handler.
func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle search mode first
	if m.searchMode {
		return m.handleSearchInput(msg)
	}

	// Handle modal keys
	if m.activeModal != ModalNone {
		return m.handleModalKeys(msg)
	}

	// Handle main view keys
	return m.handleMainViewKeys(msg)
}

// handleSearchInput handles key input when in search mode.
func (m Model) handleSearchInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEscape:
		// Clear search and exit search mode
		m.searchMode = false
		m.searchQuery = ""
		m.RefreshFilteredRepos()
		return m, nil

	case tea.KeyEnter:
		// Keep search filter applied and exit search mode
		m.searchMode = false
		return m, nil

	case tea.KeyBackspace:
		// Remove last character and refresh for real-time filtering
		if len(m.searchQuery) > 0 {
			m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
			m.RefreshFilteredRepos()
		}
		return m, nil

	case tea.KeyRunes:
		// Add typed characters to search query and refresh for real-time filtering
		m.searchQuery += string(msg.Runes)
		m.RefreshFilteredRepos()
		return m, nil

	case tea.KeySpace:
		// Add space to search query and refresh for real-time filtering
		m.searchQuery += " "
		m.RefreshFilteredRepos()
		return m, nil
	}

	return m, nil
}

// handleModalKeys handles key input when a modal is active.
func (m Model) handleModalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle confirm modal specific keys first
	if m.activeModal == ModalConfirm {
		return m.handleConfirmModalKeys(msg)
	}

	// Handle language modal specific keys
	if m.activeModal == ModalLanguage {
		return m.handleLanguageModalKeys(msg)
	}

	switch msg.String() {
	case "esc", "q":
		// Close any modal
		m.activeModal = ModalNone
		m.selectedRepo = nil
		return m, nil

	case "enter":
		// For other modals, just close
		m.activeModal = ModalNone
		m.selectedRepo = nil
		return m, nil
	}

	return m, nil
}

// handleLanguageModalKeys handles key input for the language filter modal.
func (m Model) handleLanguageModalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		// Close modal without changing filter
		m.activeModal = ModalNone
		return m, nil

	case "enter":
		// Select language and apply filter
		if m.languageCursor >= 0 && m.languageCursor < len(m.languages) {
			selected := m.languages[m.languageCursor]
			if selected.name == "All Languages" {
				// Clear language filter
				m.languageFilter = ""
			} else {
				// Apply selected language filter
				m.languageFilter = selected.name
			}
			m.RefreshFilteredRepos()
		}
		m.activeModal = ModalNone
		return m, nil

	case "j", "down":
		m.handleLanguageModalNav(1)
		return m, nil

	case "k", "up":
		m.handleLanguageModalNav(-1)
		return m, nil
	}

	return m, nil
}

// handleConfirmModalKeys handles key input for the archive confirmation modal.
func (m Model) handleConfirmModalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "Y", "y", "enter":
		// Confirm archive operation
		m.activeModal = ModalNone
		return m, m.archiveMarkedRepos()

	case "N", "n", "esc", "q":
		// Cancel archive operation
		m.activeModal = ModalNone
		m.statusMessage = "Archive cancelled"
		return m, nil
	}

	return m, nil
}

// handleMainViewKeys handles key input in the main repository list view.
func (m Model) handleMainViewKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	// Navigation keys
	case "j", "down":
		m.cursor = min(m.cursor+1, len(m.filteredRepos)-1)
		if m.cursor < 0 {
			m.cursor = 0
		}
		return m, nil

	case "k", "up":
		m.cursor = max(m.cursor-1, 0)
		return m, nil

	case "g":
		// Go to top
		m.cursor = 0
		return m, nil

	case "G":
		// Go to bottom
		if len(m.filteredRepos) > 0 {
			m.cursor = len(m.filteredRepos) - 1
		}
		return m, nil

	// Search mode
	case "/":
		m.searchMode = true
		m.searchQuery = ""
		return m, nil

	// Filter keys
	case "a":
		// 'a' has dual purpose:
		// - When repos are marked: open confirm modal for archive action
		// - When no repos are marked: set filter to FilterAll
		if len(m.marked) > 0 {
			m.activeModal = ModalConfirm
			return m, nil
		}
		m.ApplyFilter(FilterAll)
		return m, nil

	case "o":
		m.ApplyFilter(FilterOld)
		return m, nil

	case "n":
		m.ApplyFilter(FilterNoStars)
		return m, nil

	case "f":
		m.ApplyFilter(FilterForks)
		return m, nil

	case "p":
		m.ApplyFilter(FilterPrivate)
		return m, nil

	case "l":
		// Open language filter modal
		m.populateLanguages()
		m.languageCursor = 0
		m.activeModal = ModalLanguage
		return m, nil

	// Sorting keys
	case "1":
		if m.sortField == SortName {
			m.ToggleSortDirection()
		} else {
			m.sortField = SortName
			m.sortAscending = true
			m.RefreshFilteredRepos()
		}
		return m, nil

	case "2":
		if m.sortField == SortActivity {
			m.ToggleSortDirection()
		} else {
			m.sortField = SortActivity
			m.sortAscending = true
			m.RefreshFilteredRepos()
		}
		return m, nil

	case "3":
		if m.sortField == SortStars {
			m.ToggleSortDirection()
		} else {
			m.sortField = SortStars
			m.sortAscending = false // Default descending for stars
			m.RefreshFilteredRepos()
		}
		return m, nil

	case "4":
		if m.sortField == SortLanguage {
			m.ToggleSortDirection()
		} else {
			m.sortField = SortLanguage
			m.sortAscending = true
			m.RefreshFilteredRepos()
		}
		return m, nil

	// Action keys
	case " ":
		// Toggle mark on current repo
		if len(m.filteredRepos) > 0 && m.cursor < len(m.filteredRepos) {
			repo := m.filteredRepos[m.cursor]
			key := repo.FullName()
			if m.marked[key] {
				delete(m.marked, key)
			} else {
				m.marked[key] = true
			}
		}
		return m, nil

	case "A":
		// Mark all visible/filtered repos
		for _, repo := range m.filteredRepos {
			m.marked[repo.FullName()] = true
		}
		return m, nil

	case "U":
		// Unmark all repos
		m.marked = make(map[string]bool)
		return m, nil

	case "enter":
		// Open detail modal for current repo
		if len(m.filteredRepos) > 0 && m.cursor < len(m.filteredRepos) {
			repo := m.filteredRepos[m.cursor]
			m.selectedRepo = &repo
			m.activeModal = ModalDetail
		}
		return m, nil

	case "e":
		// Export marked repos
		return m, m.exportMarkedRepos()

	// Meta keys
	case "?":
		// Open help modal
		m.activeModal = ModalHelp
		return m, nil

	case "q":
		// Quit
		return m, tea.Quit

	case "esc":
		// Close modal / exit search mode (fallback)
		if m.searchMode {
			m.searchMode = false
			m.searchQuery = ""
			m.RefreshFilteredRepos()
		}
		return m, nil
	}

	return m, nil
}

// handleLanguageModalNav handles navigation within the language filter modal.
func (m *Model) handleLanguageModalNav(delta int) {
	newPos := m.languageCursor + delta
	if newPos < 0 {
		newPos = 0
	}
	if newPos >= len(m.languages) {
		newPos = len(m.languages) - 1
	}
	m.languageCursor = newPos
}

// applySearchFilter filters repos based on the current search query.
func (m *Model) applySearchFilter() {
	// The search is applied by refreshing filtered repos
	// The filter.go functions should be extended to include search
	// For now, we trigger a refresh which will apply current filters
	m.RefreshFilteredRepos()
}

// archiveMarkedRepos returns a command to archive all marked repositories.
func (m *Model) archiveMarkedRepos() tea.Cmd {
	if len(m.marked) == 0 {
		return nil
	}

	m.archiving = true
	m.archiveTotal = len(m.marked)
	m.archiveProgress = 0

	// Return a command that will handle the archive operation
	// The actual implementation will send ArchiveProgressMsg and ArchiveCompleteMsg
	return func() tea.Msg {
		// This is a placeholder - the actual archive logic will be implemented
		// in the github package and called here
		return ArchiveCompleteMsg{
			Succeeded: len(m.marked),
			Failed:    0,
			Errors:    nil,
		}
	}
}

// exportMarkedRepos returns a command to export marked repositories.
func (m *Model) exportMarkedRepos() tea.Cmd {
	if len(m.marked) == 0 {
		m.statusMessage = "No repositories marked for export"
		return nil
	}

	// Return a command that will handle the export operation
	return func() tea.Msg {
		// This is a placeholder - the actual export logic will be implemented
		// and will write to a file or stdout
		return nil
	}
}

// min returns the smaller of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// max returns the larger of two integers.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
