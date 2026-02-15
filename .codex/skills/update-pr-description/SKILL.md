---
name: update-pr-description
description: Use when a user asks to update a GitHub PR description from the actual branch changes, test status, and tracking context.
---

# Update PR Description

Use this skill to reliably refresh a PR body so it matches the current branch state.

## Preconditions

- `gh` CLI is installed and authenticated.
- Current branch has an open PR, or the PR number is provided.

## Workflow

1. Resolve target PR.
- If PR number is provided, use it.
- Otherwise detect from branch: `gh pr view --json number,title,body,url,headRefName,baseRefName`.

2. Collect current change context.
- `git diff --name-status origin/<base>...HEAD`
- `git log --oneline origin/<base>..HEAD`
- Include relevant tracking IDs (for example beads issue IDs) and quality gate results.

3. Build a concise PR body.
- `Summary`: what changed and why.
- `Changes`: key files/modules/features.
- `Testing`: commands run and pass/fail status.
- `Tracking`: issue IDs or links when applicable.
- `Follow-ups`: deferred major/risky work, if any.

4. Apply update.
- Write body to a temp file.
- `gh pr edit <number> --body-file <file>`

5. Verify.
- `gh pr view <number> --json body,url,title`
- Confirm body matches intended content.

## Output Rules

- Use exact versions/IDs when mentioning dependency updates or issue tracking.
- Do not claim tests were run unless command output confirmed it.
- Keep PR body factual and scan-friendly.

## Prompt Template

`Update the current PR description from actual branch diffs, include summary, testing, and tracking, then verify with gh pr view.`
