// SPDX-FileCopyrightText: 2026 Logan Lindquist Land
// SPDX-License-Identifier: FSL-1.1-MIT

// Package testutil provides testing utilities and helpers for the repjan project.
package testutil

import (
	"time"

	"github.com/llbbl/repjan/internal/github"
)

// RepoOption is a functional option for configuring test repositories.
type RepoOption func(*github.Repository)

// NewTestRepo creates a Repository with sensible defaults for testing.
// Use the With* option functions to customize specific fields.
func NewTestRepo(opts ...RepoOption) github.Repository {
	repo := github.Repository{
		Owner:           "testowner",
		Name:            "testrepo",
		Description:     "A test repository",
		PushedAt:        time.Now().AddDate(0, 0, -30), // 30 days ago
		CreatedAt:       time.Now().AddDate(-1, 0, 0),  // 1 year ago
		StargazerCount:  5,
		ForkCount:       2,
		IsArchived:      false,
		IsFork:          false,
		IsPrivate:       false,
		PrimaryLanguage: "Go",
	}

	for _, opt := range opts {
		opt(&repo)
	}

	repo.CalculateDaysSinceActivity()
	return repo
}

// WithOwner sets the repository owner.
func WithOwner(owner string) RepoOption {
	return func(r *github.Repository) {
		r.Owner = owner
	}
}

// WithName sets the repository name.
func WithName(name string) RepoOption {
	return func(r *github.Repository) {
		r.Name = name
	}
}

// WithDescription sets the repository description.
func WithDescription(desc string) RepoOption {
	return func(r *github.Repository) {
		r.Description = desc
	}
}

// WithStars sets the stargazer count.
func WithStars(n int) RepoOption {
	return func(r *github.Repository) {
		r.StargazerCount = n
	}
}

// WithForks sets the fork count.
func WithForks(n int) RepoOption {
	return func(r *github.Repository) {
		r.ForkCount = n
	}
}

// WithDaysInactive sets the PushedAt to n days ago and recalculates DaysSinceActivity.
func WithDaysInactive(days int) RepoOption {
	return func(r *github.Repository) {
		r.PushedAt = time.Now().AddDate(0, 0, -days)
		r.CalculateDaysSinceActivity()
	}
}

// WithLanguage sets the primary language.
func WithLanguage(lang string) RepoOption {
	return func(r *github.Repository) {
		r.PrimaryLanguage = lang
	}
}

// WithFork sets whether the repository is a fork.
func WithFork(isFork bool) RepoOption {
	return func(r *github.Repository) {
		r.IsFork = isFork
	}
}

// WithArchived sets whether the repository is archived.
func WithArchived(isArchived bool) RepoOption {
	return func(r *github.Repository) {
		r.IsArchived = isArchived
	}
}

// WithPrivate sets whether the repository is private.
func WithPrivate(isPrivate bool) RepoOption {
	return func(r *github.Repository) {
		r.IsPrivate = isPrivate
	}
}

// WithCreatedAt sets the creation time.
func WithCreatedAt(t time.Time) RepoOption {
	return func(r *github.Repository) {
		r.CreatedAt = t
	}
}

// WithPushedAt sets the last push time and recalculates DaysSinceActivity.
func WithPushedAt(t time.Time) RepoOption {
	return func(r *github.Repository) {
		r.PushedAt = t
		r.CalculateDaysSinceActivity()
	}
}

// WithMarkedForArchive sets the archive marker state (for UI testing).
func WithMarkedForArchive(marked bool, reason string) RepoOption {
	return func(r *github.Repository) {
		r.MarkedForArchive = marked
		r.ArchiveReason = reason
	}
}
