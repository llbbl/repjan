// SPDX-FileCopyrightText: 2026 Logan Lindquist Land
// SPDX-License-Identifier: FSL-1.1-MIT

// Package tui provides the Bubble Tea TUI components for repjan.
package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/llbbl/repjan/internal/github"
	"github.com/llbbl/repjan/internal/store"
	"github.com/llbbl/repjan/internal/sync"
)

// Filter represents the available repository filter options.
type Filter int

const (
	FilterAll Filter = iota
	FilterOld
	FilterNoStars
	FilterForks
)

// SortField represents the available sorting fields.
type SortField int

const (
	SortName SortField = iota
	SortActivity
	SortStars
	SortLanguage
)

// ModalType represents the type of modal currently displayed.
type ModalType int

const (
	ModalNone ModalType = iota
	ModalDetail
	ModalConfirm
	ModalHelp
	ModalLanguage
)

// languageOption represents a language filter option with its repo count.
type languageOption struct {
	name  string
	count int
}

// Model is the main TUI model for repjan.
type Model struct {
	// Data
	repos         []github.Repository
	filteredRepos []github.Repository
	owner         string
	client        *github.Client
	store         *store.Store // database store for persistence

	// UI State
	cursor         int
	viewportOffset int             // scroll offset for table pagination
	marked         map[string]bool // key: owner/name

	// Filters
	currentFilter  Filter
	languageFilter string

	// Visibility toggles (privacy-safe defaults)
	showPrivate  bool // whether to include private repos (default: false)
	showArchived bool // whether to include archived repos (default: false)

	// Sorting
	sortField     SortField
	sortAscending bool

	// Modals
	activeModal    ModalType
	selectedRepo   *github.Repository // for detail modal
	languageCursor int                // cursor position in language list
	languages      []languageOption   // cached language options

	// Search
	searchMode  bool
	searchQuery string

	// Async state
	loading         bool
	archiving       bool
	archiveProgress int
	archiveTotal    int
	archiveState    *archiveState       // tracks ongoing archive operation
	syncing         bool                // whether a sync operation is in progress
	lastSyncTime    time.Time           // when repos were last synced from GitHub
	usingCache      bool                // whether we're showing cached data
	syncCh          <-chan sync.SyncMsg // channel for receiving sync messages

	// Fabric
	fabricEnabled bool
	fabricPath    string

	// Dimensions
	width, height int

	// Messages
	lastError     error
	statusMessage string

	// Styles
	styles Styles
}

// ReposFetchedMsg is sent when repositories have been fetched.
type ReposFetchedMsg struct {
	Repos []github.Repository
	Err   error
}

// ArchiveProgressMsg is sent during archive operations to report progress.
type ArchiveProgressMsg struct {
	Current  int
	Total    int
	RepoName string
	Err      error
}

// ArchiveCompleteMsg is sent when an archive operation completes.
type ArchiveCompleteMsg struct {
	Succeeded int
	Failed    int
	Errors    []error
}

// FabricResultMsg is sent when a Fabric AI analysis completes.
type FabricResultMsg struct {
	RepoName string
	Result   string
	Err      error
}

// ErrorMsg represents an error that occurred during async operations.
type ErrorMsg struct {
	Err error
}

// ReposSyncedMsg is sent when background sync completes with new repository data.
// The TUI should update its repos while preserving UI state (cursor, marks, filters).
type ReposSyncedMsg struct {
	Repos []github.Repository
	Error error
}

// NewModel creates a new TUI model with the provided repositories and configuration.
// By default, private and archived repositories are hidden for privacy safety.
func NewModel(repos []github.Repository, owner string, client *github.Client, fabricEnabled bool, fabricPath string, syncCh <-chan sync.SyncMsg) Model {
	m := Model{
		repos:         repos,
		owner:         owner,
		client:        client,
		marked:        make(map[string]bool),
		currentFilter: FilterAll,
		sortField:     SortActivity,
		sortAscending: true, // oldest first
		activeModal:   ModalNone,
		fabricEnabled: fabricEnabled,
		fabricPath:    fabricPath,
		syncCh:        syncCh,
		lastSyncTime:  time.Now(),
		styles:        DefaultStyles(),
		// showPrivate and showArchived default to false (Go zero values)
		// This provides privacy-safe defaults by hiding sensitive repos
	}

	// Apply initial filtering to respect visibility defaults
	m.RefreshFilteredRepos()

	return m
}

// NewModelWithStore creates a new TUI model with a store for database persistence.
// This is the preferred constructor when database support is enabled.
func NewModelWithStore(repos []github.Repository, owner string, client *github.Client, s *store.Store, fabricEnabled bool, fabricPath string, syncCh <-chan sync.SyncMsg) Model {
	m := NewModel(repos, owner, client, fabricEnabled, fabricPath, syncCh)
	m.store = s
	return m
}

// NewModelWithOptions creates a new TUI model with additional options like last sync time.
func NewModelWithOptions(repos []github.Repository, owner string, client *github.Client, s *store.Store, fabricEnabled bool, fabricPath string, lastSyncTime time.Time, usingCache bool, syncCh <-chan sync.SyncMsg) Model {
	m := NewModelWithStore(repos, owner, client, s, fabricEnabled, fabricPath, syncCh)
	m.lastSyncTime = lastSyncTime
	m.usingCache = usingCache
	return m
}

// SetStore sets the store for the model (useful for testing or delayed initialization).
func (m *Model) SetStore(s *store.Store) {
	m.store = s
}

// LoadMarkedRepos loads marked repos from the database into the model's marked map.
func (m *Model) LoadMarkedRepos() error {
	if m.store == nil {
		return nil
	}

	names, err := m.store.GetMarkedRepos(m.owner)
	if err != nil {
		return err
	}

	// Convert repo names to full names (owner/name)
	for _, name := range names {
		fullName := m.owner + "/" + name
		m.marked[fullName] = true
	}

	return nil
}

// SaveMarkedRepos saves the current marked repos to the database.
func (m *Model) SaveMarkedRepos() error {
	if m.store == nil {
		return nil
	}

	// Extract repo names from marked map
	var names []string
	for fullName := range m.marked {
		// fullName is "owner/name", extract just the name
		for _, repo := range m.repos {
			if repo.FullName() == fullName {
				names = append(names, repo.Name)
				break
			}
		}
	}

	return m.store.SaveMarkedRepos(m.owner, names)
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	if m.syncCh == nil {
		return nil
	}
	return m.listenForSyncMsgs()
}

// listenForSyncMsgs returns a command that listens for messages from the sync channel.
func (m Model) listenForSyncMsgs() tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-m.syncCh
		if !ok {
			// Channel closed, syncer stopped
			return nil
		}

		// Convert sync.SyncMsg to ReposSyncedMsg
		switch msg.Type {
		case sync.SyncStarted:
			// Return a message indicating sync has started
			return syncStartedMsg{}
		case sync.SyncCompleted:
			return ReposSyncedMsg{
				Repos: msg.Repos,
			}
		case sync.SyncError:
			return ReposSyncedMsg{
				Error: msg.Error,
			}
		}
		return nil
	}
}

// syncStartedMsg indicates that a background sync has started.
type syncStartedMsg struct{}

// Update implements tea.Model - see update.go for implementation.

// View implements tea.Model - see view.go for implementation.
