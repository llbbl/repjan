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
			result := filterRepos(tt.repos, tt.filter, tt.language, filterOpts{})
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
	result := filterRepos(repos, FilterAll, "", filterOpts{})

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

func TestFilterRepos_VisibilityOptions(t *testing.T) {
	tests := []struct {
		name    string
		repos   []github.Repository
		opts    filterOpts
		wantLen int
	}{
		{
			name: "default hides private repos",
			repos: []github.Repository{
				testutil.NewTestRepo(testutil.WithName("public"), testutil.WithPrivate(false)),
				testutil.NewTestRepo(testutil.WithName("private"), testutil.WithPrivate(true)),
			},
			opts:    filterOpts{showPrivate: false, showArchived: false},
			wantLen: 1,
		},
		{
			name: "showPrivate includes private repos",
			repos: []github.Repository{
				testutil.NewTestRepo(testutil.WithName("public"), testutil.WithPrivate(false)),
				testutil.NewTestRepo(testutil.WithName("private"), testutil.WithPrivate(true)),
			},
			opts:    filterOpts{showPrivate: true, showArchived: false},
			wantLen: 2,
		},
		{
			name: "default hides archived repos",
			repos: []github.Repository{
				testutil.NewTestRepo(testutil.WithName("active")),
				testutil.NewTestRepo(testutil.WithName("archived"), testutil.WithArchived(true)),
			},
			opts:    filterOpts{showPrivate: false, showArchived: false},
			wantLen: 1,
		},
		{
			name: "showArchived includes archived repos",
			repos: []github.Repository{
				testutil.NewTestRepo(testutil.WithName("active")),
				testutil.NewTestRepo(testutil.WithName("archived"), testutil.WithArchived(true)),
			},
			opts:    filterOpts{showPrivate: false, showArchived: true},
			wantLen: 2,
		},
		{
			name: "all visibility options show all repos",
			repos: []github.Repository{
				testutil.NewTestRepo(testutil.WithName("public-active")),
				testutil.NewTestRepo(testutil.WithName("private-active"), testutil.WithPrivate(true)),
				testutil.NewTestRepo(testutil.WithName("public-archived"), testutil.WithArchived(true)),
				testutil.NewTestRepo(testutil.WithName("private-archived"), testutil.WithPrivate(true), testutil.WithArchived(true)),
			},
			opts:    filterOpts{showPrivate: true, showArchived: true},
			wantLen: 4,
		},
		{
			name: "visibility works with content filters",
			repos: []github.Repository{
				testutil.NewTestRepo(testutil.WithName("old-public"), testutil.WithDaysInactive(400)),
				testutil.NewTestRepo(testutil.WithName("old-private"), testutil.WithDaysInactive(400), testutil.WithPrivate(true)),
				testutil.NewTestRepo(testutil.WithName("recent-public"), testutil.WithDaysInactive(100)),
			},
			opts:    filterOpts{showPrivate: false, showArchived: false},
			wantLen: 1, // Only old-public matches (old filter + hidden private)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Default to FilterAll for visibility tests, except for the combined test
			filter := FilterAll
			if tt.name == "visibility works with content filters" {
				filter = FilterOld
			}
			result := filterRepos(tt.repos, filter, "", tt.opts)
			assert.Len(t, result, tt.wantLen)
		})
	}
}

func TestGetVisibilityLabel(t *testing.T) {
	tests := []struct {
		name         string
		showPrivate  bool
		showArchived bool
		want         string
	}{
		{
			name:         "default shows Public Active",
			showPrivate:  false,
			showArchived: false,
			want:         "Public Active",
		},
		{
			name:         "showArchived shows Public All",
			showPrivate:  false,
			showArchived: true,
			want:         "Public All",
		},
		{
			name:         "showPrivate shows Including Private",
			showPrivate:  true,
			showArchived: false,
			want:         "Including Private",
		},
		{
			name:         "both shows All Repos",
			showPrivate:  true,
			showArchived: true,
			want:         "All Repos",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Model{
				showPrivate:  tt.showPrivate,
				showArchived: tt.showArchived,
			}
			assert.Equal(t, tt.want, m.getVisibilityLabel())
		})
	}
}

func TestToggleShowPrivate(t *testing.T) {
	m := Model{
		repos:        []github.Repository{testutil.NewTestRepo()},
		showPrivate:  false,
		showArchived: false,
	}

	// Initially false
	assert.False(t, m.showPrivate)

	// Toggle on
	m.ToggleShowPrivate()
	assert.True(t, m.showPrivate)

	// Toggle off
	m.ToggleShowPrivate()
	assert.False(t, m.showPrivate)
}

func TestToggleShowArchived(t *testing.T) {
	m := Model{
		repos:        []github.Repository{testutil.NewTestRepo()},
		showPrivate:  false,
		showArchived: false,
	}

	// Initially false
	assert.False(t, m.showArchived)

	// Toggle on
	m.ToggleShowArchived()
	assert.True(t, m.showArchived)

	// Toggle off
	m.ToggleShowArchived()
	assert.False(t, m.showArchived)
}

// TestNewModel_DefaultVisibility verifies that NewModel creates a model with
// privacy-safe defaults (private and archived repos hidden).
func TestNewModel_DefaultVisibility(t *testing.T) {
	repos := []github.Repository{
		testutil.NewTestRepo(testutil.WithName("public")),
		testutil.NewTestRepo(testutil.WithName("private"), testutil.WithPrivate(true)),
		testutil.NewTestRepo(testutil.WithName("archived"), testutil.WithArchived(true)),
	}

	m := NewModel(repos, "testowner", nil, false, "", nil)

	// Verify default visibility flags
	assert.False(t, m.showPrivate, "showPrivate should default to false")
	assert.False(t, m.showArchived, "showArchived should default to false")

	// Verify filtered repos only includes public, non-archived repos
	require.Len(t, m.filteredRepos, 1)
	assert.Equal(t, "public", m.filteredRepos[0].Name)
}

// TestToggleShowPrivate_RefreshesFilteredRepos verifies that toggling private
// visibility immediately updates the filtered repos list.
func TestToggleShowPrivate_RefreshesFilteredRepos(t *testing.T) {
	publicRepo := testutil.NewTestRepo(testutil.WithName("public"), testutil.WithPrivate(false))
	privateRepo := testutil.NewTestRepo(testutil.WithName("private"), testutil.WithPrivate(true))

	m := Model{
		repos:        []github.Repository{publicRepo, privateRepo},
		showPrivate:  false,
		showArchived: false,
		sortField:    SortName,
	}

	// Initial state: only public visible
	m.RefreshFilteredRepos()
	require.Len(t, m.filteredRepos, 1)
	assert.Equal(t, "public", m.filteredRepos[0].Name)

	// Toggle on: both should be visible
	m.ToggleShowPrivate()
	require.Len(t, m.filteredRepos, 2)

	// Toggle off: back to public only
	m.ToggleShowPrivate()
	require.Len(t, m.filteredRepos, 1)
	assert.Equal(t, "public", m.filteredRepos[0].Name)
}

// TestToggleShowArchived_RefreshesFilteredRepos verifies that toggling archived
// visibility immediately updates the filtered repos list.
func TestToggleShowArchived_RefreshesFilteredRepos(t *testing.T) {
	activeRepo := testutil.NewTestRepo(testutil.WithName("active"), testutil.WithArchived(false))
	archivedRepo := testutil.NewTestRepo(testutil.WithName("archived"), testutil.WithArchived(true))

	m := Model{
		repos:        []github.Repository{activeRepo, archivedRepo},
		showPrivate:  false,
		showArchived: false,
		sortField:    SortName,
	}

	// Initial state: only active visible
	m.RefreshFilteredRepos()
	require.Len(t, m.filteredRepos, 1)
	assert.Equal(t, "active", m.filteredRepos[0].Name)

	// Toggle on: both should be visible
	m.ToggleShowArchived()
	require.Len(t, m.filteredRepos, 2)

	// Toggle off: back to active only
	m.ToggleShowArchived()
	require.Len(t, m.filteredRepos, 1)
	assert.Equal(t, "active", m.filteredRepos[0].Name)
}

// TestRenderPrivateWarning_ShowsWhenPrivateVisible verifies the warning banner
// behavior based on private repo visibility.
func TestRenderPrivateWarning_ShowsWhenPrivateVisible(t *testing.T) {
	tests := []struct {
		name        string
		showPrivate bool
		wantEmpty   bool
	}{
		{
			name:        "warning shown when private repos visible",
			showPrivate: true,
			wantEmpty:   false,
		},
		{
			name:        "warning hidden when private repos hidden",
			showPrivate: false,
			wantEmpty:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Model{
				showPrivate: tt.showPrivate,
				width:       80,
				styles:      DefaultStyles(),
			}

			result := m.renderPrivateWarning()

			if tt.wantEmpty {
				assert.Empty(t, result, "warning should be empty when private repos hidden")
			} else {
				assert.NotEmpty(t, result, "warning should be shown when private repos visible")
				assert.Contains(t, result, "PRIVATE REPOS VISIBLE")
			}
		})
	}
}

// TestFilterRepos_AllPrivate verifies that when all repos are private,
// the result is empty with default visibility settings.
func TestFilterRepos_AllPrivate(t *testing.T) {
	repos := []github.Repository{
		testutil.NewTestRepo(testutil.WithName("private1"), testutil.WithPrivate(true)),
		testutil.NewTestRepo(testutil.WithName("private2"), testutil.WithPrivate(true)),
		testutil.NewTestRepo(testutil.WithName("private3"), testutil.WithPrivate(true)),
	}

	// Default opts (showPrivate=false) should return empty
	result := filterRepos(repos, FilterAll, "", filterOpts{showPrivate: false, showArchived: false})
	assert.Empty(t, result, "all private repos should be hidden with default visibility")

	// With showPrivate=true, all should be visible
	result = filterRepos(repos, FilterAll, "", filterOpts{showPrivate: true, showArchived: false})
	assert.Len(t, result, 3, "all private repos should be visible when showPrivate=true")
}

// TestFilterRepos_LanguageWithVisibility verifies that language filter
// works correctly in combination with visibility options.
func TestFilterRepos_LanguageWithVisibility(t *testing.T) {
	repos := []github.Repository{
		testutil.NewTestRepo(testutil.WithName("public-go"), testutil.WithLanguage("Go"), testutil.WithPrivate(false)),
		testutil.NewTestRepo(testutil.WithName("private-go"), testutil.WithLanguage("Go"), testutil.WithPrivate(true)),
		testutil.NewTestRepo(testutil.WithName("public-rust"), testutil.WithLanguage("Rust"), testutil.WithPrivate(false)),
		testutil.NewTestRepo(testutil.WithName("archived-go"), testutil.WithLanguage("Go"), testutil.WithArchived(true)),
	}

	tests := []struct {
		name     string
		opts     filterOpts
		language string
		wantLen  int
		wantName string // expected first repo name
	}{
		{
			name:     "language filter with default visibility",
			opts:     filterOpts{showPrivate: false, showArchived: false},
			language: "Go",
			wantLen:  1,
			wantName: "public-go",
		},
		{
			name:     "language filter with private visible",
			opts:     filterOpts{showPrivate: true, showArchived: false},
			language: "Go",
			wantLen:  2,
			wantName: "private-go", // alphabetically first
		},
		{
			name:     "language filter with archived visible",
			opts:     filterOpts{showPrivate: false, showArchived: true},
			language: "Go",
			wantLen:  2,
			wantName: "archived-go", // alphabetically first
		},
		{
			name:     "language filter with all visible",
			opts:     filterOpts{showPrivate: true, showArchived: true},
			language: "Go",
			wantLen:  3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterRepos(repos, FilterAll, tt.language, tt.opts)
			assert.Len(t, result, tt.wantLen)
			if tt.wantName != "" && len(result) > 0 {
				// Results are not sorted by filterRepos, so just check the name exists
				names := make([]string, len(result))
				for i, r := range result {
					names[i] = r.Name
				}
				assert.Contains(t, names, tt.wantName)
			}
		})
	}
}

// TestFilterRepos_ContentFiltersWithVisibility verifies that content filters
// (Old, NoStars, Forks) work correctly on top of visibility filters.
func TestFilterRepos_ContentFiltersWithVisibility(t *testing.T) {
	repos := []github.Repository{
		// Public, active repos for content filter testing
		testutil.NewTestRepo(testutil.WithName("old-public"), testutil.WithDaysInactive(400)),
		testutil.NewTestRepo(testutil.WithName("recent-public"), testutil.WithDaysInactive(100)),
		testutil.NewTestRepo(testutil.WithName("nostar-public"), testutil.WithStars(0)),
		testutil.NewTestRepo(testutil.WithName("starred-public"), testutil.WithStars(50)),
		testutil.NewTestRepo(testutil.WithName("fork-public"), testutil.WithFork(true)),
		testutil.NewTestRepo(testutil.WithName("original-public"), testutil.WithFork(false)),
		// Private versions
		testutil.NewTestRepo(testutil.WithName("old-private"), testutil.WithDaysInactive(400), testutil.WithPrivate(true)),
		testutil.NewTestRepo(testutil.WithName("nostar-private"), testutil.WithStars(0), testutil.WithPrivate(true)),
		testutil.NewTestRepo(testutil.WithName("fork-private"), testutil.WithFork(true), testutil.WithPrivate(true)),
	}

	tests := []struct {
		name    string
		filter  Filter
		opts    filterOpts
		wantLen int
	}{
		{
			name:    "old filter with default visibility",
			filter:  FilterOld,
			opts:    filterOpts{showPrivate: false, showArchived: false},
			wantLen: 1, // only old-public
		},
		{
			name:    "old filter with private visible",
			filter:  FilterOld,
			opts:    filterOpts{showPrivate: true, showArchived: false},
			wantLen: 2, // old-public + old-private
		},
		{
			name:    "nostar filter with default visibility",
			filter:  FilterNoStars,
			opts:    filterOpts{showPrivate: false, showArchived: false},
			wantLen: 1, // only nostar-public
		},
		{
			name:    "nostar filter with private visible",
			filter:  FilterNoStars,
			opts:    filterOpts{showPrivate: true, showArchived: false},
			wantLen: 2, // nostar-public + nostar-private
		},
		{
			name:    "forks filter with default visibility",
			filter:  FilterForks,
			opts:    filterOpts{showPrivate: false, showArchived: false},
			wantLen: 1, // only fork-public
		},
		{
			name:    "forks filter with private visible",
			filter:  FilterForks,
			opts:    filterOpts{showPrivate: true, showArchived: false},
			wantLen: 2, // fork-public + fork-private
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterRepos(repos, tt.filter, "", tt.opts)
			assert.Len(t, result, tt.wantLen)
		})
	}
}

// TestFilterRepos_NoneLanguageWithVisibility verifies "None" language filter
// (repos with no language) works with visibility options.
func TestFilterRepos_NoneLanguageWithVisibility(t *testing.T) {
	repos := []github.Repository{
		testutil.NewTestRepo(testutil.WithName("public-no-lang"), testutil.WithLanguage("")),
		testutil.NewTestRepo(testutil.WithName("private-no-lang"), testutil.WithLanguage(""), testutil.WithPrivate(true)),
		testutil.NewTestRepo(testutil.WithName("public-go"), testutil.WithLanguage("Go")),
	}

	// Default visibility: only public no-lang
	result := filterRepos(repos, FilterAll, "None", filterOpts{showPrivate: false, showArchived: false})
	require.Len(t, result, 1)
	assert.Equal(t, "public-no-lang", result[0].Name)

	// With private visible: both no-lang repos
	result = filterRepos(repos, FilterAll, "None", filterOpts{showPrivate: true, showArchived: false})
	assert.Len(t, result, 2)
}
