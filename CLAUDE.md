# Agent Instructions



## License Header

All Go source files must include this header:
```go
// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT
```


## Issue Tracking

This project uses **bd** (beads) for issue tracking. Run `bd onboard` to get started.

```bash
bd ready              # Find available work
bd show <id>          # View issue details
bd update <id> --status in_progress  # Claim work
bd close <id>         # Complete work
bd sync               # Sync with git
```

**Note:** Git hooks for beads (`pre-commit`, `post-merge`) are disabled. They were interfering with commits made directly in Cursor. Run `bd sync` manually when you need to sync beads with git. Hooks are preserved as `.disabled` files in `.git/hooks/` if you want to re-enable them.

**MANDATORY WORKFLOW:**

1. **File issues for remaining work** - Create issues with `bd create` for anything that needs follow-up
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **Update issue status** - Close finished work with `bd close`, update in-progress items
4. **Sync beads** - Run `bd sync` to commit beads changes
5. **Commit and push** - Use the **commit-manager agent** to stage, commit, and push code changes
6. **Verify** - All changes committed AND pushed
7. **Hand off** - Provide context for next session

**CRITICAL RULES:**
- **NEVER run git add/commit/push commands directly** - always use the commit-manager agent
- Use the commit-manager agent for ALL version control operations (staging, commits, pushes, PRs)
- If resuming the commit-manager agent fails (API error, concurrency issues), **start a new commit-manager agent** instead of falling back to direct git commands
