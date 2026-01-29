// SPDX-FileCopyrightText: 2026 Logan Lindquist Land
// SPDX-License-Identifier: FSL-1.1-MIT

package github

import (
	"encoding/json"
	"testing"
	"time"
)

func TestRepository_FullName(t *testing.T) {
	tests := []struct {
		name     string
		owner    string
		repoName string
		expected string
	}{
		{"standard repo", "llbbl", "repjan", "llbbl/repjan"},
		{"empty owner", "", "repjan", "/repjan"},
		{"empty name", "llbbl", "", "llbbl/"},
		{"both empty", "", "", "/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := Repository{
				Owner: tt.owner,
				Name:  tt.repoName,
			}
			got := repo.FullName()
			if got != tt.expected {
				t.Errorf("FullName() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestRepository_CalculateDaysSinceActivity(t *testing.T) {
	tests := []struct {
		name         string
		pushedAt     time.Time
		expectedDays int
		tolerance    int // Allow for time-based variance
	}{
		{"today", time.Now(), 0, 1},
		{"yesterday", time.Now().AddDate(0, 0, -1), 1, 1},
		{"one week ago", time.Now().AddDate(0, 0, -7), 7, 1},
		{"one year ago", time.Now().AddDate(-1, 0, 0), 365, 2},
		{"two years ago", time.Now().AddDate(-2, 0, 0), 730, 2},
		{"zero time", time.Time{}, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := Repository{
				PushedAt: tt.pushedAt,
			}
			repo.CalculateDaysSinceActivity()

			diff := repo.DaysSinceActivity - tt.expectedDays
			if diff < 0 {
				diff = -diff
			}
			if diff > tt.tolerance {
				t.Errorf("CalculateDaysSinceActivity() = %d, want %d (±%d)",
					repo.DaysSinceActivity, tt.expectedDays, tt.tolerance)
			}
		})
	}
}

func TestRepository_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name      string
		json      string
		wantOwner string
		wantName  string
		wantLang  string
		wantStars int
		wantErr   bool
	}{
		{
			name:      "complete repo",
			json:      `{"owner":{"login":"llbbl"},"name":"repjan","primaryLanguage":{"name":"Go"},"stargazerCount":10}`,
			wantOwner: "llbbl",
			wantName:  "repjan",
			wantLang:  "Go",
			wantStars: 10,
		},
		{
			name:      "null language",
			json:      `{"owner":{"login":"llbbl"},"name":"test","primaryLanguage":null}`,
			wantOwner: "llbbl",
			wantName:  "test",
			wantLang:  "",
		},
		{
			name:      "missing owner",
			json:      `{"name":"test"}`,
			wantOwner: "",
			wantName:  "test",
		},
		{
			name:      "missing primaryLanguage field",
			json:      `{"owner":{"login":"user"},"name":"myrepo"}`,
			wantOwner: "user",
			wantName:  "myrepo",
			wantLang:  "",
		},
		{
			name:      "all fields populated",
			json:      `{"owner":{"login":"org"},"name":"project","description":"A project","stargazerCount":100,"forkCount":20,"isArchived":true,"isFork":false,"isPrivate":true,"primaryLanguage":{"name":"Rust"}}`,
			wantOwner: "org",
			wantName:  "project",
			wantLang:  "Rust",
			wantStars: 100,
		},
		{
			name:    "invalid json",
			json:    `{"owner":{"login":"llbbl"`,
			wantErr: true,
		},
		{
			name:      "empty json object",
			json:      `{}`,
			wantOwner: "",
			wantName:  "",
			wantLang:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var repo Repository
			err := json.Unmarshal([]byte(tt.json), &repo)

			if tt.wantErr {
				if err == nil {
					t.Error("UnmarshalJSON() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("UnmarshalJSON() unexpected error: %v", err)
			}

			if repo.Owner != tt.wantOwner {
				t.Errorf("Owner = %q, want %q", repo.Owner, tt.wantOwner)
			}
			if repo.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", repo.Name, tt.wantName)
			}
			if repo.PrimaryLanguage != tt.wantLang {
				t.Errorf("PrimaryLanguage = %q, want %q", repo.PrimaryLanguage, tt.wantLang)
			}
			if tt.wantStars != 0 && repo.StargazerCount != tt.wantStars {
				t.Errorf("StargazerCount = %d, want %d", repo.StargazerCount, tt.wantStars)
			}
		})
	}
}

func TestRepository_UnmarshalJSON_BooleanFields(t *testing.T) {
	jsonStr := `{
		"owner": {"login": "testuser"},
		"name": "testrepo",
		"isArchived": true,
		"isFork": true,
		"isPrivate": true
	}`

	var repo Repository
	err := json.Unmarshal([]byte(jsonStr), &repo)
	if err != nil {
		t.Fatalf("UnmarshalJSON() unexpected error: %v", err)
	}

	if !repo.IsArchived {
		t.Error("IsArchived = false, want true")
	}
	if !repo.IsFork {
		t.Error("IsFork = false, want true")
	}
	if !repo.IsPrivate {
		t.Error("IsPrivate = false, want true")
	}
}

func TestRepository_UnmarshalJSON_TimeFields(t *testing.T) {
	// Use RFC3339 format as that's what GitHub API returns
	pushedAt := "2024-06-15T10:30:00Z"
	createdAt := "2023-01-01T00:00:00Z"

	jsonStr := `{
		"owner": {"login": "testuser"},
		"name": "testrepo",
		"pushedAt": "` + pushedAt + `",
		"createdAt": "` + createdAt + `"
	}`

	var repo Repository
	err := json.Unmarshal([]byte(jsonStr), &repo)
	if err != nil {
		t.Fatalf("UnmarshalJSON() unexpected error: %v", err)
	}

	expectedPushed, _ := time.Parse(time.RFC3339, pushedAt)
	expectedCreated, _ := time.Parse(time.RFC3339, createdAt)

	if !repo.PushedAt.Equal(expectedPushed) {
		t.Errorf("PushedAt = %v, want %v", repo.PushedAt, expectedPushed)
	}
	if !repo.CreatedAt.Equal(expectedCreated) {
		t.Errorf("CreatedAt = %v, want %v", repo.CreatedAt, expectedCreated)
	}
}
