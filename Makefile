# SPDX-FileCopyrightText: 2026 Logan Lindquist Land
# SPDX-License-Identifier: FSL-1.1-MIT

# repjan Version Management & Build
#
# Usage:
#   make build        - Build binary for current platform
#   make test         - Run tests
#   make lint         - Run linters
#   make bump-patch   - Increment patch version (0.1.2 → 0.1.3)
#   make bump-minor   - Increment minor version (0.1.2 → 0.2.0)
#   make bump-major   - Increment major version (0.1.2 → 1.0.0)
#   make show-version - Show current version

# Get current version from git tags (defaults to 0.0.0 if no tags)
CURRENT_VERSION := $(shell git describe --tags --abbrev=0 2>/dev/null || echo "0.0.0")
VERSION_PARTS := $(subst ., ,$(CURRENT_VERSION))
MAJOR := $(word 1,$(VERSION_PARTS))
MINOR := $(word 2,$(VERSION_PARTS))
PATCH := $(word 3,$(VERSION_PARTS))

# Build variables
BINARY := repjan
BUILD_DIR := .
LDFLAGS := -ldflags="-s -w -X main.version=$(CURRENT_VERSION)"

.PHONY: build test lint clean bump-patch bump-minor bump-major show-version

# Build targets
build:
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY) ./cmd/repjan

test:
	go test -v -race ./...

lint:
	go vet ./...
	@if [ -n "$$(gofmt -l .)" ]; then \
		echo "Files not formatted:"; \
		gofmt -l .; \
		exit 1; \
	fi

clean:
	rm -f $(BUILD_DIR)/$(BINARY)

# Version bumping
bump-patch:
	@NEW_PATCH=$$(($(PATCH) + 1)); \
	NEW_VERSION="$(MAJOR).$(MINOR).$$NEW_PATCH"; \
	$(MAKE) tag-version VERSION=$$NEW_VERSION

bump-minor:
	@NEW_MINOR=$$(($(MINOR) + 1)); \
	NEW_VERSION="$(MAJOR).$$NEW_MINOR.0"; \
	$(MAKE) tag-version VERSION=$$NEW_VERSION

bump-major:
	@NEW_MAJOR=$$(($(MAJOR) + 1)); \
	NEW_VERSION="$$NEW_MAJOR.0.0"; \
	$(MAKE) tag-version VERSION=$$NEW_VERSION

# Internal target for tagging
tag-version:
	@if [ -z "$(VERSION)" ]; then \
		echo "Error: VERSION is required"; \
		exit 1; \
	fi
	@echo "Current version: $(CURRENT_VERSION)"
	@echo "New version:     $(VERSION)"
	@echo ""
	@git tag -a "$(VERSION)" -m "Release $(VERSION)"
	@echo "Created tag $(VERSION)"
	@echo ""
	@echo "Push with:"
	@echo "  git push origin main --tags"

# Show current version
show-version:
	@echo "Current version: $(CURRENT_VERSION)"
	@echo ""
	@echo "Recent tags:"
	@git tag --sort=-version:refname | head -5
