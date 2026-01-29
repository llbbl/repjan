// SPDX-FileCopyrightText: 2026 Logan Lindquist Land
// SPDX-License-Identifier: FSL-1.1-MIT

package tui

import (
	"testing"
	"time"

	"github.com/llbbl/repjan/internal/github"
	"github.com/llbbl/repjan/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestFormatRelativeTime(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		input    time.Time
		expected string
	}{
		{"zero time", time.Time{}, "unknown"},
		{"today", now, "today"},
		{"yesterday", now.AddDate(0, 0, -1), "1 day ago"},
		{"2 days ago", now.AddDate(0, 0, -2), "2 days ago"},
		{"week ago", now.AddDate(0, 0, -7), "7 days ago"},
		{"month ago", now.AddDate(0, -1, 0), "30 days ago"},    // approximately
		{"year ago", now.AddDate(-1, 0, 0), "365 days ago"},    // approximately
		{"2 years ago", now.AddDate(-2, 0, 0), "730 days ago"}, // approximately
		{"future date far", now.AddDate(1, 0, 0), "in the future"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatRelativeTime(tt.input)
			// For time-based tests, we may need fuzzy matching
			// since exact days can vary based on months/leap years
			if tt.name == "today" || tt.name == "yesterday" ||
				tt.name == "zero time" || tt.name == "future date far" {
				assert.Equal(t, tt.expected, got)
			} else {
				// Check it contains "days ago" for longer durations
				// where exact day count may vary
				assert.Contains(t, got, "days ago")
			}
		})
	}
}

func TestFormatRelativeTime_EdgeCases(t *testing.T) {
	now := time.Now()

	t.Run("very old date (10+ years)", func(t *testing.T) {
		tenYearsAgo := now.AddDate(-10, 0, 0)
		got := formatRelativeTime(tenYearsAgo)
		// Should be approximately 3650 days, but varies with leap years
		assert.Contains(t, got, "days ago")
		// Verify it's a large number (at least 3600 days)
		assert.Regexp(t, `^\d{4} days ago$`, got)
	})

	t.Run("very old date (50 years)", func(t *testing.T) {
		fiftyYearsAgo := now.AddDate(-50, 0, 0)
		got := formatRelativeTime(fiftyYearsAgo)
		assert.Contains(t, got, "days ago")
		// Should be approximately 18250 days
		assert.Regexp(t, `^\d{5} days ago$`, got)
	})

	t.Run("date just before midnight boundary", func(t *testing.T) {
		// Create a time that's 23 hours and 59 minutes ago
		// This should still be "today" since we calculate by days
		almostYesterday := now.Add(-23*time.Hour - 59*time.Minute)
		got := formatRelativeTime(almostYesterday)
		assert.Equal(t, "today", got)
	})

	t.Run("date just after midnight boundary", func(t *testing.T) {
		// Create a time that's exactly 24 hours ago
		// This should be "1 day ago"
		exactlyYesterday := now.Add(-24 * time.Hour)
		got := formatRelativeTime(exactlyYesterday)
		assert.Equal(t, "1 day ago", got)
	})

	t.Run("date 25 hours ago", func(t *testing.T) {
		// Should still be "1 day ago" since it's between 24-48 hours
		twentyFiveHoursAgo := now.Add(-25 * time.Hour)
		got := formatRelativeTime(twentyFiveHoursAgo)
		assert.Equal(t, "1 day ago", got)
	})

	t.Run("date 47 hours ago", func(t *testing.T) {
		// Should still be "1 day ago" since it's less than 48 hours
		fortySevenHoursAgo := now.Add(-47 * time.Hour)
		got := formatRelativeTime(fortySevenHoursAgo)
		assert.Equal(t, "1 day ago", got)
	})

	t.Run("date 48 hours ago", func(t *testing.T) {
		// Should be "2 days ago"
		fortyEightHoursAgo := now.Add(-48 * time.Hour)
		got := formatRelativeTime(fortyEightHoursAgo)
		assert.Equal(t, "2 days ago", got)
	})

	t.Run("future date far in future", func(t *testing.T) {
		farFuture := now.AddDate(10, 0, 0)
		got := formatRelativeTime(farFuture)
		assert.Equal(t, "in the future", got)
	})

	t.Run("future date 1 hour from now", func(t *testing.T) {
		oneHourFuture := now.Add(1 * time.Hour)
		got := formatRelativeTime(oneHourFuture)
		// Note: Due to integer truncation, times less than 24 hours in the future
		// are treated as "today" because int(-0.04) = 0, not -1.
		// This is an implementation quirk where very near future times
		// (within 24 hours) are reported as "today" rather than "in the future".
		assert.Equal(t, "today", got)
	})

	t.Run("future date 25 hours from now", func(t *testing.T) {
		twentyFiveHoursFuture := now.Add(25 * time.Hour)
		got := formatRelativeTime(twentyFiveHoursFuture)
		// At 25 hours in future, int(-25/24) = int(-1.04) = -1, which is < 0
		assert.Equal(t, "in the future", got)
	})
}

func TestFormatRelativeTime_SpecificDays(t *testing.T) {
	now := time.Now()

	// Test specific day counts to verify exact formatting
	dayTests := []struct {
		days     int
		expected string
	}{
		{0, "today"},
		{1, "1 day ago"},
		{2, "2 days ago"},
		{7, "7 days ago"},
		{30, "30 days ago"},
		{100, "100 days ago"},
		{365, "365 days ago"},
		{1000, "1000 days ago"},
	}

	for _, tt := range dayTests {
		t.Run(tt.expected, func(t *testing.T) {
			// Use hours to get exact day count
			pastTime := now.Add(-time.Duration(tt.days*24) * time.Hour)
			got := formatRelativeTime(pastTime)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestTruncateWithEllipsis(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "string shorter than max unchanged",
			input:    "hello",
			maxLen:   10,
			expected: "hello",
		},
		{
			name:     "string exactly max length unchanged",
			input:    "hello",
			maxLen:   5,
			expected: "hello",
		},
		{
			name:     "string longer than max truncated with ellipsis",
			input:    "hello world",
			maxLen:   8,
			expected: "hello...",
		},
		{
			name:     "empty string unchanged",
			input:    "",
			maxLen:   10,
			expected: "",
		},
		{
			name:     "max length less than 4 gets adjusted to 4",
			input:    "hello",
			maxLen:   3,
			expected: "h...",
		},
		{
			name:     "max length of 4 with long string",
			input:    "hello",
			maxLen:   4,
			expected: "h...",
		},
		{
			name:     "single character with max 4",
			input:    "h",
			maxLen:   4,
			expected: "h",
		},
		{
			name:     "exactly 4 chars with max 4",
			input:    "test",
			maxLen:   4,
			expected: "test",
		},
		{
			name:     "5 chars with max 4 truncates",
			input:    "tests",
			maxLen:   4,
			expected: "t...",
		},
		{
			name:     "long repository name truncation",
			input:    "very-long-repository-name-that-exceeds-limit",
			maxLen:   25,
			expected: "very-long-repository-n...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateWithEllipsis(tt.input, tt.maxLen)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTruncateWithEllipsis_EdgeCases(t *testing.T) {
	t.Run("max length of 0 gets adjusted to 4", func(t *testing.T) {
		result := truncateWithEllipsis("hello", 0)
		// max is adjusted to 4, so "hello" (5 chars) becomes "h..."
		assert.Equal(t, "h...", result)
	})

	t.Run("max length of 1 gets adjusted to 4", func(t *testing.T) {
		result := truncateWithEllipsis("hello", 1)
		assert.Equal(t, "h...", result)
	})

	t.Run("negative max length gets adjusted to 4", func(t *testing.T) {
		result := truncateWithEllipsis("hello", -5)
		assert.Equal(t, "h...", result)
	})

	t.Run("very long string", func(t *testing.T) {
		longStr := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
		result := truncateWithEllipsis(longStr, 10)
		assert.Equal(t, "abcdefg...", result)
		assert.Len(t, result, 10)
	})
}

func TestGetStatusIcon(t *testing.T) {
	tests := []struct {
		name     string
		repo     github.Repository
		expected string
	}{
		{
			name:     "archived repo returns archived icon",
			repo:     testutil.NewTestRepo(testutil.WithArchived(true)),
			expected: iconArchived,
		},
		{
			name: "archive candidate returns candidate icon",
			// A repo is an archive candidate if inactive > 365 days and no engagement
			repo:     testutil.NewTestRepo(testutil.WithDaysInactive(400), testutil.WithStars(0), testutil.WithForks(0)),
			expected: iconCandidate,
		},
		{
			name: "active repo returns active icon",
			// Recent activity and some engagement
			repo:     testutil.NewTestRepo(testutil.WithDaysInactive(30), testutil.WithStars(10)),
			expected: iconActive,
		},
		{
			name: "archived takes precedence over candidate criteria",
			// Even if it meets candidate criteria, archived should win
			repo:     testutil.NewTestRepo(testutil.WithArchived(true), testutil.WithDaysInactive(400), testutil.WithStars(0), testutil.WithForks(0)),
			expected: iconArchived,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getStatusIcon(tt.repo)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetStatusText(t *testing.T) {
	tests := []struct {
		name     string
		repo     github.Repository
		expected string
	}{
		{
			name:     "archived repo returns Archived",
			repo:     testutil.NewTestRepo(testutil.WithArchived(true)),
			expected: "Archived",
		},
		{
			name: "archive candidate returns Candidate",
			// A repo is an archive candidate if inactive > 365 days and no engagement
			repo:     testutil.NewTestRepo(testutil.WithDaysInactive(400), testutil.WithStars(0), testutil.WithForks(0)),
			expected: "Candidate",
		},
		{
			name: "active repo returns Active",
			// Recent activity and some engagement
			repo:     testutil.NewTestRepo(testutil.WithDaysInactive(30), testutil.WithStars(10)),
			expected: "Active",
		},
		{
			name: "archived takes precedence over candidate criteria",
			// Even if it meets candidate criteria, archived should win
			repo:     testutil.NewTestRepo(testutil.WithArchived(true), testutil.WithDaysInactive(400), testutil.WithStars(0), testutil.WithForks(0)),
			expected: "Archived",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getStatusText(tt.repo)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetStatusIcon_ArchiveCandidateCriteria(t *testing.T) {
	// Test various conditions that make a repo an archive candidate
	// Based on analyze.IsArchiveCandidate logic:
	// - No activity in 1+ year (365+ days) or 2+ years (730+ days)
	// - No community engagement (0 stars AND 0 forks)
	// - Stale fork (is fork AND > 180 days inactive)
	// - Legacy language and inactive > 365 days

	tests := []struct {
		name        string
		repo        github.Repository
		isCandidate bool
	}{
		{
			name:        "old repo with no engagement is candidate",
			repo:        testutil.NewTestRepo(testutil.WithDaysInactive(400), testutil.WithStars(0), testutil.WithForks(0)),
			isCandidate: true,
		},
		{
			name:        "old repo with stars is candidate (inactivity alone qualifies)",
			repo:        testutil.NewTestRepo(testutil.WithDaysInactive(400), testutil.WithStars(10)),
			isCandidate: true,
		},
		{
			name:        "recent repo with no engagement is candidate",
			repo:        testutil.NewTestRepo(testutil.WithDaysInactive(100), testutil.WithStars(0), testutil.WithForks(0)),
			isCandidate: true,
		},
		{
			name:        "recent repo with engagement is not candidate",
			repo:        testutil.NewTestRepo(testutil.WithDaysInactive(100), testutil.WithStars(5)),
			isCandidate: false,
		},
		{
			name:        "stale fork is candidate",
			repo:        testutil.NewTestRepo(testutil.WithFork(true), testutil.WithDaysInactive(200), testutil.WithStars(5)),
			isCandidate: true,
		},
		{
			name:        "recent fork is not candidate (if has engagement)",
			repo:        testutil.NewTestRepo(testutil.WithFork(true), testutil.WithDaysInactive(100), testutil.WithStars(5)),
			isCandidate: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			icon := getStatusIcon(tt.repo)
			if tt.isCandidate {
				assert.Equal(t, iconCandidate, icon, "expected candidate icon")
			} else {
				assert.Equal(t, iconActive, icon, "expected active icon")
			}
		})
	}
}

func TestGetStatusText_ArchiveCandidateCriteria(t *testing.T) {
	// Same tests as icon but for text
	tests := []struct {
		name        string
		repo        github.Repository
		isCandidate bool
	}{
		{
			name:        "old repo with no engagement is candidate",
			repo:        testutil.NewTestRepo(testutil.WithDaysInactive(400), testutil.WithStars(0), testutil.WithForks(0)),
			isCandidate: true,
		},
		{
			name:        "recent repo with engagement is not candidate",
			repo:        testutil.NewTestRepo(testutil.WithDaysInactive(100), testutil.WithStars(5)),
			isCandidate: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text := getStatusText(tt.repo)
			if tt.isCandidate {
				assert.Equal(t, "Candidate", text)
			} else {
				assert.Equal(t, "Active", text)
			}
		})
	}
}
