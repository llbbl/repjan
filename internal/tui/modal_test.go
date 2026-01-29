// SPDX-FileCopyrightText: 2026 Logan Lindquist Land
// SPDX-License-Identifier: FSL-1.1-MIT

package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatDaysAgo(t *testing.T) {
	tests := []struct {
		name     string
		days     int
		expected string
	}{
		// Zero/today
		{
			name:     "0 days is today",
			days:     0,
			expected: "today",
		},
		// Yesterday
		{
			name:     "1 day is yesterday",
			days:     1,
			expected: "yesterday",
		},
		// Days (2-6)
		{
			name:     "2 days",
			days:     2,
			expected: "2 days ago",
		},
		{
			name:     "6 days",
			days:     6,
			expected: "6 days ago",
		},
		// Weeks boundary (7-13 is 1 week)
		{
			name:     "7 days is 1 week",
			days:     7,
			expected: "1 week ago",
		},
		{
			name:     "13 days is still 1 week",
			days:     13,
			expected: "1 week ago",
		},
		// Multiple weeks (14-29)
		{
			name:     "14 days is 2 weeks",
			days:     14,
			expected: "2 weeks ago",
		},
		{
			name:     "21 days is 3 weeks",
			days:     21,
			expected: "3 weeks ago",
		},
		{
			name:     "28 days is 4 weeks",
			days:     28,
			expected: "4 weeks ago",
		},
		{
			name:     "29 days is still 4 weeks",
			days:     29,
			expected: "4 weeks ago",
		},
		// Months boundary (30-364)
		{
			name:     "30 days is 1 month",
			days:     30,
			expected: "1 month ago",
		},
		{
			name:     "59 days is still 1 month",
			days:     59,
			expected: "1 month ago",
		},
		{
			name:     "60 days is 2 months",
			days:     60,
			expected: "2 months ago",
		},
		{
			name:     "90 days is 3 months",
			days:     90,
			expected: "3 months ago",
		},
		{
			name:     "180 days is 6 months",
			days:     180,
			expected: "6 months ago",
		},
		{
			name:     "364 days is 12 months",
			days:     364,
			expected: "12 months ago",
		},
		// Years boundary (365+)
		{
			name:     "365 days is 1 year",
			days:     365,
			expected: "1 year ago",
		},
		{
			name:     "729 days is still 1 year",
			days:     729,
			expected: "1 year ago",
		},
		{
			name:     "730 days is 2 years",
			days:     730,
			expected: "2 years ago",
		},
		{
			name:     "1095 days is 3 years",
			days:     1095,
			expected: "3 years ago",
		},
		{
			name:     "3650 days is 10 years",
			days:     3650,
			expected: "10 years ago",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDaysAgo(tt.days)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatDaysAgo_BoundaryConditions(t *testing.T) {
	// Test the boundaries between each range
	tests := []struct {
		name        string
		days        int
		expectedNot string // What it should NOT be
	}{
		// Day/Week boundary: 6 days = "X days ago", 7 days = "1 week ago"
		{
			name:        "6 days should not be weeks",
			days:        6,
			expectedNot: "week",
		},
		// Week/Month boundary: 29 days = "X weeks ago", 30 days = "1 month ago"
		{
			name:        "29 days should not be months",
			days:        29,
			expectedNot: "month",
		},
		// Month/Year boundary: 364 days = "X months ago", 365 days = "1 year ago"
		{
			name:        "364 days should not be years",
			days:        364,
			expectedNot: "year",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDaysAgo(tt.days)
			assert.NotContains(t, result, tt.expectedNot)
		})
	}
}

func TestSortLanguagesByCount(t *testing.T) {
	tests := []struct {
		name     string
		names    []string
		counts   map[string]int
		expected []string
	}{
		{
			name:     "empty slice returns empty",
			names:    []string{},
			counts:   map[string]int{},
			expected: []string{},
		},
		{
			name:     "single element returns same",
			names:    []string{"Go"},
			counts:   map[string]int{"Go": 5},
			expected: []string{"Go"},
		},
		{
			name:     "multiple elements sorted by count descending",
			names:    []string{"Go", "Rust", "Python"},
			counts:   map[string]int{"Go": 10, "Rust": 5, "Python": 20},
			expected: []string{"Python", "Go", "Rust"},
		},
		{
			name:     "equal counts sorted alphabetically ascending",
			names:    []string{"Rust", "Go", "Python"},
			counts:   map[string]int{"Go": 5, "Rust": 5, "Python": 5},
			expected: []string{"Go", "Python", "Rust"},
		},
		{
			name:     "mix of different counts and alphabetical ties",
			names:    []string{"Rust", "Go", "Python", "Java", "C"},
			counts:   map[string]int{"Go": 10, "Rust": 5, "Python": 10, "Java": 5, "C": 3},
			expected: []string{"Go", "Python", "Java", "Rust", "C"},
		},
		{
			name:     "two elements same count sorted alphabetically",
			names:    []string{"Zebra", "Apple"},
			counts:   map[string]int{"Zebra": 10, "Apple": 10},
			expected: []string{"Apple", "Zebra"},
		},
		{
			name:     "already sorted remains sorted",
			names:    []string{"A", "B", "C"},
			counts:   map[string]int{"A": 30, "B": 20, "C": 10},
			expected: []string{"A", "B", "C"},
		},
		{
			name:     "reverse order gets sorted",
			names:    []string{"C", "B", "A"},
			counts:   map[string]int{"A": 30, "B": 20, "C": 10},
			expected: []string{"A", "B", "C"},
		},
		{
			name:     "zero counts sorted correctly",
			names:    []string{"Go", "None"},
			counts:   map[string]int{"Go": 5, "None": 0},
			expected: []string{"Go", "None"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make a copy since the function sorts in place
			namesCopy := make([]string, len(tt.names))
			copy(namesCopy, tt.names)

			sortLanguagesByCount(namesCopy, tt.counts)
			assert.Equal(t, tt.expected, namesCopy)
		})
	}
}

func TestSortLanguagesByCount_ModifiesSliceInPlace(t *testing.T) {
	names := []string{"C", "B", "A"}
	counts := map[string]int{"A": 30, "B": 20, "C": 10}

	sortLanguagesByCount(names, counts)

	// The original slice should be modified
	assert.Equal(t, []string{"A", "B", "C"}, names)
}
