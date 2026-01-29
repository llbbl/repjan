// SPDX-FileCopyrightText: 2026 Logan Lindquist Land
// SPDX-License-Identifier: FSL-1.1-MIT

// Package github provides a wrapper around the gh CLI for interacting with GitHub.
package github

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// Custom error types for common gh CLI failures.
var (
	// ErrNotAuthenticated indicates the user is not authenticated with gh CLI.
	ErrNotAuthenticated = errors.New("not authenticated with gh CLI")

	// ErrNotFound indicates the requested resource was not found.
	ErrNotFound = errors.New("resource not found")

	// ErrRateLimit indicates the GitHub API rate limit has been exceeded.
	ErrRateLimit = errors.New("GitHub API rate limit exceeded")
)

// Client wraps the gh CLI to provide GitHub API access.
type Client struct {
	executor CommandExecutor
}

// NewClient creates a new GitHub client with the provided executor.
func NewClient(executor CommandExecutor) *Client {
	return &Client{executor: executor}
}

// NewDefaultClient creates a new GitHub client using the real gh CLI executor.
func NewDefaultClient() *Client {
	return &Client{executor: &RealExecutor{}}
}

// FetchRepositories fetches all repositories for the given owner.
// It returns up to 1000 repositories with their metadata populated.
func (c *Client) FetchRepositories(owner string) ([]Repository, error) {
	if owner == "" {
		return nil, fmt.Errorf("owner cannot be empty")
	}

	output, err := c.executor.Execute(
		"gh", "repo", "list", owner,
		"--json", "name,description,pushedAt,createdAt,stargazerCount,forkCount,isArchived,isFork,isPrivate,primaryLanguage,owner",
		"--limit", "1000",
	)
	if err != nil {
		return nil, c.wrapError(err, output, "fetching repositories for %s", owner)
	}

	// Handle empty response gracefully
	trimmed := strings.TrimSpace(string(output))
	if trimmed == "" || trimmed == "[]" {
		return []Repository{}, nil
	}

	var repos []Repository
	if err := json.Unmarshal(output, &repos); err != nil {
		return nil, fmt.Errorf("parsing repository list: %w", err)
	}

	// Calculate days since activity for each repository
	for i := range repos {
		repos[i].CalculateDaysSinceActivity()
	}

	return repos, nil
}

// GetAuthenticatedUser returns the login of the currently authenticated user.
// This is useful when --owner flag is not provided.
func (c *Client) GetAuthenticatedUser() (string, error) {
	output, err := c.executor.Execute("gh", "api", "user", "--jq", ".login")
	if err != nil {
		return "", c.wrapError(err, output, "getting authenticated user")
	}

	login := strings.TrimSpace(string(output))
	if login == "" {
		return "", ErrNotAuthenticated
	}

	return login, nil
}

// ArchiveRepository archives the specified repository.
func (c *Client) ArchiveRepository(owner, name string) error {
	if owner == "" || name == "" {
		return fmt.Errorf("owner and name cannot be empty")
	}

	repoFullName := owner + "/" + name
	output, err := c.executor.Execute("gh", "repo", "archive", repoFullName, "--yes")
	if err != nil {
		return c.wrapError(err, output, "archiving repository %s", repoFullName)
	}

	return nil
}

// FetchReadme fetches the README content for the specified repository.
// Returns an empty string (not an error) if no README exists.
func (c *Client) FetchReadme(owner, name string) (string, error) {
	if owner == "" || name == "" {
		return "", fmt.Errorf("owner and name cannot be empty")
	}

	endpoint := fmt.Sprintf("repos/%s/%s/readme", owner, name)
	output, err := c.executor.Execute("gh", "api", endpoint, "--jq", ".content")
	if err != nil {
		// Check if this is a 404 (no README) - return empty string, not error
		if c.isNotFoundError(err, output) {
			return "", nil
		}
		return "", c.wrapError(err, output, "fetching README for %s/%s", owner, name)
	}

	// Handle empty response
	content := strings.TrimSpace(string(output))
	if content == "" {
		return "", nil
	}

	// GitHub returns README content as base64-encoded
	// The content may have newlines that need to be removed before decoding
	content = strings.ReplaceAll(content, "\n", "")
	decoded, err := base64.StdEncoding.DecodeString(content)
	if err != nil {
		return "", fmt.Errorf("decoding README content: %w", err)
	}

	return string(decoded), nil
}

// wrapError wraps command execution errors with context and checks for common error types.
func (c *Client) wrapError(err error, output []byte, format string, args ...any) error {
	msg := fmt.Sprintf(format, args...)
	errOutput := strings.ToLower(string(output))
	errString := strings.ToLower(err.Error())
	combined := errOutput + " " + errString

	// Check for authentication errors
	if strings.Contains(combined, "not logged in") ||
		strings.Contains(combined, "authentication") ||
		strings.Contains(combined, "gh auth login") {
		return fmt.Errorf("%s: %w", msg, ErrNotAuthenticated)
	}

	// Check for rate limit errors
	if strings.Contains(combined, "rate limit") ||
		strings.Contains(combined, "api rate limit exceeded") {
		return fmt.Errorf("%s: %w", msg, ErrRateLimit)
	}

	// Check for not found errors
	if c.isNotFoundError(err, output) {
		return fmt.Errorf("%s: %w", msg, ErrNotFound)
	}

	// Generic error with output context
	if len(output) > 0 {
		return fmt.Errorf("%s: %w (output: %s)", msg, err, strings.TrimSpace(string(output)))
	}
	return fmt.Errorf("%s: %w", msg, err)
}

// isNotFoundError checks if the error indicates a 404 Not Found response.
func (c *Client) isNotFoundError(err error, output []byte) bool {
	combined := strings.ToLower(string(output)) + " " + strings.ToLower(err.Error())
	return strings.Contains(combined, "404") ||
		strings.Contains(combined, "not found") ||
		strings.Contains(combined, "could not resolve")
}
