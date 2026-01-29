-- SPDX-FileCopyrightText: 2026 api2spec
-- SPDX-License-Identifier: FSL-1.1-MIT

-- +goose Up
CREATE TABLE sync_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    owner TEXT NOT NULL,
    started_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at DATETIME,
    status TEXT NOT NULL DEFAULT 'running',  -- running, success, error, partial
    repos_fetched INTEGER DEFAULT 0,
    repos_inserted INTEGER DEFAULT 0,
    repos_updated INTEGER DEFAULT 0,
    error_message TEXT,
    duration_ms INTEGER
);

CREATE INDEX idx_sync_history_owner ON sync_history(owner);
CREATE INDEX idx_sync_history_started_at ON sync_history(started_at DESC);
CREATE INDEX idx_sync_history_owner_status ON sync_history(owner, status);

-- +goose Down
DROP INDEX IF EXISTS idx_sync_history_owner_status;
DROP INDEX IF EXISTS idx_sync_history_started_at;
DROP INDEX IF EXISTS idx_sync_history_owner;
DROP TABLE IF EXISTS sync_history;
