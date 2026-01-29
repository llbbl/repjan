// SPDX-FileCopyrightText: 2026 Logan Lindquist Land
// SPDX-License-Identifier: FSL-1.1-MIT

package tui

import (
	"fmt"
	"log/slog"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// reservedRows is the number of rows reserved for UI chrome (not available for table content).
// This includes: header line, filter line, sort bar, table header, footer keybindings,
// status bar, and vertical margins/padding.
const reservedRows = 8

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
		slog.Debug("received ArchiveProgressMsg",
			"component", "tui",
			"current", msg.Current,
			"total", msg.Total,
			"repoName", msg.RepoName,
			"err", msg.Err,
			"archiveMode", m.archiveMode,
		)
		m.archiveProgress = msg.Current
		m.archiveTotal = msg.Total
		if msg.Err != nil {
			m.lastError = msg.Err
		} else if msg.RepoName != "" {
			// Operation succeeded - update the repo's IsArchived field based on mode
			if m.archiveMode == "unarchive" {
				m.markRepoAsUnarchived(msg.RepoName)
			} else {
				m.markRepoAsArchived(msg.RepoName)
			}
		}
		// Continue to the next repo if there are more
		if m.archiveState != nil && msg.Current < msg.Total {
			slog.Debug("chaining to next repo",
				"component", "tui",
				"nextIndex", msg.Current,
				"archiveMode", m.archiveMode,
			)
			if m.archiveMode == "unarchive" {
				return m, unarchiveNextRepo(m.client, m.archiveState.repos, msg.Current, m.archiveState)
			}
			return m, archiveNextRepo(m.client, m.archiveState.repos, msg.Current, m.archiveState)
		}
		// All repos processed - send completion message.
		//
		// Archive/unarchive completion logic:
		// The archiveNextRepo/unarchiveNextRepo command processes one repo at a time and sends
		// an ArchiveProgressMsg when done. The msg.Current field indicates how many repos have
		// been processed so far. When msg.Current >= msg.Total, all repos have been processed,
		// so we emit an ArchiveCompleteMsg to transition out of archiving state. This pattern
		// ensures the completion message is sent exactly once, after the final repo is processed,
		// preventing the operation from hanging indefinitely.
		if m.archiveState != nil && msg.Current >= msg.Total {
			slog.Debug("archive/unarchive operation complete, sending ArchiveCompleteMsg",
				"component", "tui",
				"succeeded", m.archiveState.succeeded,
				"failed", m.archiveState.failed,
				"archiveMode", m.archiveMode,
			)
			return m, func() tea.Msg {
				return ArchiveCompleteMsg{
					Succeeded: m.archiveState.succeeded,
					Failed:    m.archiveState.failed,
					Errors:    m.archiveState.errors,
				}
			}
		}
	case ArchiveCompleteMsg:
		m.archiving = false
		m.archiveProgress = 0
		m.archiveTotal = 0
		archiveMode := m.archiveMode // Save before clearing state
		m.archiveState = nil
		// Clear marks for successfully archived/unarchived repos and update status message
		if msg.Succeeded > 0 {
			m.clearArchivedMarks()
		}
		if archiveMode == "unarchive" {
			if msg.Failed > 0 {
				m.statusMessage = fmt.Sprintf("Unarchive completed: %d succeeded, %d failed", msg.Succeeded, msg.Failed)
			} else {
				m.statusMessage = fmt.Sprintf("Successfully unarchived %d repo%s", msg.Succeeded, pluralize(msg.Succeeded))
			}
		} else {
			if msg.Failed > 0 {
				m.statusMessage = fmt.Sprintf("Archive completed: %d succeeded, %d failed", msg.Succeeded, msg.Failed)
			} else {
				m.statusMessage = fmt.Sprintf("Successfully archived %d repo%s", msg.Succeeded, pluralize(msg.Succeeded))
			}
		}
		m.RefreshFilteredRepos()
	case FabricResultMsg:
		if msg.Err != nil {
			m.lastError = msg.Err
		}
	case ErrorMsg:
		m.lastError = msg.Err
	case syncStartedMsg:
		// Background sync has started
		m.syncing = true
		m.statusMessage = "Syncing repositories..."
		// Continue listening for more sync messages
		if m.syncCh != nil {
			return m, m.listenForSyncMsgs()
		}
	case ReposSyncedMsg:
		// Handle sync updates from background syncer
		m.syncing = false
		if msg.Error != nil {
			m.statusMessage = fmt.Sprintf("Sync failed: %v", msg.Error)
		} else if len(msg.Repos) > 0 {
			// Preserve cursor position relative to current repo if possible
			var currentRepoName string
			if m.cursor < len(m.filteredRepos) {
				currentRepoName = m.filteredRepos[m.cursor].FullName()
			}

			m.repos = msg.Repos
			m.lastSyncTime = time.Now()
			m.usingCache = false
			m.statusMessage = fmt.Sprintf("Synced %d repos", len(msg.Repos))
			m.RefreshFilteredRepos()

			// Try to restore cursor to same repo
			if currentRepoName != "" {
				for i, repo := range m.filteredRepos {
					if repo.FullName() == currentRepoName {
						m.cursor = i
						break
					}
				}
			}
			// Ensure cursor is within bounds
			if m.cursor >= len(m.filteredRepos) {
				m.cursor = max(0, len(m.filteredRepos)-1)
			}
		}
		// Continue listening for more sync messages
		if m.syncCh != nil {
			return m, m.listenForSyncMsgs()
		}
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

// handleConfirmModalKeys handles key input for the archive/unarchive confirmation modal.
func (m Model) handleConfirmModalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "Y", "y", "enter":
		// Confirm archive/unarchive operation
		m.activeModal = ModalNone
		if m.archiveMode == "unarchive" {
			return m, m.unarchiveMarkedRepos()
		}
		return m, m.archiveMarkedRepos()

	case "N", "n", "esc", "q":
		// Cancel archive/unarchive operation
		m.activeModal = ModalNone
		if m.archiveMode == "unarchive" {
			m.statusMessage = "Unarchive cancelled"
		} else {
			m.statusMessage = "Archive cancelled"
		}
		return m, nil
	}

	return m, nil
}

// handleMainViewKeys handles key input in the main repository list view.
func (m Model) handleMainViewKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	visibleRows := m.getVisibleRows()

	switch msg.String() {
	// Navigation keys
	case "j", "down":
		if len(m.filteredRepos) == 0 {
			return m, nil
		}
		m.cursor = min(m.cursor+1, len(m.filteredRepos)-1)
		// Scroll down if cursor goes below visible area
		if m.cursor >= m.viewportOffset+visibleRows {
			m.viewportOffset = m.cursor - visibleRows + 1
		}
		return m, nil

	case "k", "up":
		m.cursor = max(m.cursor-1, 0)
		// Scroll up if cursor goes above visible area
		if m.cursor < m.viewportOffset {
			m.viewportOffset = m.cursor
		}
		return m, nil

	case "pgdown", "ctrl+d":
		// Page down - move cursor and viewport by visible rows
		if len(m.filteredRepos) == 0 {
			return m, nil
		}
		m.cursor = min(m.cursor+visibleRows, len(m.filteredRepos)-1)
		m.viewportOffset = min(m.viewportOffset+visibleRows, max(0, len(m.filteredRepos)-visibleRows))
		return m, nil

	case "pgup", "ctrl+u":
		// Page up - move cursor and viewport by visible rows
		m.cursor = max(m.cursor-visibleRows, 0)
		m.viewportOffset = max(m.viewportOffset-visibleRows, 0)
		return m, nil

	case "g":
		// Go to top
		m.cursor = 0
		m.viewportOffset = 0
		return m, nil

	case "G":
		// Go to bottom
		if len(m.filteredRepos) > 0 {
			m.cursor = len(m.filteredRepos) - 1
			m.viewportOffset = max(0, len(m.filteredRepos)-visibleRows)
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
		// - When repos are marked: open confirm modal for archive/unarchive action
		// - When no repos are marked: set filter to FilterAll
		if len(m.marked) > 0 {
			// Determine if we're archiving or unarchiving based on marked repos
			markedRepos := m.getMarkedRepos()
			allArchived := true
			allUnarchived := true
			for _, repo := range markedRepos {
				if repo.IsArchived {
					allUnarchived = false
				} else {
					allArchived = false
				}
			}

			if allArchived {
				m.archiveMode = "unarchive"
			} else if allUnarchived {
				m.archiveMode = "archive"
			} else {
				// Mixed state - show error
				m.statusMessage = "Cannot mix archived and unarchived repos"
				return m, nil
			}

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
		// Toggle visibility of private repos (privacy-safe default: hidden)
		m.ToggleShowPrivate()
		return m, nil

	case "x":
		// Toggle visibility of archived repos
		m.ToggleShowArchived()
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
				// Persist removal to database
				if m.store != nil {
					_ = m.store.RemoveMarkedRepo(m.owner, repo.Name)
				}
			} else {
				m.marked[key] = true
				// Persist addition to database
				if m.store != nil {
					_ = m.store.AddMarkedRepo(m.owner, repo.Name)
				}
			}
		}
		return m, nil

	case "A":
		// Mark all visible/filtered repos
		for _, repo := range m.filteredRepos {
			m.marked[repo.FullName()] = true
		}
		// Persist all marks to database
		if m.store != nil {
			_ = m.SaveMarkedRepos()
		}
		return m, nil

	case "U":
		// Unmark all repos
		m.marked = make(map[string]bool)
		// Clear all marks from database
		if m.store != nil {
			_ = m.store.ClearMarkedRepos(m.owner)
		}
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
	slog.Debug("archiveMarkedRepos called",
		"component", "tui",
		"markedCount", len(m.marked),
	)

	if len(m.marked) == 0 {
		slog.Debug("no repos marked, returning nil",
			"component", "tui",
		)
		return nil
	}

	// Collect marked repos
	toArchive := m.getMarkedRepos()
	if len(toArchive) == 0 {
		slog.Debug("getMarkedRepos returned empty, returning nil",
			"component", "tui",
		)
		return nil
	}

	// Log the repos being archived
	repoNames := make([]string, len(toArchive))
	for i, repo := range toArchive {
		repoNames[i] = repo.FullName()
	}
	slog.Debug("starting archive operation",
		"component", "tui",
		"repoCount", len(toArchive),
		"repos", repoNames,
	)

	m.archiving = true
	m.archiveTotal = len(toArchive)
	m.archiveProgress = 0
	m.archiveState = &archiveState{
		repos:     toArchive,
		succeeded: 0,
		failed:    0,
		errors:    nil,
	}

	// Start archiving the first repo
	return archiveNextRepo(m.client, toArchive, 0, m.archiveState)
}

// unarchiveMarkedRepos returns a command to unarchive all marked repositories.
func (m *Model) unarchiveMarkedRepos() tea.Cmd {
	slog.Debug("unarchiveMarkedRepos called",
		"component", "tui",
		"markedCount", len(m.marked),
	)

	if len(m.marked) == 0 {
		slog.Debug("no repos marked, returning nil",
			"component", "tui",
		)
		return nil
	}

	// Collect marked repos
	toUnarchive := m.getMarkedRepos()
	if len(toUnarchive) == 0 {
		slog.Debug("getMarkedRepos returned empty, returning nil",
			"component", "tui",
		)
		return nil
	}

	// Log the repos being unarchived
	repoNames := make([]string, len(toUnarchive))
	for i, repo := range toUnarchive {
		repoNames[i] = repo.FullName()
	}
	slog.Debug("starting unarchive operation",
		"component", "tui",
		"repoCount", len(toUnarchive),
		"repos", repoNames,
	)

	m.archiving = true
	m.archiveTotal = len(toUnarchive)
	m.archiveProgress = 0
	m.archiveState = &archiveState{
		repos:     toUnarchive,
		succeeded: 0,
		failed:    0,
		errors:    nil,
	}

	// Start unarchiving the first repo
	return unarchiveNextRepo(m.client, toUnarchive, 0, m.archiveState)
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

// markRepoAsArchived updates a repo's IsArchived field in the model.
func (m *Model) markRepoAsArchived(fullName string) {
	for i := range m.repos {
		if m.repos[i].FullName() == fullName {
			m.repos[i].IsArchived = true
			break
		}
	}
}

// markRepoAsUnarchived updates a repo's IsArchived field to false in the model.
func (m *Model) markRepoAsUnarchived(fullName string) {
	for i := range m.repos {
		if m.repos[i].FullName() == fullName {
			m.repos[i].IsArchived = false
			break
		}
	}
}

// clearArchivedMarks removes marks from repos that have been archived or unarchived.
// It also updates the database: removes marks and updates is_archived flag.
// For archive mode: clears marks from repos where IsArchived == true
// For unarchive mode: clears marks from repos where IsArchived == false
func (m *Model) clearArchivedMarks() {
	for key := range m.marked {
		for i, repo := range m.repos {
			if repo.FullName() != key {
				continue
			}
			// For archive mode, check if repo is now archived
			// For unarchive mode, check if repo is now unarchived
			shouldClear := false
			if m.archiveMode == "unarchive" {
				shouldClear = !repo.IsArchived
			} else {
				shouldClear = repo.IsArchived
			}

			if shouldClear {
				delete(m.marked, key)
				// Remove from database marked_repos
				if m.store != nil {
					_ = m.store.RemoveMarkedRepo(m.owner, repo.Name)
					// Update is_archived in repositories table
					_ = m.store.UpdateRepository(m.repos[i])
				}
			}
			break
		}
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

// getVisibleRows returns the number of visible rows in the table viewport.
func (m Model) getVisibleRows() int {
	visibleRows := m.height - reservedRows
	if visibleRows < 1 {
		visibleRows = 5
	}
	return visibleRows
}
