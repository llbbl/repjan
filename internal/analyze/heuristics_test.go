// SPDX-FileCopyrightText: 2026 Logan Lindquist Land
// SPDX-License-Identifier: FSL-1.1-MIT

package analyze

import (
	"testing"

	"github.com/llbbl/repjan/internal/testutil"
)

func TestIsArchiveCandidate(t *testing.T) {
	t.Run("age-based criteria", func(t *testing.T) {
		tests := []struct {
			name          string
			daysInactive  int
			wantCandidate bool
			wantReason    string
		}{
			{
				name:          "0 days inactive is not a candidate",
				daysInactive:  0,
				wantCandidate: false,
				wantReason:    "",
			},
			{
				name:          "364 days inactive is not a candidate",
				daysInactive:  364,
				wantCandidate: false,
				wantReason:    "",
			},
			{
				name:          "365 days inactive is not a candidate (boundary)",
				daysInactive:  365,
				wantCandidate: false,
				wantReason:    "",
			},
			{
				name:          "366 days inactive is a candidate",
				daysInactive:  366,
				wantCandidate: true,
				wantReason:    "No activity in 1+ year",
			},
			{
				name:          "730 days inactive is still 1+ year (boundary)",
				daysInactive:  730,
				wantCandidate: true,
				wantReason:    "No activity in 1+ year",
			},
			{
				name:          "731 days inactive is 2+ years",
				daysInactive:  731,
				wantCandidate: true,
				wantReason:    "No activity in 2+ years",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// Use stars/forks > 0 to avoid triggering engagement criteria
				repo := testutil.NewTestRepo(
					testutil.WithDaysInactive(tt.daysInactive),
					testutil.WithStars(10),
					testutil.WithForks(5),
				)

				gotCandidate, gotReason := IsArchiveCandidate(repo)

				if gotCandidate != tt.wantCandidate {
					t.Errorf("IsArchiveCandidate() candidate = %v, want %v", gotCandidate, tt.wantCandidate)
				}
				if gotReason != tt.wantReason {
					t.Errorf("IsArchiveCandidate() reason = %q, want %q", gotReason, tt.wantReason)
				}
			})
		}
	})

	t.Run("engagement-based criteria", func(t *testing.T) {
		tests := []struct {
			name          string
			stars         int
			forks         int
			wantCandidate bool
			wantReason    string
		}{
			{
				name:          "0 stars and 0 forks is a candidate",
				stars:         0,
				forks:         0,
				wantCandidate: true,
				wantReason:    "No community engagement",
			},
			{
				name:          "1 star and 0 forks is not a candidate",
				stars:         1,
				forks:         0,
				wantCandidate: false,
				wantReason:    "",
			},
			{
				name:          "0 stars and 1 fork is not a candidate",
				stars:         0,
				forks:         1,
				wantCandidate: false,
				wantReason:    "",
			},
			{
				name:          "1 star and 1 fork is not a candidate",
				stars:         1,
				forks:         1,
				wantCandidate: false,
				wantReason:    "",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// Use recent activity to avoid triggering age criteria
				repo := testutil.NewTestRepo(
					testutil.WithDaysInactive(0),
					testutil.WithStars(tt.stars),
					testutil.WithForks(tt.forks),
				)

				gotCandidate, gotReason := IsArchiveCandidate(repo)

				if gotCandidate != tt.wantCandidate {
					t.Errorf("IsArchiveCandidate() candidate = %v, want %v", gotCandidate, tt.wantCandidate)
				}
				if gotReason != tt.wantReason {
					t.Errorf("IsArchiveCandidate() reason = %q, want %q", gotReason, tt.wantReason)
				}
			})
		}
	})

	t.Run("fork-based criteria", func(t *testing.T) {
		tests := []struct {
			name          string
			isFork        bool
			daysInactive  int
			wantCandidate bool
			wantReason    string
		}{
			{
				name:          "fork with 179 days inactive is not a candidate",
				isFork:        true,
				daysInactive:  179,
				wantCandidate: false,
				wantReason:    "",
			},
			{
				name:          "fork with 180 days inactive is not a candidate (boundary)",
				isFork:        true,
				daysInactive:  180,
				wantCandidate: false,
				wantReason:    "",
			},
			{
				name:          "fork with 181 days inactive is a candidate",
				isFork:        true,
				daysInactive:  181,
				wantCandidate: true,
				wantReason:    "Stale fork",
			},
			{
				name:          "non-fork with 181 days inactive is not a candidate",
				isFork:        false,
				daysInactive:  181,
				wantCandidate: false,
				wantReason:    "",
			},
			{
				name:          "fork with 366 days triggers multiple reasons",
				isFork:        true,
				daysInactive:  366,
				wantCandidate: true,
				wantReason:    "No activity in 1+ year; Stale fork",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// Use stars/forks > 0 to avoid triggering engagement criteria
				repo := testutil.NewTestRepo(
					testutil.WithDaysInactive(tt.daysInactive),
					testutil.WithFork(tt.isFork),
					testutil.WithStars(10),
					testutil.WithForks(5),
				)

				gotCandidate, gotReason := IsArchiveCandidate(repo)

				if gotCandidate != tt.wantCandidate {
					t.Errorf("IsArchiveCandidate() candidate = %v, want %v", gotCandidate, tt.wantCandidate)
				}
				if gotReason != tt.wantReason {
					t.Errorf("IsArchiveCandidate() reason = %q, want %q", gotReason, tt.wantReason)
				}
			})
		}
	})

	t.Run("language-based criteria", func(t *testing.T) {
		tests := []struct {
			name          string
			language      string
			daysInactive  int
			wantCandidate bool
			wantReason    string
		}{
			{
				name:          "PHP with 366 days inactive triggers legacy + age",
				language:      "PHP",
				daysInactive:  366,
				wantCandidate: true,
				wantReason:    "No activity in 1+ year; Legacy language, inactive",
			},
			{
				name:          "Go with 366 days inactive only triggers age",
				language:      "Go",
				daysInactive:  366,
				wantCandidate: true,
				wantReason:    "No activity in 1+ year",
			},
			{
				name:          "PHP with 365 days inactive is not a candidate",
				language:      "PHP",
				daysInactive:  365,
				wantCandidate: false,
				wantReason:    "",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// Use stars/forks > 0 to avoid triggering engagement criteria
				repo := testutil.NewTestRepo(
					testutil.WithDaysInactive(tt.daysInactive),
					testutil.WithLanguage(tt.language),
					testutil.WithStars(10),
					testutil.WithForks(5),
				)

				gotCandidate, gotReason := IsArchiveCandidate(repo)

				if gotCandidate != tt.wantCandidate {
					t.Errorf("IsArchiveCandidate() candidate = %v, want %v", gotCandidate, tt.wantCandidate)
				}
				if gotReason != tt.wantReason {
					t.Errorf("IsArchiveCandidate() reason = %q, want %q", gotReason, tt.wantReason)
				}
			})
		}
	})

	t.Run("combination criteria", func(t *testing.T) {
		tests := []struct {
			name          string
			opts          []testutil.RepoOption
			wantCandidate bool
			wantReason    string
		}{
			{
				name: "all criteria met triggers 4 reasons",
				opts: []testutil.RepoOption{
					testutil.WithDaysInactive(731), // 2+ years
					testutil.WithStars(0),
					testutil.WithForks(0),
					testutil.WithFork(true),
					testutil.WithLanguage("PHP"),
				},
				wantCandidate: true,
				wantReason:    "No activity in 2+ years; No community engagement; Stale fork; Legacy language, inactive",
			},
			{
				name: "active repo with stars is not a candidate",
				opts: []testutil.RepoOption{
					testutil.WithDaysInactive(0),
					testutil.WithStars(100),
					testutil.WithForks(50),
					testutil.WithFork(false),
					testutil.WithLanguage("Go"),
				},
				wantCandidate: false,
				wantReason:    "",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				repo := testutil.NewTestRepo(tt.opts...)

				gotCandidate, gotReason := IsArchiveCandidate(repo)

				if gotCandidate != tt.wantCandidate {
					t.Errorf("IsArchiveCandidate() candidate = %v, want %v", gotCandidate, tt.wantCandidate)
				}
				if gotReason != tt.wantReason {
					t.Errorf("IsArchiveCandidate() reason = %q, want %q", gotReason, tt.wantReason)
				}
			})
		}
	})
}

func TestIsLegacyLanguage(t *testing.T) {
	t.Run("legacy languages return true", func(t *testing.T) {
		legacyLanguages := []string{
			"PHP",
			"CoffeeScript",
			"Perl",
			"ActionScript",
			"Objective-C",
		}

		for _, lang := range legacyLanguages {
			t.Run(lang, func(t *testing.T) {
				if !IsLegacyLanguage(lang) {
					t.Errorf("IsLegacyLanguage(%q) = false, want true", lang)
				}
			})
		}
	})

	t.Run("modern languages return false", func(t *testing.T) {
		modernLanguages := []string{
			"Go",
			"JavaScript",
			"Python",
			"TypeScript",
			"Rust",
		}

		for _, lang := range modernLanguages {
			t.Run(lang, func(t *testing.T) {
				if IsLegacyLanguage(lang) {
					t.Errorf("IsLegacyLanguage(%q) = true, want false", lang)
				}
			})
		}
	})

	t.Run("empty string returns false", func(t *testing.T) {
		if IsLegacyLanguage("") {
			t.Error("IsLegacyLanguage(\"\") = true, want false")
		}
	})

	t.Run("case insensitivity", func(t *testing.T) {
		cases := []string{"php", "PHP", "Php", "pHp", "phP"}

		for _, c := range cases {
			t.Run(c, func(t *testing.T) {
				if !IsLegacyLanguage(c) {
					t.Errorf("IsLegacyLanguage(%q) = false, want true", c)
				}
			})
		}
	})
}
