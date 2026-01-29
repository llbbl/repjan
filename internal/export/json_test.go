// SPDX-FileCopyrightText: 2026 Logan Lindquist Land
// SPDX-License-Identifier: FSL-1.1-MIT

package export

import (
	"encoding/json"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/llbbl/repjan/internal/github"
	"github.com/llbbl/repjan/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExport(t *testing.T) {
	tests := []struct {
		name      string
		repos     []github.Repository
		owner     string
		wantErr   bool
		checkFunc func(t *testing.T, data ExportData)
	}{
		{
			name: "single repo",
			repos: []github.Repository{
				testutil.NewTestRepo(testutil.WithName("test-repo")),
			},
			owner: "testowner",
			checkFunc: func(t *testing.T, data ExportData) {
				assert.Equal(t, 1, data.TotalMarked)
				assert.Equal(t, "testowner", data.Owner)
				assert.Len(t, data.Repositories, 1)
			},
		},
		{
			name: "multiple repos",
			repos: []github.Repository{
				testutil.NewTestRepo(testutil.WithName("repo1")),
				testutil.NewTestRepo(testutil.WithName("repo2")),
			},
			owner: "testowner",
			checkFunc: func(t *testing.T, data ExportData) {
				assert.Equal(t, 2, data.TotalMarked)
			},
		},
		{
			name:  "empty repo list",
			repos: []github.Repository{},
			owner: "testowner",
			checkFunc: func(t *testing.T, data ExportData) {
				assert.Equal(t, 0, data.TotalMarked)
				assert.Empty(t, data.Repositories)
			},
		},
		{
			name: "preserves owner from parameter",
			repos: []github.Repository{
				testutil.NewTestRepo(testutil.WithOwner("different-owner")),
			},
			owner: "parameter-owner",
			checkFunc: func(t *testing.T, data ExportData) {
				assert.Equal(t, "parameter-owner", data.Owner)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Change to temp directory
			tmpDir := t.TempDir()
			oldWd, _ := os.Getwd()
			err := os.Chdir(tmpDir)
			require.NoError(t, err)
			defer func() { _ = os.Chdir(oldWd) }()

			filename, err := Export(tt.repos, tt.owner)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)

			// Read and parse the file
			content, err := os.ReadFile(filename)
			require.NoError(t, err)

			var data ExportData
			err = json.Unmarshal(content, &data)
			require.NoError(t, err)

			tt.checkFunc(t, data)
		})
	}
}

func TestExport_ExportedAtTimestamp(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	err := os.Chdir(tmpDir)
	require.NoError(t, err)
	defer func() { _ = os.Chdir(oldWd) }()

	beforeExport := time.Now().Add(-time.Second)

	filename, err := Export([]github.Repository{
		testutil.NewTestRepo(),
	}, "testowner")
	require.NoError(t, err)

	afterExport := time.Now().Add(time.Second)

	content, err := os.ReadFile(filename)
	require.NoError(t, err)

	var data ExportData
	err = json.Unmarshal(content, &data)
	require.NoError(t, err)

	assert.False(t, data.ExportedAt.IsZero(), "exported_at should not be zero")
	assert.True(t, data.ExportedAt.After(beforeExport), "exported_at should be after test start")
	assert.True(t, data.ExportedAt.Before(afterExport), "exported_at should be before test end")
}

func TestExport_FieldsPopulatedCorrectly(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	err := os.Chdir(tmpDir)
	require.NoError(t, err)
	defer func() { _ = os.Chdir(oldWd) }()

	pushedAt := time.Now().AddDate(0, 0, -400) // 400 days ago triggers heuristic

	repo := testutil.NewTestRepo(
		testutil.WithOwner("myowner"),
		testutil.WithName("myrepo"),
		testutil.WithStars(42),
		testutil.WithForks(7),
		testutil.WithLanguage("Python"),
		testutil.WithFork(true),
		testutil.WithPrivate(true),
		testutil.WithPushedAt(pushedAt),
	)

	filename, err := Export([]github.Repository{repo}, "myowner")
	require.NoError(t, err)

	content, err := os.ReadFile(filename)
	require.NoError(t, err)

	var data ExportData
	err = json.Unmarshal(content, &data)
	require.NoError(t, err)

	require.Len(t, data.Repositories, 1)
	exported := data.Repositories[0]

	assert.Equal(t, "myrepo", exported.Name)
	assert.Equal(t, "myowner/myrepo", exported.FullName)
	assert.Equal(t, 42, exported.Stars)
	assert.Equal(t, 7, exported.Forks)
	assert.Equal(t, "Python", exported.Language)
	assert.True(t, exported.IsFork)
	assert.True(t, exported.IsPrivate)
	assert.Equal(t, repo.DaysSinceActivity, exported.DaysSinceActivity)
	assert.WithinDuration(t, pushedAt, exported.LastPush, time.Second)
}

func TestExport_ReasonFromHeuristics(t *testing.T) {
	tests := []struct {
		name           string
		repo           github.Repository
		expectReason   string
		expectContains []string
	}{
		{
			name: "inactive 2+ years",
			repo: testutil.NewTestRepo(
				testutil.WithDaysInactive(800),
				testutil.WithStars(10),
				testutil.WithForks(5),
			),
			expectContains: []string{"No activity in 2+ years"},
		},
		{
			name: "inactive 1+ year",
			repo: testutil.NewTestRepo(
				testutil.WithDaysInactive(400),
				testutil.WithStars(10),
				testutil.WithForks(5),
			),
			expectContains: []string{"No activity in 1+ year"},
		},
		{
			name: "no engagement",
			repo: testutil.NewTestRepo(
				testutil.WithStars(0),
				testutil.WithForks(0),
				testutil.WithDaysInactive(30),
			),
			expectContains: []string{"No community engagement"},
		},
		{
			name: "stale fork",
			repo: testutil.NewTestRepo(
				testutil.WithFork(true),
				testutil.WithDaysInactive(200),
				testutil.WithStars(10),
				testutil.WithForks(5),
			),
			expectContains: []string{"Stale fork"},
		},
		{
			name: "multiple reasons",
			repo: testutil.NewTestRepo(
				testutil.WithDaysInactive(800),
				testutil.WithStars(0),
				testutil.WithForks(0),
			),
			expectContains: []string{"No activity in 2+ years", "No community engagement"},
		},
		{
			name: "active repo no reason",
			repo: testutil.NewTestRepo(
				testutil.WithDaysInactive(10),
				testutil.WithStars(100),
				testutil.WithForks(50),
			),
			expectReason: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			oldWd, _ := os.Getwd()
			err := os.Chdir(tmpDir)
			require.NoError(t, err)
			defer func() { _ = os.Chdir(oldWd) }()

			filename, err := Export([]github.Repository{tt.repo}, "testowner")
			require.NoError(t, err)

			content, err := os.ReadFile(filename)
			require.NoError(t, err)

			var data ExportData
			err = json.Unmarshal(content, &data)
			require.NoError(t, err)

			require.Len(t, data.Repositories, 1)
			reason := data.Repositories[0].Reason

			if tt.expectReason != "" {
				assert.Equal(t, tt.expectReason, reason)
			}

			for _, expected := range tt.expectContains {
				assert.Contains(t, reason, expected, "reason should contain: %s", expected)
			}
		})
	}
}

func TestExport_FilenameFormat(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	err := os.Chdir(tmpDir)
	require.NoError(t, err)
	defer func() { _ = os.Chdir(oldWd) }()

	filename, err := Export([]github.Repository{
		testutil.NewTestRepo(),
	}, "testowner")
	require.NoError(t, err)

	// Expected format: archived-repos-YYYY-MM-DD-HHMMSS.json
	pattern := `^archived-repos-\d{4}-\d{2}-\d{2}-\d{6}\.json$`
	matched, err := regexp.MatchString(pattern, filename)
	require.NoError(t, err)
	assert.True(t, matched, "filename %q should match pattern %q", filename, pattern)

	// Verify file exists
	_, err = os.Stat(filename)
	assert.NoError(t, err, "exported file should exist")
}

func TestExport_JSONValidFormat(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	err := os.Chdir(tmpDir)
	require.NoError(t, err)
	defer func() { _ = os.Chdir(oldWd) }()

	repos := []github.Repository{
		testutil.NewTestRepo(testutil.WithName("repo1")),
		testutil.NewTestRepo(testutil.WithName("repo2")),
	}

	filename, err := Export(repos, "testowner")
	require.NoError(t, err)

	content, err := os.ReadFile(filename)
	require.NoError(t, err)

	// Check that JSON is valid and indented
	assert.True(t, json.Valid(content), "JSON should be valid")
	assert.Contains(t, string(content), "\n", "JSON should be indented (contain newlines)")
	assert.Contains(t, string(content), "  ", "JSON should be indented (contain spaces)")
}

func TestExport_WriteError(t *testing.T) {
	// Test that export fails gracefully when directory is not writable
	// Create a read-only directory
	tmpDir := t.TempDir()
	readOnlyDir := tmpDir + "/readonly"
	err := os.Mkdir(readOnlyDir, 0555)
	require.NoError(t, err)

	oldWd, _ := os.Getwd()
	err = os.Chdir(readOnlyDir)
	require.NoError(t, err)
	defer func() { _ = os.Chdir(oldWd) }()

	_, err = Export([]github.Repository{
		testutil.NewTestRepo(),
	}, "testowner")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to write file")
}
