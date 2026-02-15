# repjan - justfile
# Run `just` or `just help` to see available commands

# Get current version from git tags (defaults to 0.0.0 if no tags)
version := `git describe --tags --abbrev=0 2>/dev/null || echo "0.0.0"`
binary := "repjan"

# Default recipe: show help
default:
    @just --list --unsorted

# Build binary for current platform
build:
    go build -ldflags="-s -w -X main.version={{version}}" -o ./{{binary}} ./cmd/repjan

# Run the app
run *ARGS:
    go run ./cmd/repjan {{ ARGS }}

# Run the app with debug logging (logs to stderr)
debug *ARGS:
    REPJAN_LOG_LEVEL=debug go run ./cmd/repjan {{ ARGS }}

# Run the app with debug logging to file
debug-file *ARGS:
    REPJAN_LOG_LEVEL=debug go run ./cmd/repjan {{ ARGS }} 2>debug.log

# Run tests
test *ARGS:
    go test -v -race ./... {{ ARGS }}

# Run linter
lint:
    #!/bin/sh
    set -e
    go vet ./...
    UNFORMATTED=$(gofmt -l .)
    if [ -n "$UNFORMATTED" ]; then
        echo "Files not formatted:"
        echo "$UNFORMATTED"
        exit 1
    fi

# Format code
format:
    gofmt -w .

# Clean build artifacts
clean:
    rm -f ./{{binary}}

# Run all checks (lint + test)
check: lint test

# Ship current feature branch: check, sync beads, commit, and push
# Usage: just ship "chore(scope): description"
ship message:
    #!/bin/sh
    set -e
    BRANCH=$(git branch --show-current)
    if [ "$BRANCH" = "main" ]; then
        echo "Refusing to ship from main. Use a feature branch."
        exit 1
    fi
    if ! printf '%s' "{{message}}" | grep -Eq '^(feat|fix|docs|style|refactor|perf|test|chore|ci|deps)(\([a-z0-9._/-]+\))?: .+'; then
        echo "Commit message must use conventional commit format: type(scope): description"
        exit 1
    fi
    just check
    bd sync
    git add -A
    if git diff --cached --quiet; then
        echo "No staged changes to commit."
        exit 1
    fi
    git commit -m "{{message}}"
    git push -u origin "$BRANCH"

# Suggest a conventional commit message command for current working tree
suggest-commit:
    #!/bin/sh
    set -e
    FILES=$( { git diff --name-only HEAD; git ls-files --others --exclude-standard; } | sed '/^$$/d' | sort -u )
    if [ -z "$FILES" ]; then
        echo 'No changes detected.'
        exit 1
    fi

    TYPE="chore"
    SCOPE="repo"
    DESC="update project files"

    ONLY_DOCS=true
    ONLY_DEPS=true
    ONLY_CI=true
    ONLY_TESTS=true
    HAS_JUSTFILE=false

    for f in $FILES; do
        case "$f" in
            docs/*|*.md) ;;
            *) ONLY_DOCS=false ;;
        esac
        case "$f" in
            go.mod|go.sum) ;;
            *) ONLY_DEPS=false ;;
        esac
        case "$f" in
            .github/workflows/*) ;;
            *) ONLY_CI=false ;;
        esac
        case "$f" in
            *_test.go) ;;
            *) ONLY_TESTS=false ;;
        esac
        if [ "$f" = "justfile" ]; then
            HAS_JUSTFILE=true
        fi
    done

    if [ "$ONLY_DEPS" = true ]; then
        TYPE="deps"
        SCOPE="go"
        DESC="update Go module dependencies"
    elif [ "$ONLY_DOCS" = true ]; then
        TYPE="docs"
        SCOPE="docs"
        DESC="update project documentation"
    elif [ "$ONLY_CI" = true ]; then
        TYPE="ci"
        SCOPE="github"
        DESC="update workflow configuration"
    elif [ "$ONLY_TESTS" = true ]; then
        TYPE="test"
        SCOPE="go"
        DESC="update Go tests"
    elif [ "$HAS_JUSTFILE" = true ]; then
        TYPE="chore"
        SCOPE="build"
        DESC="update justfile workflow"
    fi

    echo "just ship \"$TYPE($SCOPE): $DESC\""

# Generate full changelog
changelog:
    git cliff -o CHANGELOG.md

# Preview unreleased changes
changelog-preview:
    git cliff --unreleased

# ============================================================================
# Version Management
# ============================================================================

# Show current version
show-version:
    @echo "Current version: {{version}}"
    @echo ""
    @echo "Recent tags:"
    @git tag --sort=-version:refname | head -5

# Bump patch version (0.1.2 → 0.1.3)
bump-patch:
    #!/bin/sh
    set -e
    CURRENT="{{version}}"
    echo "Current version: $CURRENT"
    MAJOR=$(echo "$CURRENT" | cut -d. -f1)
    MINOR=$(echo "$CURRENT" | cut -d. -f2)
    PATCH=$(echo "$CURRENT" | cut -d. -f3)
    NEW="$MAJOR.$MINOR.$((PATCH + 1))"
    echo "New version: $NEW"
    git tag -a "$NEW" -m "Release $NEW"
    echo ""
    echo "Created tag $NEW"
    echo ""
    echo "Push with:"
    echo "  git push origin main --tags"

# Bump minor version (0.1.2 → 0.2.0)
bump-minor:
    #!/bin/sh
    set -e
    CURRENT="{{version}}"
    echo "Current version: $CURRENT"
    MAJOR=$(echo "$CURRENT" | cut -d. -f1)
    MINOR=$(echo "$CURRENT" | cut -d. -f2)
    NEW="$MAJOR.$((MINOR + 1)).0"
    echo "New version: $NEW"
    git tag -a "$NEW" -m "Release $NEW"
    echo ""
    echo "Created tag $NEW"
    echo ""
    echo "Push with:"
    echo "  git push origin main --tags"

# Bump major version (0.1.2 → 1.0.0)
bump-major:
    #!/bin/sh
    set -e
    CURRENT="{{version}}"
    echo "Current version: $CURRENT"
    MAJOR=$(echo "$CURRENT" | cut -d. -f1)
    NEW="$((MAJOR + 1)).0.0"
    echo "New version: $NEW"
    git tag -a "$NEW" -m "Release $NEW"
    echo ""
    echo "Created tag $NEW"
    echo ""
    echo "Push with:"
    echo "  git push origin main --tags"

# Release: bump patch and push
release-patch: bump-patch
    git push origin main --tags

# Release: bump minor and push
release-minor: bump-minor
    git push origin main --tags

# Release: bump major and push
release-major: bump-major
    git push origin main --tags
