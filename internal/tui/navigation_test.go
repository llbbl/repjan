// SPDX-FileCopyrightText: 2026 Logan Lindquist Land
// SPDX-License-Identifier: FSL-1.1-MIT

package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/llbbl/repjan/internal/github"
)

// createTestModelWithRepos creates a test model with the specified number of repos.
func createTestModelWithRepos(count int, height int) Model {
	repos := make([]github.Repository, count)
	for i := 0; i < count; i++ {
		repos[i] = github.Repository{
			Name:              string(rune('a' + i%26)),
			Owner:             "testowner",
			DaysSinceActivity: i * 10,
		}
	}

	m := NewModel(repos, "testowner", nil, false, "", nil)
	m.height = height
	m.width = 100
	m.filteredRepos = repos
	return m
}

func TestNavigationDown_ScrollsWhenCursorPastVisible(t *testing.T) {
	// Create model with 50 repos and height that shows ~10 rows
	m := createTestModelWithRepos(50, 18) // 18 - 8 reserved = 10 visible rows

	// Cursor starts at 0, viewport at 0
	if m.cursor != 0 {
		t.Errorf("expected cursor at 0, got %d", m.cursor)
	}
	if m.viewportOffset != 0 {
		t.Errorf("expected viewportOffset at 0, got %d", m.viewportOffset)
	}

	// Navigate down 15 times (past the visible area of 10)
	for i := 0; i < 15; i++ {
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
		m = newModel.(Model)
	}

	// Cursor should be at 15
	if m.cursor != 15 {
		t.Errorf("expected cursor at 15, got %d", m.cursor)
	}

	// Viewport should have scrolled (cursor should be visible)
	visibleRows := m.getVisibleRows()
	if m.cursor < m.viewportOffset || m.cursor >= m.viewportOffset+visibleRows {
		t.Errorf("cursor %d not visible in viewport (offset=%d, visibleRows=%d)",
			m.cursor, m.viewportOffset, visibleRows)
	}
}

func TestNavigationUp_ScrollsWhenCursorAboveVisible(t *testing.T) {
	// Create model with 50 repos
	m := createTestModelWithRepos(50, 18)

	// Start at position 20 with viewport offset at 15
	m.cursor = 20
	m.viewportOffset = 15

	// Navigate up 10 times (past the top of visible area)
	for i := 0; i < 10; i++ {
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
		m = newModel.(Model)
	}

	// Cursor should be at 10
	if m.cursor != 10 {
		t.Errorf("expected cursor at 10, got %d", m.cursor)
	}

	// Viewport should have scrolled to show cursor
	if m.cursor < m.viewportOffset {
		t.Errorf("cursor %d above viewport offset %d", m.cursor, m.viewportOffset)
	}
}

func TestPageDown_MovesViewportByVisibleRows(t *testing.T) {
	m := createTestModelWithRepos(50, 18) // 10 visible rows
	visibleRows := m.getVisibleRows()

	// Initial state
	initialCursor := m.cursor
	initialOffset := m.viewportOffset

	// Press Page Down
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	m = newModel.(Model)

	// Cursor should move by visibleRows (clamped to max)
	expectedCursor := min(initialCursor+visibleRows, len(m.filteredRepos)-1)
	if m.cursor != expectedCursor {
		t.Errorf("expected cursor at %d after pgdown, got %d", expectedCursor, m.cursor)
	}

	// Viewport should move by visibleRows (clamped appropriately)
	if m.viewportOffset <= initialOffset && len(m.filteredRepos) > visibleRows {
		t.Errorf("expected viewportOffset to increase after pgdown, was %d now %d",
			initialOffset, m.viewportOffset)
	}
}

func TestPageUp_MovesViewportByVisibleRows(t *testing.T) {
	m := createTestModelWithRepos(50, 18)
	visibleRows := m.getVisibleRows()

	// Start at position 30 with viewport at 25
	m.cursor = 30
	m.viewportOffset = 25

	// Press Page Up
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	m = newModel.(Model)

	// Cursor should move up by visibleRows
	expectedCursor := max(30-visibleRows, 0)
	if m.cursor != expectedCursor {
		t.Errorf("expected cursor at %d after pgup, got %d", expectedCursor, m.cursor)
	}

	// Viewport should move up
	expectedOffset := max(25-visibleRows, 0)
	if m.viewportOffset != expectedOffset {
		t.Errorf("expected viewportOffset at %d after pgup, got %d", expectedOffset, m.viewportOffset)
	}
}

func TestGoToTop_ResetsCursorAndViewport(t *testing.T) {
	m := createTestModelWithRepos(50, 18)

	// Start somewhere in the middle
	m.cursor = 30
	m.viewportOffset = 25

	// Press 'g' to go to top
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	m = newModel.(Model)

	if m.cursor != 0 {
		t.Errorf("expected cursor at 0 after 'g', got %d", m.cursor)
	}
	if m.viewportOffset != 0 {
		t.Errorf("expected viewportOffset at 0 after 'g', got %d", m.viewportOffset)
	}
}

func TestGoToBottom_SetsCursorAndViewportToEnd(t *testing.T) {
	m := createTestModelWithRepos(50, 18)
	visibleRows := m.getVisibleRows()

	// Press 'G' to go to bottom
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	m = newModel.(Model)

	expectedCursor := len(m.filteredRepos) - 1
	if m.cursor != expectedCursor {
		t.Errorf("expected cursor at %d after 'G', got %d", expectedCursor, m.cursor)
	}

	expectedOffset := max(0, len(m.filteredRepos)-visibleRows)
	if m.viewportOffset != expectedOffset {
		t.Errorf("expected viewportOffset at %d after 'G', got %d", expectedOffset, m.viewportOffset)
	}
}

func TestFilterChange_ResetsViewportWhenNeeded(t *testing.T) {
	// Create model with repos, some with stars and some without
	repos := []github.Repository{
		{Name: "repo1", Owner: "test", StargazerCount: 10},
		{Name: "repo2", Owner: "test", StargazerCount: 0},
		{Name: "repo3", Owner: "test", StargazerCount: 5},
		{Name: "repo4", Owner: "test", StargazerCount: 0},
		{Name: "repo5", Owner: "test", StargazerCount: 0},
	}

	m := NewModel(repos, "test", nil, false, "", nil)
	m.height = 18
	m.width = 100

	// Set cursor and viewport to position 4
	m.cursor = 4
	m.viewportOffset = 2

	// Apply NoStars filter (should reduce to 3 repos)
	m.ApplyFilter(FilterNoStars)

	// Cursor should be clamped to valid range
	if m.cursor >= len(m.filteredRepos) {
		t.Errorf("cursor %d out of bounds for %d filtered repos", m.cursor, len(m.filteredRepos))
	}

	// Viewport offset should be adjusted if needed
	if m.viewportOffset > m.cursor {
		t.Errorf("viewportOffset %d greater than cursor %d", m.viewportOffset, m.cursor)
	}
}

func TestNavigationWithEmptyList(t *testing.T) {
	m := createTestModelWithRepos(0, 18)

	// Navigate down should not panic or change anything
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = newModel.(Model)

	if m.cursor != 0 {
		t.Errorf("cursor should stay at 0 with empty list, got %d", m.cursor)
	}

	// Page down should not panic
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	m = newModel.(Model)

	if m.cursor != 0 {
		t.Errorf("cursor should stay at 0 with empty list after pgdown, got %d", m.cursor)
	}
}

func TestCtrlD_PageDown(t *testing.T) {
	m := createTestModelWithRepos(50, 18)
	visibleRows := m.getVisibleRows()

	// Press Ctrl+D (alternative page down)
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
	m = newModel.(Model)

	expectedCursor := min(visibleRows, len(m.filteredRepos)-1)
	if m.cursor != expectedCursor {
		t.Errorf("expected cursor at %d after ctrl+d, got %d", expectedCursor, m.cursor)
	}
}

func TestCtrlU_PageUp(t *testing.T) {
	m := createTestModelWithRepos(50, 18)
	visibleRows := m.getVisibleRows()

	// Start at position 30
	m.cursor = 30
	m.viewportOffset = 25

	// Press Ctrl+U (alternative page up)
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlU})
	m = newModel.(Model)

	expectedCursor := max(30-visibleRows, 0)
	if m.cursor != expectedCursor {
		t.Errorf("expected cursor at %d after ctrl+u, got %d", expectedCursor, m.cursor)
	}
}

func TestCursorStaysVisibleAfterScroll(t *testing.T) {
	m := createTestModelWithRepos(100, 18)
	visibleRows := m.getVisibleRows()

	// Navigate down a lot
	for i := 0; i < 50; i++ {
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
		m = newModel.(Model)

		// After each move, cursor should be visible
		if m.cursor < m.viewportOffset {
			t.Errorf("after %d moves down: cursor %d above viewport %d", i+1, m.cursor, m.viewportOffset)
		}
		if m.cursor >= m.viewportOffset+visibleRows {
			t.Errorf("after %d moves down: cursor %d below viewport end %d",
				i+1, m.cursor, m.viewportOffset+visibleRows)
		}
	}

	// Navigate back up
	for i := 0; i < 50; i++ {
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
		m = newModel.(Model)

		// After each move, cursor should be visible
		if m.cursor < m.viewportOffset {
			t.Errorf("after %d moves up: cursor %d above viewport %d", i+1, m.cursor, m.viewportOffset)
		}
		if m.cursor >= m.viewportOffset+visibleRows {
			t.Errorf("after %d moves up: cursor %d below viewport end %d",
				i+1, m.cursor, m.viewportOffset+visibleRows)
		}
	}
}
