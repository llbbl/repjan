// SPDX-FileCopyrightText: 2026 Logan Lindquist Land
// SPDX-License-Identifier: FSL-1.1-MIT

package github

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"testing"
)

// mockResponse represents a mocked command response.
type mockResponse struct {
	output []byte
	err    error
}

// MockExecutor implements CommandExecutor for testing.
type MockExecutor struct {
	// Map command string to response
	responses map[string]mockResponse
}

// NewMockExecutor creates a new MockExecutor with empty responses.
func NewMockExecutor() *MockExecutor {
	return &MockExecutor{
		responses: make(map[string]mockResponse),
	}
}

// AddResponse registers a mock response for a command.
func (m *MockExecutor) AddResponse(name string, args []string, output []byte, err error) {
	key := m.buildKey(name, args)
	m.responses[key] = mockResponse{output: output, err: err}
}

// Execute returns the mocked response for the given command.
func (m *MockExecutor) Execute(name string, args ...string) ([]byte, error) {
	key := m.buildKey(name, args)
	if resp, ok := m.responses[key]; ok {
		return resp.output, resp.err
	}
	return nil, fmt.Errorf("unexpected command: %s", key)
}

// buildKey constructs a lookup key from command name and args.
func (m *MockExecutor) buildKey(name string, args []string) string {
	return name + " " + strings.Join(args, " ")
}

func TestClient_FetchRepositories(t *testing.T) {
	tests := []struct {
		name     string
		owner    string
		mockResp string
		mockErr  error
		wantLen  int
		wantErr  bool
		errCheck func(error) bool // Optional error type check
	}{
		{
			name:     "valid response with single repo",
			owner:    "testowner",
			mockResp: `[{"owner":{"login":"testowner"},"name":"repo1","stargazerCount":5}]`,
			wantLen:  1,
		},
		{
			name:     "valid response with multiple repos",
			owner:    "testowner",
			mockResp: `[{"owner":{"login":"testowner"},"name":"repo1","stargazerCount":5},{"owner":{"login":"testowner"},"name":"repo2","stargazerCount":10}]`,
			wantLen:  2,
		},
		{
			name:     "empty response",
			owner:    "testowner",
			mockResp: `[]`,
			wantLen:  0,
		},
		{
			name:     "empty string response",
			owner:    "testowner",
			mockResp: ``,
			wantLen:  0,
		},
		{
			name:     "whitespace only response",
			owner:    "testowner",
			mockResp: `   `,
			wantLen:  0,
		},
		{
			name:     "malformed JSON",
			owner:    "testowner",
			mockResp: `{invalid`,
			wantErr:  true,
		},
		{
			name:    "empty owner",
			owner:   "",
			wantErr: true,
		},
		{
			name:    "executor error",
			owner:   "testowner",
			mockErr: errors.New("command failed"),
			wantErr: true,
		},
		{
			name:    "not authenticated error",
			owner:   "testowner",
			mockErr: errors.New("gh auth login required"),
			wantErr: true,
			errCheck: func(err error) bool {
				return errors.Is(err, ErrNotAuthenticated)
			},
		},
		{
			name:    "rate limit error",
			owner:   "testowner",
			mockErr: errors.New("API rate limit exceeded"),
			wantErr: true,
			errCheck: func(err error) bool {
				return errors.Is(err, ErrRateLimit)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockExecutor()
			if tt.owner != "" {
				mock.AddResponse("gh", []string{
					"repo", "list", tt.owner,
					"--json", "name,description,pushedAt,createdAt,stargazerCount,forkCount,isArchived,isFork,isPrivate,primaryLanguage,owner",
					"--limit", "1000",
				}, []byte(tt.mockResp), tt.mockErr)
			}

			client := NewClient(mock)
			repos, err := client.FetchRepositories(tt.owner)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				if tt.errCheck != nil && !tt.errCheck(err) {
					t.Errorf("error check failed for error: %v", err)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(repos) != tt.wantLen {
				t.Errorf("got %d repos, want %d", len(repos), tt.wantLen)
			}
		})
	}
}

func TestClient_FetchRepositories_PopulatesFields(t *testing.T) {
	mock := NewMockExecutor()
	jsonResp := `[{
		"owner": {"login": "testowner"},
		"name": "myrepo",
		"description": "A test repository",
		"stargazerCount": 42,
		"forkCount": 7,
		"isArchived": false,
		"isFork": true,
		"isPrivate": false,
		"primaryLanguage": {"name": "Go"}
	}]`
	mock.AddResponse("gh", []string{
		"repo", "list", "testowner",
		"--json", "name,description,pushedAt,createdAt,stargazerCount,forkCount,isArchived,isFork,isPrivate,primaryLanguage,owner",
		"--limit", "1000",
	}, []byte(jsonResp), nil)

	client := NewClient(mock)
	repos, err := client.FetchRepositories("testowner")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(repos) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(repos))
	}

	repo := repos[0]
	if repo.Owner != "testowner" {
		t.Errorf("Owner = %q, want %q", repo.Owner, "testowner")
	}
	if repo.Name != "myrepo" {
		t.Errorf("Name = %q, want %q", repo.Name, "myrepo")
	}
	if repo.Description != "A test repository" {
		t.Errorf("Description = %q, want %q", repo.Description, "A test repository")
	}
	if repo.StargazerCount != 42 {
		t.Errorf("StargazerCount = %d, want %d", repo.StargazerCount, 42)
	}
	if repo.ForkCount != 7 {
		t.Errorf("ForkCount = %d, want %d", repo.ForkCount, 7)
	}
	if repo.IsArchived != false {
		t.Errorf("IsArchived = %v, want %v", repo.IsArchived, false)
	}
	if repo.IsFork != true {
		t.Errorf("IsFork = %v, want %v", repo.IsFork, true)
	}
	if repo.IsPrivate != false {
		t.Errorf("IsPrivate = %v, want %v", repo.IsPrivate, false)
	}
	if repo.PrimaryLanguage != "Go" {
		t.Errorf("PrimaryLanguage = %q, want %q", repo.PrimaryLanguage, "Go")
	}
}

func TestClient_GetAuthenticatedUser(t *testing.T) {
	tests := []struct {
		name     string
		mockResp string
		mockErr  error
		want     string
		wantErr  bool
		errCheck func(error) bool
	}{
		{
			name:     "valid username",
			mockResp: "testuser\n",
			want:     "testuser",
		},
		{
			name:     "username with whitespace",
			mockResp: "  testuser  \n",
			want:     "testuser",
		},
		{
			name:     "empty response returns ErrNotAuthenticated",
			mockResp: "",
			wantErr:  true,
			errCheck: func(err error) bool {
				return errors.Is(err, ErrNotAuthenticated)
			},
		},
		{
			name:     "whitespace only response returns ErrNotAuthenticated",
			mockResp: "   \n",
			wantErr:  true,
			errCheck: func(err error) bool {
				return errors.Is(err, ErrNotAuthenticated)
			},
		},
		{
			name:    "executor error",
			mockErr: errors.New("command failed"),
			wantErr: true,
		},
		{
			name:    "not logged in error",
			mockErr: errors.New("not logged in"),
			wantErr: true,
			errCheck: func(err error) bool {
				return errors.Is(err, ErrNotAuthenticated)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockExecutor()
			mock.AddResponse("gh", []string{"api", "user", "--jq", ".login"}, []byte(tt.mockResp), tt.mockErr)

			client := NewClient(mock)
			got, err := client.GetAuthenticatedUser()

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				if tt.errCheck != nil && !tt.errCheck(err) {
					t.Errorf("error check failed for error: %v", err)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestClient_ArchiveRepository(t *testing.T) {
	tests := []struct {
		name    string
		owner   string
		repo    string
		mockErr error
		wantErr bool
		errMsg  string
	}{
		{
			name:  "success",
			owner: "testowner",
			repo:  "testrepo",
		},
		{
			name:    "empty owner",
			owner:   "",
			repo:    "testrepo",
			wantErr: true,
		},
		{
			name:    "empty repo name",
			owner:   "testowner",
			repo:    "",
			wantErr: true,
		},
		{
			name:    "executor error",
			owner:   "testowner",
			repo:    "testrepo",
			mockErr: errors.New("archive failed"),
			wantErr: true,
		},
		{
			name:    "not found error",
			owner:   "testowner",
			repo:    "nonexistent",
			mockErr: errors.New("repository not found"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockExecutor()
			if tt.owner != "" && tt.repo != "" {
				fullName := tt.owner + "/" + tt.repo
				mock.AddResponse("gh", []string{"repo", "archive", fullName, "--yes"}, nil, tt.mockErr)
			}

			client := NewClient(mock)
			err := client.ArchiveRepository(tt.owner, tt.repo)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestClient_ArchiveRepository_ErrorWrapping(t *testing.T) {
	mock := NewMockExecutor()
	mock.AddResponse("gh", []string{"repo", "archive", "owner/repo", "--yes"}, []byte("error output"), errors.New("command failed"))

	client := NewClient(mock)
	err := client.ArchiveRepository("owner", "repo")

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Check that error message contains context about archiving
	if !strings.Contains(err.Error(), "archiving") {
		t.Errorf("error should contain 'archiving', got: %v", err)
	}

	// Check that error message contains the repo name
	if !strings.Contains(err.Error(), "owner/repo") {
		t.Errorf("error should contain 'owner/repo', got: %v", err)
	}
}

func TestClient_FetchReadme(t *testing.T) {
	readmeContent := "# Test README\n\nThis is a test."
	encodedContent := base64.StdEncoding.EncodeToString([]byte(readmeContent))

	tests := []struct {
		name     string
		owner    string
		repo     string
		mockResp string
		mockErr  error
		want     string
		wantErr  bool
	}{
		{
			name:     "valid readme",
			owner:    "testowner",
			repo:     "testrepo",
			mockResp: encodedContent,
			want:     readmeContent,
		},
		{
			name:     "readme with newlines in base64",
			owner:    "testowner",
			repo:     "testrepo",
			mockResp: encodedContent[:20] + "\n" + encodedContent[20:],
			want:     readmeContent,
		},
		{
			name:     "empty readme returns empty string",
			owner:    "testowner",
			repo:     "testrepo",
			mockResp: "",
			want:     "",
		},
		{
			name:    "404 not found returns empty string not error",
			owner:   "testowner",
			repo:    "testrepo",
			mockErr: errors.New("HTTP 404: Not Found"),
			want:    "",
		},
		{
			name:    "could not resolve returns empty string",
			owner:   "testowner",
			repo:    "testrepo",
			mockErr: errors.New("could not resolve"),
			want:    "",
		},
		{
			name:    "empty owner",
			owner:   "",
			repo:    "testrepo",
			wantErr: true,
		},
		{
			name:    "empty repo",
			owner:   "testowner",
			repo:    "",
			wantErr: true,
		},
		{
			name:    "non-404 error returns error",
			owner:   "testowner",
			repo:    "testrepo",
			mockErr: errors.New("network timeout"),
			wantErr: true,
		},
		{
			name:     "invalid base64 returns error",
			owner:    "testowner",
			repo:     "testrepo",
			mockResp: "not-valid-base64!!!",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockExecutor()
			if tt.owner != "" && tt.repo != "" {
				endpoint := fmt.Sprintf("repos/%s/%s/readme", tt.owner, tt.repo)
				mock.AddResponse("gh", []string{"api", endpoint, "--jq", ".content"}, []byte(tt.mockResp), tt.mockErr)
			}

			client := NewClient(mock)
			got, err := client.FetchReadme(tt.owner, tt.repo)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewClient(t *testing.T) {
	mock := NewMockExecutor()
	client := NewClient(mock)

	if client == nil {
		t.Fatal("NewClient returned nil")
	}

	if client.executor != mock {
		t.Error("executor not set correctly")
	}
}

func TestNewDefaultClient(t *testing.T) {
	client := NewDefaultClient()

	if client == nil {
		t.Fatal("NewDefaultClient returned nil")
	}

	// Verify it has a RealExecutor
	if _, ok := client.executor.(*RealExecutor); !ok {
		t.Error("NewDefaultClient should use RealExecutor")
	}
}
