// SPDX-FileCopyrightText: 2026 Logan Lindquist Land
// SPDX-License-Identifier: FSL-1.1-MIT

package tui

import "github.com/charmbracelet/lipgloss"

// Status colors for repository states.
const (
	ColorActive    = lipgloss.Color("#00FF00") // Green - active repos
	ColorCandidate = lipgloss.Color("#FFFF00") // Yellow - archive candidates
	ColorArchived  = lipgloss.Color("#808080") // Gray - already archived
	ColorMarked    = lipgloss.Color("#0000FF") // Blue - marked for archiving
)

// UI colors for general interface elements.
const (
	ColorPrimary   = lipgloss.Color("#7D56F4") // Purple accent
	ColorSecondary = lipgloss.Color("#FFFDF5") // Off-white text
	ColorMuted     = lipgloss.Color("#626262") // Muted text
	ColorBorder    = lipgloss.Color("#383838") // Border color
)

// Warning colors for private repository visibility.
const (
	ColorWarningBg     = lipgloss.Color("#8B0000") // Dark red background
	ColorWarningText   = lipgloss.Color("#FFFFFF") // White text
	ColorWarningAccent = lipgloss.Color("#FF4500") // Orange-red for emphasis
	ColorWarningBorder = lipgloss.Color("#FF0000") // Bright red border
)

// Styles contains all lipgloss style definitions for the TUI.
type Styles struct {
	// Header styles
	Header      lipgloss.Style
	HeaderTitle lipgloss.Style
	HeaderInfo  lipgloss.Style

	// Filter/Sort bars
	FilterBar    lipgloss.Style
	ActiveFilter lipgloss.Style

	// Table styles
	TableHeader lipgloss.Style
	TableRow    lipgloss.Style
	SelectedRow lipgloss.Style
	MarkedRow   lipgloss.Style

	// Status indicators
	StatusActive    lipgloss.Style
	StatusCandidate lipgloss.Style
	StatusArchived  lipgloss.Style

	// Modal styles
	ModalBorder  lipgloss.Style
	ModalTitle   lipgloss.Style
	ModalContent lipgloss.Style

	// Help styles
	HelpKey  lipgloss.Style
	HelpDesc lipgloss.Style

	// Message styles
	Error   lipgloss.Style
	Warning lipgloss.Style
	Success lipgloss.Style

	// Private warning styles
	PrivateWarningBanner lipgloss.Style // Full warning banner
	PrivateWarningText   lipgloss.Style // Text within the warning
	PrivateIndicator     lipgloss.Style // Bold red indicator for filter bar
	PrivateHeaderTint    lipgloss.Style // Tinted header when private visible
}

// DefaultStyles creates a new Styles instance with default styling.
func DefaultStyles() Styles {
	return Styles{
		// Header
		Header: lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(ColorBorder).
			BorderBottom(true).
			Padding(0, 1),

		HeaderTitle: lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true),

		HeaderInfo: lipgloss.NewStyle().
			Foreground(ColorMuted),

		// Filter/Sort bars
		FilterBar: lipgloss.NewStyle().
			Padding(0, 1).
			MarginBottom(1),

		ActiveFilter: lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true),

		// Table
		TableHeader: lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Bold(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(ColorBorder).
			BorderBottom(true).
			Padding(0, 1),

		TableRow: lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Padding(0, 1),

		SelectedRow: lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Background(ColorPrimary).
			Bold(true).
			Padding(0, 1),

		MarkedRow: lipgloss.NewStyle().
			Foreground(ColorMarked).
			Bold(true).
			Padding(0, 1),

		// Status indicators
		StatusActive: lipgloss.NewStyle().
			Foreground(ColorActive).
			Bold(true),

		StatusCandidate: lipgloss.NewStyle().
			Foreground(ColorCandidate).
			Bold(true),

		StatusArchived: lipgloss.NewStyle().
			Foreground(ColorArchived),

		// Modal
		ModalBorder: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(1, 2),

		ModalTitle: lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true).
			MarginBottom(1),

		ModalContent: lipgloss.NewStyle().
			Foreground(ColorSecondary),

		// Help
		HelpKey: lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true),

		HelpDesc: lipgloss.NewStyle().
			Foreground(ColorMuted),

		// Messages
		Error: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000")).
			Bold(true),

		Warning: lipgloss.NewStyle().
			Foreground(ColorCandidate).
			Bold(true),

		Success: lipgloss.NewStyle().
			Foreground(ColorActive).
			Bold(true),

		// Private warning styles
		PrivateWarningBanner: lipgloss.NewStyle().
			Background(ColorWarningBg).
			Foreground(ColorWarningText).
			Bold(true).
			Padding(0, 1).
			BorderStyle(lipgloss.ThickBorder()).
			BorderForeground(ColorWarningBorder).
			BorderTop(true).
			BorderBottom(true),

		PrivateWarningText: lipgloss.NewStyle().
			Foreground(ColorWarningText).
			Background(ColorWarningBg).
			Bold(true),

		PrivateIndicator: lipgloss.NewStyle().
			Foreground(ColorWarningBorder).
			Background(ColorWarningBg).
			Bold(true),

		PrivateHeaderTint: lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(ColorWarningBorder).
			BorderBottom(true).
			Background(lipgloss.Color("#3D0000")).
			Padding(0, 1),
	}
}

// DefaultStyle is the default style instance for the TUI.
var DefaultStyle = DefaultStyles()
