# Contributing to Scanfrog 🐸

Thank you for your interest in contributing to Scanfrog! This guide will help you get started with development and ensure your contributions meet our quality standards.

## Table of Contents
- [Getting Started](#getting-started)
- [Development Workflow](#development-workflow)
- [Code Standards](#code-standards)
- [Testing Guidelines](#testing-guidelines)
- [Submitting Changes](#submitting-changes)
- [Release Process](#release-process)

## Getting Started

### Prerequisites

- Go 1.24 or later
- Git
- Make
- Grype (for live container scanning) - [Installation guide](https://github.com/anchore/grype#installation)

### First-Time Setup

1. Fork and clone the repository:
   ```bash
   git clone https://github.com/YOUR_USERNAME/scanfrog.git
   cd scanfrog
   ```

2. Install development tools:
   ```bash
   make setup
   ```
   This installs:
   - golangci-lint (code quality)
   - gosec (security scanner)
   - go-licenses (license checker)
   - govulncheck (vulnerability scanner)
   - goreleaser (release automation)

3. Verify your setup:
   ```bash
   make all
   ```
   This runs all checks that CI will run.

## Development Workflow

### Before Making Changes

1. Check existing tests pass:
   ```bash
   make test
   ```

2. Create a feature branch:
   ```bash
   git checkout -b feature/your-feature-name
   ```

### While Developing

1. **Make your changes** - Follow existing code patterns and conventions

2. **Run checks frequently**:
   ```bash
   make fmt         # Format code
   make lint        # Check code quality
   make test        # Run tests
   ```

3. **Test the game**:
   ```bash
   # Build and test with sample data
   make build
   make smoke-test
   
   # Or test manually
   ./scanfrog --json testdata/sample-vulns.json
   ```

### Before Committing

Run all pre-commit checks:
```bash
make pre-commit
```

This ensures your code is properly formatted, passes linting, and tests pass.

### Before Opening a PR

Run the full check suite (same as CI):
```bash
make all
```

## Code Standards

### General Guidelines

- **Simplicity over cleverness**: Write clear, maintainable code
- **Match existing style**: Follow patterns in surrounding code
- **Document complex logic**: Add comments for non-obvious code
- **No magic numbers**: Use named constants, especially for game mechanics

### Go-Specific Standards

- All code must be `go fmt`'d
- Follow [Effective Go](https://golang.org/doc/effective_go.html) guidelines
- Export only what's necessary
- Prefer table-driven tests

### Game-Specific Considerations

- **Terminal compatibility**: Test on different terminal sizes (minimum 80x24)
- **Color schemes**: Ensure readability in both light and dark terminals
- **Performance**: Maintain 60 FPS with 50+ obstacles
- **Emoji handling**: Remember emojis are double-width characters

## Testing Guidelines

### Unit Tests

All new code should include tests. We currently require 40% coverage but aim to increase this over time.

```bash
# Run tests
make test

# Check coverage
make test-coverage-report
```

### Writing Game Tests

When testing game mechanics:

```go
func TestCollisionDetection(t *testing.T) {
    model := NewModel(vulns, "test", 80, 24)
    model.state = StatePlaying
    
    // Set up specific game state
    model.frogX = 10
    model.frogY = 10
    model.obstacles = []Obstacle{{x: 10, y: 10, width: 1}}
    
    // Test the behavior
    if !model.checkCollision() {
        t.Error("expected collision")
    }
}
```

### Terminal Interaction Tests

For testing user interactions (future enhancement):

```go
// +build interactive

func TestPlayerCompletesWave(t *testing.T) {
    // Use Bubble Tea test helpers
    // Simulate key presses
    // Verify game state changes
}
```

### Integration Tests

Test with real vulnerability data:

```bash
make test-integration
```

## Submitting Changes

### Pull Request Process

1. **Update your fork**:
   ```bash
   git remote add upstream https://github.com/luhring/scanfrog.git
   git fetch upstream
   git rebase upstream/main
   ```

2. **Push your branch**:
   ```bash
   git push origin feature/your-feature-name
   ```

3. **Open a Pull Request** with:
   - Clear title describing the change
   - Description of what and why
   - Any relevant issue numbers
   - Screenshots/recordings for UI changes

### PR Requirements

All PRs must:
- ✅ Pass all CI checks
- ✅ Include tests for new functionality
- ✅ Maintain or improve code coverage
- ✅ Follow code standards
- ✅ Include meaningful commit messages

### Commit Messages

Follow conventional commits:
```
feat: add power-up system for temporary invincibility
fix: correct collision detection for wide obstacles
docs: update README with new command flags
test: add integration tests for wave progression
```

## Release Process

Releases are automated using GoReleaser when a tag is pushed:

```bash
# Create and push a tag (maintainers only)
git tag -s v1.2.3 -m "Release v1.2.3"
git push origin v1.2.3
```

This triggers:
1. Multi-platform builds
2. GitHub release creation
3. Changelog generation
4. Checksum creation

## Getting Help

- Check existing issues and discussions
- Ask questions in pull requests
- Review the [architecture guide](CLAUDE.md)

## Security

If you discover a security vulnerability, please report it through the GitHub repository's [security UI](https://github.com/luhring/scanfrog/security/advisories/new), instead of opening an issue.

---

Thank you for contributing to making container security more fun! 🎮🐸
