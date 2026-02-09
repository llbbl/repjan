# Agent Instructions


## Git Workflow

**ALWAYS work in feature branches, not main.**

- New features: `feat/<description>` (e.g., `feat/add-spinner`)
- Bug fixes: `fix/<description>` (e.g., `fix/tui-freeze`)
- Refactoring: `refactor/<description>`
- Multiple changes: `fix/<primary-change>-and-<secondary>` or descriptive name

**Workflow:**
1. Create feature branch from main: `git checkout -b feat/my-feature`
2. Make changes and commit using commit-manager agent
3. Create PR when ready for review/merge
4. Merge to main via PR (not direct commits)

**Direct commits to main are only allowed for:**
- Emergency hotfixes
- Documentation-only changes (optional)


## License Header

All Go source files must include this header:
```go
// SPDX-FileCopyrightText: 2026 Logan Lindquist Land
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
- **DO NOT specify attribution lines or Co-Authored-By** - Let commit-manager follow its own guidelines for those


## Commit Message Format (Conventional Commits)

This project uses **Conventional Commits** for git-cliff changelog generation. When calling commit-manager, instruct it to use this format for the commit message:

```
<type>(<scope>): <short description>

[optional body]
```

**Types:**
- `feat` - New feature
- `fix` - Bug fix
- `docs` - Documentation only
- `style` - Formatting, no code change
- `refactor` - Code restructuring, no behavior change
- `perf` - Performance improvement
- `test` - Adding/updating tests
- `chore` - Maintenance tasks
- `ci` - CI/CD changes
- `deps` - Dependency updates

**Examples:**
```
feat(tui): add pagination with viewport scrolling
fix(archive): send completion message when all repos processed
docs: update README with installation instructions
ci: add GitHub Actions release workflow
```

**When calling commit-manager**, include: "Use conventional commit format (type(scope): description)" but do NOT specify attribution or Co-Authored-By formatting.

**Version Tags:**
Use semver format without `v` prefix: `0.1.0`, `1.2.3` (not `v0.1.0`)


## Build System

This project uses **just** (not make). Run `just` to see available commands:

```bash
just build         # Build binary
just test          # Run tests
just lint          # Run linter
just check         # Run lint + test
just show-version  # Show current version
just bump-patch    # Bump patch version and create tag
just release-patch # Bump, tag, and push
```


## TUI Defaults

- Default sort: Activity ascending (oldest first, not most recent)
- All keybindings should be visible in the footer navigation hints
