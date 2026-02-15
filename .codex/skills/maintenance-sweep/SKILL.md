---
name: maintenance-sweep
description: Use when performing routine dependency maintenance in this repo: create or attach beads work, apply safe Go dependency updates (patch/minor direct dependencies), run checks, update issue status, sync beads, and provide handoff notes without committing.
---

# Maintenance Sweep

Use this skill for low-risk dependency maintenance runs in this repository.

## Scope

- Default scope: direct Go dependencies in `go.mod`.
- Upgrade policy: patch/minor updates only.
- Exclude major version upgrades unless the user explicitly asks.
- Beads tracking is mandatory for every maintenance sweep.
- Do not commit by default.

## Workflow

1. Create or attach a beads task.
- If a beads issue is provided: `bd show <id>` then `bd update <id> --status in_progress`.
- If no issue is provided: create one for the maintenance run with `bd create`, then set it in progress.

2. Start from updated branch context.
- If requested to update main: `git checkout main` then `git pull --ff-only`.
- Create a working branch: `git checkout -b fix/deps-maintenance-<short-date-or-topic>`.

3. Scan for outdated dependencies.
- `go list -m -u all`
- Identify direct dependencies from `go.mod` (`require` block without `// indirect`).
- Build a candidate list limited to patch/minor upgrades for direct dependencies.

4. Apply upgrades conservatively.
- Prefer targeted upgrades: `go get module@version`.
- Run `go mod tidy` after updates.
- Keep unrelated dependency churn minimal.

5. Run quality gates.
- `just check`
- If checks fail, fix issues or revert only the risky dependency bump and re-run checks.

6. Update tracking.
- Close completed issue: `bd close <id>`.
- Sync beads state: `bd sync`.

7. Handoff output.
- Provide:
  - beads issue id
  - branch name
  - upgraded dependencies (from -> to)
  - risk notes per dependency
  - `just check` result
  - deferred upgrades and reason
- Explicitly state no commits were made.

## Standard Prompt Template

Use this template when running the sweep:

`Upgrade only patch/minor direct deps, run checks, summarize risk per dep, no commits.`

## Risk Heuristics

- Low risk: patch updates and minor updates on stable APIs used indirectly.
- Medium risk: minor updates on direct libraries with broad API surface.
- High risk: major updates, database driver leaps, or upgrades that trigger widespread transitive changes.
