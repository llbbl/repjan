// SPDX-FileCopyrightText: 2026 Logan Lindquist Land
// SPDX-License-Identifier: FSL-1.1-MIT

package github

import (
	"encoding/json"
	"time"
)

// Repository represents a GitHub repository with fields matching gh CLI JSON output.
type Repository struct {
	Owner             string    `json:"-"` // Populated from ownerJSON
	Name              string    `json:"name"`
	Description       string    `json:"description"`
	PushedAt          time.Time `json:"pushedAt"`
	CreatedAt         time.Time `json:"createdAt"`
	StargazerCount    int       `json:"stargazerCount"`
	ForkCount         int       `json:"forkCount"`
	IsArchived        bool      `json:"isArchived"`
	IsFork            bool      `json:"isFork"`
	IsPrivate         bool      `json:"isPrivate"`
	PrimaryLanguage   string    `json:"-"` // Populated from primaryLanguageJSON
	DaysSinceActivity int       `json:"-"` // Calculated field
	MarkedForArchive  bool      `json:"-"` // UI state
	ArchiveReason     string    `json:"-"` // UI state
}

// ownerJSON represents the nested owner object from gh CLI.
type ownerJSON struct {
	Login string `json:"login"`
}

// primaryLanguageJSON represents the nested primaryLanguage object from gh CLI.
type primaryLanguageJSON struct {
	Name string `json:"name"`
}

// repositoryJSON is used for unmarshaling the raw gh CLI JSON response.
type repositoryJSON struct {
	Owner           *ownerJSON           `json:"owner"`
	Name            string               `json:"name"`
	Description     string               `json:"description"`
	PushedAt        time.Time            `json:"pushedAt"`
	CreatedAt       time.Time            `json:"createdAt"`
	StargazerCount  int                  `json:"stargazerCount"`
	ForkCount       int                  `json:"forkCount"`
	IsArchived      bool                 `json:"isArchived"`
	IsFork          bool                 `json:"isFork"`
	IsPrivate       bool                 `json:"isPrivate"`
	PrimaryLanguage *primaryLanguageJSON `json:"primaryLanguage"`
}

// UnmarshalJSON implements custom JSON unmarshaling to handle gh CLI's nested format.
func (r *Repository) UnmarshalJSON(data []byte) error {
	var raw repositoryJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	r.Name = raw.Name
	r.Description = raw.Description
	r.PushedAt = raw.PushedAt
	r.CreatedAt = raw.CreatedAt
	r.StargazerCount = raw.StargazerCount
	r.ForkCount = raw.ForkCount
	r.IsArchived = raw.IsArchived
	r.IsFork = raw.IsFork
	r.IsPrivate = raw.IsPrivate

	if raw.Owner != nil {
		r.Owner = raw.Owner.Login
	}

	if raw.PrimaryLanguage != nil {
		r.PrimaryLanguage = raw.PrimaryLanguage.Name
	}

	return nil
}

// FullName returns the repository's full name in "owner/name" format.
func (r *Repository) FullName() string {
	return r.Owner + "/" + r.Name
}

// CalculateDaysSinceActivity calculates and sets DaysSinceActivity from PushedAt.
func (r *Repository) CalculateDaysSinceActivity() {
	if r.PushedAt.IsZero() {
		r.DaysSinceActivity = 0
		return
	}
	duration := time.Since(r.PushedAt)
	r.DaysSinceActivity = int(duration.Hours() / 24)
}
