// SPDX-FileCopyrightText: 2026 Logan Lindquist Land
// SPDX-License-Identifier: FSL-1.1-MIT

// Package analyze provides repository analysis capabilities.
package analyze

import (
	"strings"

	"github.com/llbbl/repjan/internal/github"
)

// legacyLanguages contains languages considered legacy/outdated.
var legacyLanguages = map[string]bool{
	"php":          true,
	"coffeescript": true,
	"perl":         true,
	"actionscript": true,
	"objective-c":  true,
}

// IsLegacyLanguage returns true if the language is considered legacy.
func IsLegacyLanguage(lang string) bool {
	return legacyLanguages[strings.ToLower(lang)]
}

// IsArchiveCandidate determines if a repository is a candidate for archiving.
// Returns (true, reasons) if the repo is a candidate, (false, "") otherwise.
// Reasons are returned as a semicolon-separated string when multiple criteria match.
func IsArchiveCandidate(repo github.Repository) (bool, string) {
	var reasons []string

	// Age-based criteria (check higher threshold first)
	if repo.DaysSinceActivity > 730 {
		reasons = append(reasons, "No activity in 2+ years")
	} else if repo.DaysSinceActivity > 365 {
		reasons = append(reasons, "No activity in 1+ year")
	}

	// Engagement-based criteria
	if repo.StargazerCount == 0 && repo.ForkCount == 0 {
		reasons = append(reasons, "No community engagement")
	}

	// Fork-based criteria
	if repo.IsFork && repo.DaysSinceActivity > 180 {
		reasons = append(reasons, "Stale fork")
	}

	// Language-based criteria
	if IsLegacyLanguage(repo.PrimaryLanguage) && repo.DaysSinceActivity > 365 {
		reasons = append(reasons, "Legacy language, inactive")
	}

	if len(reasons) == 0 {
		return false, ""
	}

	return true, strings.Join(reasons, "; ")
}
