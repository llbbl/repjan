// SPDX-FileCopyrightText: 2026 Logan Lindquist Land
// SPDX-License-Identifier: FSL-1.1-MIT

package tui

import (
	"testing"

	"github.com/llbbl/repjan/internal/github"
	"github.com/llbbl/repjan/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilterRepos(t *testing.T) {
	tests := []struct {
		name     string
		repos    []github.Repository
		filter   Filter
		language string
		wantLen  int
	}{
		{
			name:     "empty repo list returns empty",
			repos:    []github.Repository{},
			filter:   FilterAll,
			language: "",
			wantLen:  0,
		},
		{
			name: "single repo passes through",
			repos: []github.Repository{
				testutil.NewTestRepo(),
			},
			filter:   FilterAll,
			language: "",
			wantLen:  1,
		},
		{
			name: "all filter excludes archived",
			repos: []github.Repository{
				testutil.NewTestRepo(testutil.WithName("active")),
				testutil.NewTestRepo(testutil.WithName("archived"), testutil.WithArchived(true)),
			},
			filter:   FilterAll,
			language: "",
			wantLen:  1,
		},
		{
			name: "all filter includes all non-archived repos",
			repos: []github.Repository{
				testutil.NewTestRepo(testutil.WithName("repo1")),
				testutil.NewTestRepo(testutil.WithName("repo2")),
				testutil.NewTestRepo(testutil.WithName("repo3")),
			},
			filter:   FilterAll,
			language: "",
			wantLen:  3,
		},
		{
			name: "old filter includes repos inactive for more than 365 days",
			repos: []github.Repository{
				testutil.NewTestRepo(testutil.WithName("old"), testutil.WithDaysInactive(366)),
				testutil.NewTestRepo(testutil.WithName("recent"), testutil.WithDaysInactive(100)),
			},
			filter:   FilterOld,
			language: "",
			wantLen:  1,
		},
		{
			name: "old filter excludes exactly 365 days",
			repos: []github.Repository{
				testutil.NewTestRepo(testutil.WithName("boundary"), testutil.WithDaysInactive(365)),
				testutil.NewTestRepo(testutil.WithName("old"), testutil.WithDaysInactive(400)),
			},
			filter:   FilterOld,
			language: "",
			wantLen:  1,
		},
		{
			name: "old filter excludes archived even if old",
			repos: []github.Repository{
				testutil.NewTestRepo(testutil.WithName("old-archived"), testutil.WithDaysInactive(500), testutil.WithArchived(true)),
				testutil.NewTestRepo(testutil.WithName("old-active"), testutil.WithDaysInactive(400)),
			},
			filter:   FilterOld,
			language: "",
			wantLen:  1,
		},
		{
			name: "no stars filter includes only zero star repos",
			repos: []github.Repository{
				testutil.NewTestRepo(testutil.WithName("popular"), testutil.WithStars(100)),
				testutil.NewTestRepo(testutil.WithName("unpopular"), testutil.WithStars(0)),
			},
			filter:   FilterNoStars,
			language: "",
			wantLen:  1,
		},
		{
			name: "no stars filter excludes archived zero star repos",
			repos: []github.Repository{
				testutil.NewTestRepo(testutil.WithName("no-stars-archived"), testutil.WithStars(0), testutil.WithArchived(true)),
				testutil.NewTestRepo(testutil.WithName("no-stars-active"), testutil.WithStars(0)),
			},
			filter:   FilterNoStars,
			language: "",
			wantLen:  1,
		},
		{
			name: "forks filter includes only forks",
			repos: []github.Repository{
				testutil.NewTestRepo(testutil.WithName("original"), testutil.WithFork(false)),
				testutil.NewTestRepo(testutil.WithName("forked"), testutil.WithFork(true)),
			},
			filter:   FilterForks,
			language: "",
			wantLen:  1,
		},
		{
			name: "forks filter excludes archived forks",
			repos: []github.Repository{
				testutil.NewTestRepo(testutil.WithName("fork-archived"), testutil.WithFork(true), testutil.WithArchived(true)),
				testutil.NewTestRepo(testutil.WithName("fork-active"), testutil.WithFork(true)),
			},
			filter:   FilterForks,
			language: "",
			wantLen:  1,
		},
		{
			name: "private filter includes only private repos",
			repos: []github.Repository{
				testutil.NewTestRepo(testutil.WithName("public"), testutil.WithPrivate(false)),
				testutil.NewTestRepo(testutil.WithName("private"), testutil.WithPrivate(true)),
			},
			filter:   FilterPrivate,
			language: "",
			wantLen:  1,
		},
		{
			name: "private filter excludes archived private repos",
			repos: []github.Repository{
				testutil.NewTestRepo(testutil.WithName("private-archived"), testutil.WithPrivate(true), testutil.WithArchived(true)),
				testutil.NewTestRepo(testutil.WithName("private-active"), testutil.WithPrivate(true)),
			},
			filter:   FilterPrivate,
			language: "",
			wantLen:  1,
		},
		{
			name: "language filter filters by language",
			repos: []github.Repository{
				testutil.NewTestRepo(testutil.WithName("go-repo"), testutil.WithLanguage("Go")),
				testutil.NewTestRepo(testutil.WithName("rust-repo"), testutil.WithLanguage("Rust")),
				testutil.NewTestRepo(testutil.WithName("python-repo"), testutil.WithLanguage("Python")),
			},
			filter:   FilterAll,
			language: "Go",
			wantLen:  1,
		},
		{
			name: "language filter is case sensitive",
			repos: []github.Repository{
				testutil.NewTestRepo(testutil.WithName("go-repo"), testutil.WithLanguage("Go")),
				testutil.NewTestRepo(testutil.WithName("GO-repo"), testutil.WithLanguage("GO")),
			},
			filter:   FilterAll,
			language: "Go",
			wantLen:  1,
		},
		{
			name: "language filter combined with type filter",
			repos: []github.Repository{
				testutil.NewTestRepo(testutil.WithName("go-fork"), testutil.WithLanguage("Go"), testutil.WithFork(true)),
				testutil.NewTestRepo(testutil.WithName("go-original"), testutil.WithLanguage("Go"), testutil.WithFork(false)),
				testutil.NewTestRepo(testutil.WithName("rust-fork"), testutil.WithLanguage("Rust"), testutil.WithFork(true)),
			},
			filter:   FilterForks,
			language: "Go",
			wantLen:  1,
		},
		{
			name: "empty language filter includes all languages",
			repos: []github.Repository{
				testutil.NewTestRepo(testutil.WithName("go-repo"), testutil.WithLanguage("Go")),
				testutil.NewTestRepo(testutil.WithName("rust-repo"), testutil.WithLanguage("Rust")),
				testutil.NewTestRepo(testutil.WithName("no-lang"), testutil.WithLanguage("")),
			},
			filter:   FilterAll,
			language: "",
			wantLen:  3,
		},
		{
			name: "all repos archived returns empty",
			repos: []github.Repository{
				testutil.NewTestRepo(testutil.WithName("archived1"), testutil.WithArchived(true)),
				testutil.NewTestRepo(testutil.WithName("archived2"), testutil.WithArchived(true)),
			},
			filter:   FilterAll,
			language: "",
			wantLen:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterRepos(tt.repos, tt.filter, tt.language)
			assert.Len(t, result, tt.wantLen)
		})
	}
}

func TestFilterRepos_PreservesRepoData(t *testing.T) {
	// Verify that filtering doesn't mutate or lose repository data
	original := testutil.NewTestRepo(
		testutil.WithName("test-repo"),
		testutil.WithOwner("test-owner"),
		testutil.WithStars(42),
		testutil.WithLanguage("Go"),
		testutil.WithDescription("Test description"),
	)

	repos := []github.Repository{original}
	result := filterRepos(repos, FilterAll, "")

	require.Len(t, result, 1)
	assert.Equal(t, "test-repo", result[0].Name)
	assert.Equal(t, "test-owner", result[0].Owner)
	assert.Equal(t, 42, result[0].StargazerCount)
	assert.Equal(t, "Go", result[0].PrimaryLanguage)
	assert.Equal(t, "Test description", result[0].Description)
}

func TestSortRepos(t *testing.T) {
	tests := []struct {
		name      string
		repos     []github.Repository
		field     SortField
		ascending bool
		wantFirst string
		wantLast  string
	}{
		{
			name:      "empty repo list returns empty",
			repos:     []github.Repository{},
			field:     SortName,
			ascending: true,
			wantFirst: "",
			wantLast:  "",
		},
		{
			name: "single repo returns same repo",
			repos: []github.Repository{
				testutil.NewTestRepo(testutil.WithName("only")),
			},
			field:     SortName,
			ascending: true,
			wantFirst: "only",
			wantLast:  "only",
		},
		{
			name: "sort by name ascending",
			repos: []github.Repository{
				testutil.NewTestRepo(testutil.WithName("zebra")),
				testutil.NewTestRepo(testutil.WithName("alpha")),
				testutil.NewTestRepo(testutil.WithName("mike")),
			},
			field:     SortName,
			ascending: true,
			wantFirst: "alpha",
			wantLast:  "zebra",
		},
		{
			name: "sort by name descending",
			repos: []github.Repository{
				testutil.NewTestRepo(testutil.WithName("zebra")),
				testutil.NewTestRepo(testutil.WithName("alpha")),
				testutil.NewTestRepo(testutil.WithName("mike")),
			},
			field:     SortName,
			ascending: false,
			wantFirst: "zebra",
			wantLast:  "alpha",
		},
		{
			name: "sort by name is case insensitive",
			repos: []github.Repository{
				testutil.NewTestRepo(testutil.WithName("Zebra")),
				testutil.NewTestRepo(testutil.WithName("alpha")),
				testutil.NewTestRepo(testutil.WithName("MIKE")),
			},
			field:     SortName,
			ascending: true,
			wantFirst: "alpha",
			wantLast:  "Zebra",
		},
		{
			name: "sort by activity ascending (most recent first)",
			repos: []github.Repository{
				testutil.NewTestRepo(testutil.WithName("old"), testutil.WithDaysInactive(100)),
				testutil.NewTestRepo(testutil.WithName("recent"), testutil.WithDaysInactive(10)),
				testutil.NewTestRepo(testutil.WithName("ancient"), testutil.WithDaysInactive(500)),
			},
			field:     SortActivity,
			ascending: true,
			wantFirst: "recent",
			wantLast:  "ancient",
		},
		{
			name: "sort by activity descending (oldest first)",
			repos: []github.Repository{
				testutil.NewTestRepo(testutil.WithName("old"), testutil.WithDaysInactive(100)),
				testutil.NewTestRepo(testutil.WithName("recent"), testutil.WithDaysInactive(10)),
				testutil.NewTestRepo(testutil.WithName("ancient"), testutil.WithDaysInactive(500)),
			},
			field:     SortActivity,
			ascending: false,
			wantFirst: "ancient",
			wantLast:  "recent",
		},
		{
			name: "sort by stars ascending",
			repos: []github.Repository{
				testutil.NewTestRepo(testutil.WithName("popular"), testutil.WithStars(100)),
				testutil.NewTestRepo(testutil.WithName("unpopular"), testutil.WithStars(0)),
				testutil.NewTestRepo(testutil.WithName("medium"), testutil.WithStars(50)),
			},
			field:     SortStars,
			ascending: true,
			wantFirst: "unpopular",
			wantLast:  "popular",
		},
		{
			name: "sort by stars descending",
			repos: []github.Repository{
				testutil.NewTestRepo(testutil.WithName("popular"), testutil.WithStars(100)),
				testutil.NewTestRepo(testutil.WithName("unpopular"), testutil.WithStars(0)),
				testutil.NewTestRepo(testutil.WithName("medium"), testutil.WithStars(50)),
			},
			field:     SortStars,
			ascending: false,
			wantFirst: "popular",
			wantLast:  "unpopular",
		},
		{
			name: "sort by language ascending",
			repos: []github.Repository{
				testutil.NewTestRepo(testutil.WithName("rust-repo"), testutil.WithLanguage("Rust")),
				testutil.NewTestRepo(testutil.WithName("go-repo"), testutil.WithLanguage("Go")),
				testutil.NewTestRepo(testutil.WithName("python-repo"), testutil.WithLanguage("Python")),
			},
			field:     SortLanguage,
			ascending: true,
			wantFirst: "go-repo",
			wantLast:  "rust-repo",
		},
		{
			name: "sort by language descending",
			repos: []github.Repository{
				testutil.NewTestRepo(testutil.WithName("rust-repo"), testutil.WithLanguage("Rust")),
				testutil.NewTestRepo(testutil.WithName("go-repo"), testutil.WithLanguage("Go")),
				testutil.NewTestRepo(testutil.WithName("python-repo"), testutil.WithLanguage("Python")),
			},
			field:     SortLanguage,
			ascending: false,
			wantFirst: "rust-repo",
			wantLast:  "go-repo",
		},
		{
			name: "sort by language is case insensitive",
			repos: []github.Repository{
				testutil.NewTestRepo(testutil.WithName("rust-repo"), testutil.WithLanguage("RUST")),
				testutil.NewTestRepo(testutil.WithName("go-repo"), testutil.WithLanguage("go")),
				testutil.NewTestRepo(testutil.WithName("python-repo"), testutil.WithLanguage("Python")),
			},
			field:     SortLanguage,
			ascending: true,
			wantFirst: "go-repo",
			wantLast:  "rust-repo",
		},
		{
			name: "unknown sort field defaults to name",
			repos: []github.Repository{
				testutil.NewTestRepo(testutil.WithName("zebra")),
				testutil.NewTestRepo(testutil.WithName("alpha")),
			},
			field:     SortField(99), // invalid field
			ascending: true,
			wantFirst: "alpha",
			wantLast:  "zebra",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sortRepos(tt.repos, tt.field, tt.ascending)

			if len(tt.repos) == 0 {
				assert.Empty(t, result)
				return
			}

			assert.Equal(t, tt.wantFirst, result[0].Name)
			assert.Equal(t, tt.wantLast, result[len(result)-1].Name)
		})
	}
}

func TestSortRepos_StableSort(t *testing.T) {
	// Verify that stable sort preserves relative order of equal elements
	repos := []github.Repository{
		testutil.NewTestRepo(testutil.WithName("first"), testutil.WithStars(10)),
		testutil.NewTestRepo(testutil.WithName("second"), testutil.WithStars(10)),
		testutil.NewTestRepo(testutil.WithName("third"), testutil.WithStars(10)),
	}

	result := sortRepos(repos, SortStars, true)

	// With stable sort, order should be preserved when values are equal
	require.Len(t, result, 3)
	assert.Equal(t, "first", result[0].Name)
	assert.Equal(t, "second", result[1].Name)
	assert.Equal(t, "third", result[2].Name)
}

func TestSortRepos_DoesNotMutateOriginal(t *testing.T) {
	original := []github.Repository{
		testutil.NewTestRepo(testutil.WithName("zebra")),
		testutil.NewTestRepo(testutil.WithName("alpha")),
	}

	// Store original order
	firstName := original[0].Name
	secondName := original[1].Name

	_ = sortRepos(original, SortName, true)

	// Verify original slice is unchanged
	assert.Equal(t, firstName, original[0].Name)
	assert.Equal(t, secondName, original[1].Name)
}

func TestGetUniqueLanguages(t *testing.T) {
	tests := []struct {
		name      string
		repos     []github.Repository
		wantLangs []string
	}{
		{
			name:      "empty repo list returns empty",
			repos:     []github.Repository{},
			wantLangs: []string{},
		},
		{
			name: "single repo single language",
			repos: []github.Repository{
				testutil.NewTestRepo(testutil.WithLanguage("Go")),
			},
			wantLangs: []string{"Go"},
		},
		{
			name: "multiple repos same language",
			repos: []github.Repository{
				testutil.NewTestRepo(testutil.WithName("repo1"), testutil.WithLanguage("Go")),
				testutil.NewTestRepo(testutil.WithName("repo2"), testutil.WithLanguage("Go")),
			},
			wantLangs: []string{"Go"},
		},
		{
			name: "multiple repos different languages sorted",
			repos: []github.Repository{
				testutil.NewTestRepo(testutil.WithName("rust-repo"), testutil.WithLanguage("Rust")),
				testutil.NewTestRepo(testutil.WithName("go-repo"), testutil.WithLanguage("Go")),
				testutil.NewTestRepo(testutil.WithName("python-repo"), testutil.WithLanguage("Python")),
			},
			wantLangs: []string{"Go", "Python", "Rust"},
		},
		{
			name: "excludes empty language",
			repos: []github.Repository{
				testutil.NewTestRepo(testutil.WithName("go-repo"), testutil.WithLanguage("Go")),
				testutil.NewTestRepo(testutil.WithName("no-lang"), testutil.WithLanguage("")),
			},
			wantLangs: []string{"Go"},
		},
		{
			name: "all repos have empty language",
			repos: []github.Repository{
				testutil.NewTestRepo(testutil.WithName("repo1"), testutil.WithLanguage("")),
				testutil.NewTestRepo(testutil.WithName("repo2"), testutil.WithLanguage("")),
			},
			wantLangs: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getUniqueLanguages(tt.repos)
			assert.Equal(t, tt.wantLangs, result)
		})
	}
}
