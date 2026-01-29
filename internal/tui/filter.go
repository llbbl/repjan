// SPDX-FileCopyrightText: 2026 Logan Lindquist Land
// SPDX-License-Identifier: FSL-1.1-MIT

package tui

import (
	"sort"
	"strings"

	"github.com/llbbl/repjan/internal/github"
)

// filterOpts contains visibility options for filtering repositories.
type filterOpts struct {
	showPrivate  bool
	showArchived bool
}

// filterRepos returns a filtered slice of repositories based on the filter type, language, and visibility options.
// By default (when opts has zero values), private and archived repos are hidden for privacy safety.
func filterRepos(repos []github.Repository, filter Filter, language string, opts filterOpts) []github.Repository {
	result := make([]github.Repository, 0, len(repos))

	for _, repo := range repos {
		// Apply visibility filters first (privacy-safe defaults)
		if !opts.showArchived && repo.IsArchived {
			continue
		}
		if !opts.showPrivate && repo.IsPrivate {
			continue
		}

		// Apply filter type
		switch filter {
		case FilterAll:
			// Include all visible repos
		case FilterOld:
			if repo.DaysSinceActivity <= 365 {
				continue
			}
		case FilterNoStars:
			if repo.StargazerCount != 0 {
				continue
			}
		case FilterForks:
			if !repo.IsFork {
				continue
			}
		}

		// Apply language filter if specified
		if language != "" {
			// "None" matches repos with empty PrimaryLanguage
			if language == "None" {
				if repo.PrimaryLanguage != "" {
					continue
				}
			} else if repo.PrimaryLanguage != language {
				continue
			}
		}

		result = append(result, repo)
	}

	return result
}

// sortRepos returns a sorted copy of the repositories slice.
// Uses stable sort to preserve relative order of equal elements.
func sortRepos(repos []github.Repository, field SortField, ascending bool) []github.Repository {
	// Create a copy to avoid mutating the original
	result := make([]github.Repository, len(repos))
	copy(result, repos)

	sort.SliceStable(result, func(i, j int) bool {
		var less bool

		switch field {
		case SortName:
			less = strings.ToLower(result[i].Name) < strings.ToLower(result[j].Name)
		case SortActivity:
			// Invert: ascending = oldest first (most days since activity)
			less = result[i].DaysSinceActivity > result[j].DaysSinceActivity
		case SortStars:
			less = result[i].StargazerCount < result[j].StargazerCount
		case SortLanguage:
			less = strings.ToLower(result[i].PrimaryLanguage) < strings.ToLower(result[j].PrimaryLanguage)
		default:
			less = strings.ToLower(result[i].Name) < strings.ToLower(result[j].Name)
		}

		if ascending {
			return less
		}
		return !less
	})

	return result
}

// searchRepos returns a filtered slice of repositories matching the search query.
// The search is case-insensitive and matches against the repository name.
func searchRepos(repos []github.Repository, query string) []github.Repository {
	if query == "" {
		return repos
	}
	query = strings.ToLower(query)
	var result []github.Repository
	for _, repo := range repos {
		if strings.Contains(strings.ToLower(repo.Name), query) {
			result = append(result, repo)
		}
	}
	return result
}

// getUniqueLanguages returns a sorted slice of unique languages from the repositories.
// Empty languages are excluded from the result.
func getUniqueLanguages(repos []github.Repository) []string {
	languageSet := make(map[string]struct{})

	for _, repo := range repos {
		if repo.PrimaryLanguage != "" {
			languageSet[repo.PrimaryLanguage] = struct{}{}
		}
	}

	languages := make([]string, 0, len(languageSet))
	for lang := range languageSet {
		languages = append(languages, lang)
	}

	sort.Strings(languages)
	return languages
}

// ApplyFilter sets the current filter and refreshes the filtered repos.
func (m *Model) ApplyFilter(filter Filter) {
	m.currentFilter = filter
	m.RefreshFilteredRepos()
}

// ApplyLanguageFilter sets the language filter and refreshes the filtered repos.
func (m *Model) ApplyLanguageFilter(lang string) {
	m.languageFilter = lang
	m.RefreshFilteredRepos()
}

// ApplySort sets the sort field and refreshes the filtered repos.
func (m *Model) ApplySort(field SortField) {
	m.sortField = field
	m.RefreshFilteredRepos()
}

// ToggleSortDirection toggles between ascending and descending sort order.
func (m *Model) ToggleSortDirection() {
	m.sortAscending = !m.sortAscending
	m.RefreshFilteredRepos()
}

// RefreshFilteredRepos applies the current filter, search, and sort to the repos.
func (m *Model) RefreshFilteredRepos() {
	// Build visibility options from model state
	opts := filterOpts{
		showPrivate:  m.showPrivate,
		showArchived: m.showArchived,
	}

	// Apply filter first (with visibility options)
	filtered := filterRepos(m.repos, m.currentFilter, m.languageFilter, opts)
	// Then apply search
	if m.searchQuery != "" {
		filtered = searchRepos(filtered, m.searchQuery)
	}
	// Then sort
	m.filteredRepos = sortRepos(filtered, m.sortField, m.sortAscending)

	// Reset cursor and viewport offset if out of bounds
	if m.cursor >= len(m.filteredRepos) {
		m.cursor = max(0, len(m.filteredRepos)-1)
	}
	// Reset viewport offset to ensure cursor is visible
	if m.viewportOffset > m.cursor {
		m.viewportOffset = m.cursor
	}
}

// ToggleShowPrivate toggles the visibility of private repositories.
func (m *Model) ToggleShowPrivate() {
	m.showPrivate = !m.showPrivate
	m.RefreshFilteredRepos()
}

// ToggleShowArchived toggles the visibility of archived repositories.
func (m *Model) ToggleShowArchived() {
	m.showArchived = !m.showArchived
	m.RefreshFilteredRepos()
}
