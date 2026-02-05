# Agent Instructions

This document provides guidelines for AI agents and developers working on this project.

## Code Quality Requirements

### Before Every Commit
All changes **must** pass these checks before committing:

```bash
# Format code
go fmt ./...

# Run linter
golangci-lint run

# Run tests
go test -v ./...
```

### Goreleaser Validation
Whenever `.goreleaser.yaml` is created or modified:

```bash
goreleaser check
```

This **must** pass before committing any goreleaser configuration changes.

## Commit Guidelines

### Semantic Commits
Use [Conventional Commits](https://www.conventionalcommits.org/) format:

- `feat:` - New feature
- `fix:` - Bug fix
- `docs:` - Documentation changes
- `test:` - Adding or updating tests
- `ci:` - CI/CD changes
- `refactor:` - Code refactoring
- `chore:` - Maintenance tasks

Examples:
```
feat: add JWT header decoding
fix: handle missing Authorization header gracefully
test: add e2e tests for echo endpoint
ci: add release workflow for tag pushes
docs: update README with usage examples
```

### Small Reviewable Chunks
- Each commit should represent a single logical change
- Keep commits focused and atomic
- Separate refactoring from feature changes
- Tests should be in the same commit as the code they test

## Testing Requirements

### Unit Tests
- Use Go's built-in testing package
- Place tests in `*_test.go` files alongside source

### E2E / Smoke Tests
- Located in `tests/` directory
- Use `curl` and `jq` for HTTP testing
- Scripts should be executable and return non-zero on failure

Example smoke test pattern:
```bash
#!/bin/bash
set -euo pipefail

# Start server in background
./http-echo &
PID=$!
trap "kill $PID 2>/dev/null" EXIT
sleep 1

# Test echo endpoint
RESPONSE=$(curl -s http://localhost:8080/test)
echo "$RESPONSE" | jq -e '.path == "/test"' > /dev/null

echo "Smoke tests passed"
```

## CI/CD

### CI Workflow
Runs on every push and pull request:
- `go fmt` check (fail if not formatted)
- `golangci-lint run`
- `go test -v ./...`
- `goreleaser check`

### Release Workflow
Triggered on tags matching `v*`:
- Runs full CI checks
- Executes `goreleaser release`
- Publishes container images via ko
