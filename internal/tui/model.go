// SPDX-FileCopyrightText: 2026 Logan Lindquist Land
// SPDX-License-Identifier: FSL-1.1-MIT

// Package tui provides the Bubble Tea TUI components for repjan.
package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/llbbl/repjan/internal/github"
)

// Filter represents the available repository filter options.
type Filter int

const (
	FilterAll Filter = iota
	FilterOld
	FilterNoStars
	FilterForks
	FilterPrivate
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

	// UI State
	cursor int
	marked map[string]bool // key: owner/name

	// Filters
	currentFilter  Filter
	languageFilter string

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

// NewModel creates a new TUI model with the provided repositories and configuration.
func NewModel(repos []github.Repository, owner string, fabricEnabled bool, fabricPath string) Model {
	// Copy repos to filteredRepos
	filteredRepos := make([]github.Repository, len(repos))
	copy(filteredRepos, repos)

	return Model{
		repos:         repos,
		filteredRepos: filteredRepos,
		owner:         owner,
		marked:        make(map[string]bool),
		currentFilter: FilterAll,
		sortField:     SortName,
		sortAscending: true,
		activeModal:   ModalNone,
		fabricEnabled: fabricEnabled,
		fabricPath:    fabricPath,
		styles:        DefaultStyles(),
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model - see update.go for implementation.

// View implements tea.Model - see view.go for implementation.
