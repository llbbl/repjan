-- SPDX-FileCopyrightText: 2026 api2spec
-- SPDX-License-Identifier: FSL-1.1-MIT

-- +goose Up
CREATE TABLE repo_changes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    owner TEXT NOT NULL,
    repo_name TEXT NOT NULL,
    action TEXT NOT NULL,  -- archived, marked, unmarked, deleted, synced
    performed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    performed_by TEXT NOT NULL DEFAULT 'user',  -- user, system, sync
    previous_state TEXT,  -- JSON blob of before state (nullable)
    new_state TEXT,       -- JSON blob of after state (nullable)
    notes TEXT            -- optional notes/reason
);

CREATE INDEX idx_repo_changes_owner ON repo_changes(owner);
CREATE INDEX idx_repo_changes_repo ON repo_changes(owner, repo_name);
CREATE INDEX idx_repo_changes_action ON repo_changes(action);
CREATE INDEX idx_repo_changes_performed_at ON repo_changes(performed_at DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_repo_changes_performed_at;
DROP INDEX IF EXISTS idx_repo_changes_action;
DROP INDEX IF EXISTS idx_repo_changes_repo;
DROP INDEX IF EXISTS idx_repo_changes_owner;
DROP TABLE IF EXISTS repo_changes;
