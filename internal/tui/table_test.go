// SPDX-FileCopyrightText: 2026 Logan Lindquist Land
// SPDX-License-Identifier: FSL-1.1-MIT

package tui

import (
	"testing"
	"time"

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
		{"month ago", now.AddDate(0, -1, 0), "30 days ago"},   // approximately
		{"year ago", now.AddDate(-1, 0, 0), "365 days ago"},   // approximately
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
