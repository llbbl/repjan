-- SPDX-FileCopyrightText: 2026 api2spec
-- SPDX-License-Identifier: FSL-1.1-MIT

-- +goose Up
CREATE TABLE repositories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    owner TEXT NOT NULL,
    name TEXT NOT NULL,
    full_name TEXT NOT NULL UNIQUE,
    description TEXT,
    stars INTEGER DEFAULT 0,
    forks INTEGER DEFAULT 0,
    is_archived BOOLEAN DEFAULT FALSE,
    is_fork BOOLEAN DEFAULT FALSE,
    is_private BOOLEAN DEFAULT FALSE,
    primary_language TEXT,
    pushed_at DATETIME,
    created_at DATETIME,
    days_since_activity INTEGER DEFAULT 0,
    synced_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(owner, name)
);

CREATE INDEX idx_repositories_owner ON repositories(owner);
CREATE INDEX idx_repositories_synced_at ON repositories(synced_at);

-- +goose Down
DROP TABLE repositories;
