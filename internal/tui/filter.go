// SPDX-FileCopyrightText: 2026 Logan Lindquist Land
// SPDX-License-Identifier: FSL-1.1-MIT

package tui

import (
	"sort"
	"strings"

	"github.com/llbbl/repjan/internal/github"
)

// filterRepos returns a filtered slice of repositories based on the filter type and language.
// Archived repositories are always excluded regardless of filter type.
func filterRepos(repos []github.Repository, filter Filter, language string) []github.Repository {
	result := make([]github.Repository, 0, len(repos))

	for _, repo := range repos {
		// Always exclude already archived repos
		if repo.IsArchived {
			continue
		}

		// Apply filter type
		switch filter {
		case FilterAll:
			// Include all non-archived repos
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
		case FilterPrivate:
			if !repo.IsPrivate {
				continue
			}
		}

		// Apply language filter if specified
		if language != "" && repo.PrimaryLanguage != language {
			continue
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
			less = result[i].DaysSinceActivity < result[j].DaysSinceActivity
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

// RefreshFilteredRepos applies the current filter and sort to the repos.
func (m *Model) RefreshFilteredRepos() {
	filtered := filterRepos(m.repos, m.currentFilter, m.languageFilter)
	m.filteredRepos = sortRepos(filtered, m.sortField, m.sortAscending)

	// Reset cursor if it's out of bounds
	if m.cursor >= len(m.filteredRepos) {
		m.cursor = len(m.filteredRepos) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}
