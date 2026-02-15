# Workflow

Repository-level workflow guidance for dependency maintenance and Codex-assisted execution.

## Dependency Update Policy

- Safe updates are grouped: patch/minor updates for direct Go dependencies can be batched together.
- Major updates are isolated: one major dependency upgrade per PR for easier review and rollback.
- Default cadence:
  - Weekly: safe grouped updates
  - Monthly: major upgrade review batch
  - ASAP: security-critical updates

## Dependabot or Renovate Strategy

- Configure grouped PRs for safe Go updates.
- Keep major version bumps in separate PRs.
- Include clear labels (for example: `dependencies`, `safe-update`, `major-upgrade`).
- Require CI (`just check`) before merge.

## Codex Maintenance Sweep

Use the repository skill:
- `.codex/skills/maintenance-sweep/SKILL.md`

Beads tracking is mandatory for maintenance sweeps.

Expected execution flow:
1. create or attach a beads issue (`bd create` or provided `<id>`)
2. set issue in progress (`bd update <id> --status in_progress`)
3. update patch/minor direct dependencies
4. `just check`
5. close and sync beads (`bd close <id>`, `bd sync`)
6. provide handoff notes

## Standard Codex Prompt

Use this prompt for routine dependency maintenance:

`Upgrade only patch/minor direct deps, run checks, summarize risk per dep, no commits.`

## Output Expectations

Dependency maintenance handoff should always include:
- beads issue id
- branch name
- dependency changes (`from -> to`)
- risk per dependency (`low`, `medium`, `high`)
- test/check result (`just check`)
- deferred upgrades with reason
