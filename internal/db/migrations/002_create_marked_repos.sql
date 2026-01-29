-- SPDX-FileCopyrightText: 2026 api2spec
-- SPDX-License-Identifier: FSL-1.1-MIT

-- +goose Up
CREATE TABLE marked_repos (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    owner TEXT NOT NULL,
    repo_name TEXT NOT NULL,
    marked_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(owner, repo_name)
);

CREATE INDEX idx_marked_repos_owner ON marked_repos(owner);

-- +goose Down
DROP TABLE marked_repos;
