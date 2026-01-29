// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

// Package sync provides background repository synchronization.
package sync

import (
	"log/slog"
	"time"

	"github.com/llbbl/repjan/internal/github"
	"github.com/llbbl/repjan/internal/store"
)

// SyncMsgType represents the type of sync message.
type SyncMsgType int

const (
	// SyncStarted indicates a sync operation has begun.
	SyncStarted SyncMsgType = iota
	// SyncCompleted indicates a sync operation completed successfully.
	SyncCompleted
	// SyncError indicates a sync operation encountered an error.
	SyncError
)

// SyncMsg represents a message sent from the syncer to the TUI.
type SyncMsg struct {
	Type  SyncMsgType
	Repos []github.Repository // only populated for SyncCompleted
	Error error               // only populated for SyncError
}

// SyncResult represents the result of a single sync operation.
type SyncResult struct {
	Repos []github.Repository
	Error error
}

// Syncer handles background repository synchronization.
type Syncer struct {
	store    *store.Store
	client   *github.Client
	owner    string
	interval time.Duration
	stopCh   chan struct{}
	msgCh    chan SyncMsg
}

// New creates a new Syncer with the given configuration.
func New(store *store.Store, client *github.Client, owner string, interval time.Duration) *Syncer {
	return &Syncer{
		store:    store,
		client:   client,
		owner:    owner,
		interval: interval,
		stopCh:   make(chan struct{}),
		msgCh:    make(chan SyncMsg, 10), // buffered to prevent blocking
	}
}

// Start begins background sync. Returns a channel that receives messages for the TUI.
// The channel will be closed when Stop is called.
func (s *Syncer) Start() <-chan SyncMsg {
	go s.run()
	return s.msgCh
}

// Stop stops the background sync and closes the message channel.
func (s *Syncer) Stop() {
	close(s.stopCh)
}

// SyncOnce performs a single sync and returns the result.
// This is useful for testing or one-off sync operations.
func (s *Syncer) SyncOnce() SyncResult {
	repos, err := s.doSync()
	return SyncResult{
		Repos: repos,
		Error: err,
	}
}

// run is the main background sync loop.
func (s *Syncer) run() {
	defer close(s.msgCh)

	// Check if we need to sync on startup
	if s.shouldSyncOnStartup() {
		s.performSync()
	}

	// Start the ticker for periodic sync
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			slog.Debug("sync tick", "component", "sync", "interval", s.interval)
			s.performSync()
		}
	}
}

// shouldSyncOnStartup checks if data is stale enough to warrant immediate sync.
func (s *Syncer) shouldSyncOnStartup() bool {
	lastSync, err := s.store.GetLastSyncTime(s.owner)
	if err != nil {
		// If we can't determine last sync time, sync anyway
		slog.Warn("failed to get last sync time", "component", "sync", "error", err)
		return true
	}

	// If no previous sync, we should sync
	if lastSync.IsZero() {
		return true
	}

	// If last sync was longer than interval ago, sync now
	return time.Since(lastSync) >= s.interval
}

// performSync executes a sync and sends appropriate messages.
func (s *Syncer) performSync() {
	// Send start message
	select {
	case s.msgCh <- SyncMsg{Type: SyncStarted}:
	case <-s.stopCh:
		return
	}

	repos, err := s.doSync()

	// Send result message
	var msg SyncMsg
	if err != nil {
		msg = SyncMsg{
			Type:  SyncError,
			Error: err,
		}
		slog.Error("sync failed", "component", "sync", "error", err, "owner", s.owner)
	} else {
		msg = SyncMsg{
			Type:  SyncCompleted,
			Repos: repos,
		}
		slog.Info("sync completed", "component", "sync", "owner", s.owner, "repos", len(repos))
	}

	select {
	case s.msgCh <- msg:
	case <-s.stopCh:
		return
	}
}

// doSync fetches repositories from GitHub and upserts them to the database.
func (s *Syncer) doSync() ([]github.Repository, error) {
	slog.Debug("starting sync", "component", "sync", "owner", s.owner)

	// Fetch from GitHub
	repos, err := s.client.FetchRepositories(s.owner)
	if err != nil {
		return nil, err
	}

	// Upsert to database
	if err := s.store.UpsertRepositories(s.owner, repos); err != nil {
		return nil, err
	}

	return repos, nil
}
