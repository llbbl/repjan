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
