// SPDX-FileCopyrightText: 2026 Logan Lindquist Land
// SPDX-License-Identifier: FSL-1.1-MIT

package analyze

import (
	"strings"
	"testing"

	"github.com/llbbl/repjan/internal/github"
	"github.com/llbbl/repjan/internal/testutil"
)

func TestCheckFabricAvailable(t *testing.T) {
	tests := []struct {
		name       string
		fabricPath string
		want       bool
	}{
		{
			name:       "empty path defaults to fabric lookup",
			fabricPath: "",
			// fabric is likely not installed in test environment
			want: false,
		},
		{
			name:       "nonexistent binary returns false",
			fabricPath: "/nonexistent/path/fabric",
			want:       false,
		},
		{
			name:       "known binary returns true",
			fabricPath: "go", // go should be available in test environment
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CheckFabricAvailable(tt.fabricPath)
			if got != tt.want {
				t.Errorf("CheckFabricAvailable(%q) = %v, want %v", tt.fabricPath, got, tt.want)
			}
		})
	}
}

func TestTruncateReadme(t *testing.T) {
	tests := []struct {
		name   string
		readme string
		maxLen int
		want   string
	}{
		{
			name:   "short readme unchanged",
			readme: "Hello",
			maxLen: 100,
			want:   "Hello",
		},
		{
			name:   "exact length unchanged",
			readme: "Hello",
			maxLen: 5,
			want:   "Hello",
		},
		{
			name:   "truncated with suffix",
			readme: "Hello World",
			maxLen: 5,
			want:   "Hello\n... [truncated]",
		},
		{
			name:   "empty readme unchanged",
			readme: "",
			maxLen: 100,
			want:   "",
		},
		{
			name:   "unicode content truncated correctly",
			readme: "Hello World",
			maxLen: 7,
			want:   "Hello W\n... [truncated]",
		},
		{
			name:   "maxLen zero truncates everything",
			readme: "Hello",
			maxLen: 0,
			want:   "\n... [truncated]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateReadme(tt.readme, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateReadme(%q, %d) = %q, want %q", tt.readme, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestAnalyzeRepo_ErrorHandling(t *testing.T) {
	t.Run("nonexistent fabric path returns error", func(t *testing.T) {
		repo := testutil.NewTestRepo(
			testutil.WithOwner("testowner"),
			testutil.WithName("testrepo"),
			testutil.WithDescription("Test description"),
			testutil.WithLanguage("Go"),
			testutil.WithStars(10),
			testutil.WithForks(5),
			testutil.WithDaysInactive(100),
		)

		_, err := AnalyzeRepo(repo, "README content", "/nonexistent/fabric")
		if err == nil {
			t.Error("AnalyzeRepo() expected error, got nil")
		}
		if !strings.Contains(err.Error(), "fabric analysis failed") {
			t.Errorf("AnalyzeRepo() error = %q, want error containing %q", err.Error(), "fabric analysis failed")
		}
	})

	t.Run("empty readme is handled", func(t *testing.T) {
		repo := testutil.NewTestRepo(
			testutil.WithOwner("testowner"),
			testutil.WithName("testrepo"),
		)

		_, err := AnalyzeRepo(repo, "", "/nonexistent/fabric")
		if err == nil {
			t.Error("AnalyzeRepo() expected error, got nil")
		}
		// Error should still be about fabric failing, not about empty readme
		if !strings.Contains(err.Error(), "fabric analysis failed") {
			t.Errorf("AnalyzeRepo() error = %q, want error containing %q", err.Error(), "fabric analysis failed")
		}
	})
}

func TestAnalyzeRepos_EmptyInput(t *testing.T) {
	tests := []struct {
		name       string
		repos      []github.Repository
		fabricPath string
		wantErr    string
	}{
		{
			name:       "empty slice returns error",
			repos:      []github.Repository{},
			fabricPath: "fabric",
			wantErr:    "no repositories",
		},
		{
			name:       "nil slice returns error",
			repos:      nil,
			fabricPath: "fabric",
			wantErr:    "no repositories",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := AnalyzeRepos(tt.repos, tt.fabricPath)
			if err == nil {
				t.Error("AnalyzeRepos() expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("AnalyzeRepos() error = %q, want error containing %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestAnalyzeRepos_ErrorHandling(t *testing.T) {
	t.Run("nonexistent fabric path returns error", func(t *testing.T) {
		repos := []github.Repository{
			testutil.NewTestRepo(
				testutil.WithOwner("owner1"),
				testutil.WithName("repo1"),
			),
			testutil.NewTestRepo(
				testutil.WithOwner("owner2"),
				testutil.WithName("repo2"),
			),
		}

		_, err := AnalyzeRepos(repos, "/nonexistent/fabric")
		if err == nil {
			t.Error("AnalyzeRepos() expected error, got nil")
		}
		if !strings.Contains(err.Error(), "fabric batch analysis failed") {
			t.Errorf("AnalyzeRepos() error = %q, want error containing %q", err.Error(), "fabric batch analysis failed")
		}
	})
}
