# Future Improvements

Lightweight tracking of planned improvements and technical debt.

## Test Coverage

- [ ] `internal/cmd/db.go` - Add integration tests for db migrate, status, reset commands
- [ ] `internal/cmd/sync.go` - Add integration tests for sync command
- [ ] `internal/cmd/root.go` - Add integration tests for main TUI command initialization
- [ ] `internal/sync` - Add tests for Syncer lifecycle (Start, Stop, background sync)

## Features Under Development

### Fabric AI Integration

The Fabric AI integration (`--fabric` flag) is experimental and may change:

- [ ] Stabilize Fabric analysis API
- [ ] Add comprehensive tests for analyze package Fabric integration
- [ ] Document Fabric setup and configuration
- [ ] Consider making Fabric analysis async/background
- [ ] Add caching for Fabric analysis results

## Code Quality

- [ ] Refactor cmd package to use dependency injection for better testability
- [ ] Extract common database setup logic from commands
- [ ] Add golangci-lint to CI pipeline
- [ ] Consider adding benchmarks for hot paths (filtering, sorting)

## Documentation

- [ ] Add usage examples to README
- [ ] Document keyboard shortcuts
- [ ] Add architecture overview diagram
