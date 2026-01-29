# Release Process

This project uses automated releases based on conventional commits.

## How It Works

```
Push to main → Auto Tag → Release Workflow → GitHub Release with binaries
```

1. **Auto Tag** (`.github/workflows/auto-tag.yml`) - Analyzes commits and creates a semver tag
2. **Release** (`.github/workflows/release.yml`) - Builds binaries and creates GitHub Release

## Version Bumping

Version is determined automatically from commit messages:

| Commit Type | Bump | Example |
|-------------|------|---------|
| `fix:` | patch | 0.1.0 → 0.1.1 |
| `docs:`, `chore:`, `ci:`, `test:`, `refactor:` | patch | 0.1.0 → 0.1.1 |
| `feat:` | minor | 0.1.0 → 0.2.0 |
| `feat!:` or `BREAKING CHANGE:` | major | 0.1.0 → 1.0.0 |

## Commit Message Format

```
<type>(<scope>): <description>

[optional body]

[optional BREAKING CHANGE: description]
```

**Examples:**
```bash
# Patch release
fix(tui): correct cursor position on filter change
docs: update README with new flags

# Minor release
feat(archive): add unarchive functionality

# Major release
feat!: change config file format

# Or with body
feat: new export format

BREAKING CHANGE: JSON export structure changed, see migration guide
```

## Manual Releases

Use the Makefile for manual control:

```bash
make show-version    # Show current version
make bump-patch      # 0.1.0 → 0.1.1
make bump-minor      # 0.1.0 → 0.2.0
make bump-major      # 0.1.0 → 1.0.0
git push origin main --tags
```

## Changelog

Changelogs are auto-generated using [git-cliff](https://git-cliff.org/) from conventional commits. Config in `cliff.toml`.

## CI/CD Files

- `.github/workflows/ci.yml` - Tests, lint, build on every push/PR
- `.github/workflows/auto-tag.yml` - Creates version tag on push to main
- `.github/workflows/release.yml` - Builds binaries on new tag
- `.github/dependabot.yml` - Weekly dependency updates
- `cliff.toml` - Changelog generation config
- `Makefile` - Local build and manual version bumping
