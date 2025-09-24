# Contributing to Gor

We love your input! We want to make contributing to Gor as easy and transparent as possible, whether it's:

- Reporting a bug
- Discussing the current state of the code
- Submitting a fix
- Proposing new features
- Becoming a maintainer

## Development Process

We use GitHub to host code, to track issues and feature requests, as well as accept pull requests.

### Getting Started

1. **Fork the repository** on GitHub
2. **Clone your fork** locally:
   ```bash
   git clone https://github.com/yourusername/gor.git
   cd gor
   ```
3. **Set up your development environment**:
   ```bash
   # Install Go 1.21+ if not already installed
   # Install development tools
   make tools

   # Run tests to ensure everything works
   make test
   ```

### Development Workflow

1. **Create a feature branch** from `main`:
   ```bash
   git checkout -b feature/amazing-feature
   ```

2. **Make your changes** following our coding standards:
   - Write tests for new functionality
   - Update documentation as needed
   - Follow Go conventions and best practices
   - Ensure all tests pass: `make test`
   - Run quality checks: `make ci`

3. **Commit your changes** using conventional commits:
   ```bash
   git commit -m "feat: add amazing new feature"
   ```

4. **Push to your fork**:
   ```bash
   git push origin feature/amazing-feature
   ```

5. **Create a Pull Request** on GitHub

## Coding Standards

### Go Code Style

- Follow standard Go formatting: `gofmt -w .`
- Use `go vet` to check for common mistakes
- Follow effective Go practices
- Write self-documenting code with clear variable names
- Add comments for exported functions and complex logic

### Security Standards

- **Security scanning**: All code is scanned with gosec
- **Build tags**: Use `//go:build debug` for debug-only features (e.g., pprof)
- **Integer safety**: Check bounds before type conversions to prevent overflow
- **HTTP security**: Always set timeouts on HTTP servers
- **Template safety**: Document when bypassing HTML escaping (#nosec G203)
- **Cryptography**: Use SHA256+ for hashing, never MD5 for security
- **Type safety**: Use consistent context key types across packages

See `.gosec.toml` for configuration and accepted security exclusions.

### Testing Requirements

- **Write tests** for all new functionality
- **Maintain test coverage** above 75% (aim for 80%+)
- Use **table-driven tests** for comprehensive coverage
- Test both happy path and error cases
- Use `t.TempDir()` for file system tests

Example test structure:
```go
func TestFeature(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
        wantErr  bool
    }{
        {
            name:     "valid input",
            input:    "test",
            expected: "TEST",
            wantErr:  false,
        },
        // More test cases...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := Feature(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("Feature() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if result != tt.expected {
                t.Errorf("Feature() = %v, want %v", result, tt.expected)
            }
        })
    }
}
```

### Commit Message Guidelines

We use [Conventional Commits](https://www.conventionalcommits.org/):

- `feat:` - New features
- `fix:` - Bug fixes
- `docs:` - Documentation updates
- `test:` - Adding tests
- `refactor:` - Code refactoring
- `perf:` - Performance improvements
- `chore:` - Maintenance tasks

Examples:
```
feat: add database connection pooling
fix: resolve memory leak in cache layer
docs: update API documentation for authentication
test: add integration tests for ORM package
```

## Pull Request Process

### Before Submitting

1. **Ensure all tests pass**: `make test`
2. **Run quality checks**: `make ci`
3. **Update documentation** if needed
4. **Add tests** for new functionality
5. **Check test coverage**: `make test-coverage`

### PR Requirements

- **Clear description** of what the PR does
- **Reference related issues** using "Fixes #123" or "Closes #123"
- **Include screenshots** for UI changes
- **Update CHANGELOG.md** for user-facing changes
- **Ensure CI passes** before requesting review

### PR Template

When creating a PR, please include:

```markdown
## Description
Brief description of changes

## Type of Change
- [ ] Bug fix (non-breaking change)
- [ ] New feature (non-breaking change)
- [ ] Breaking change
- [ ] Documentation update

## Testing
- [ ] Tests added/updated
- [ ] All tests pass
- [ ] Manual testing completed

## Checklist
- [ ] Code follows style guidelines
- [ ] Self-review completed
- [ ] Documentation updated
- [ ] CHANGELOG.md updated (if needed)
```

## Issue Reporting

### Bug Reports

Use the bug report template and include:
- **Gor version** you're using
- **Go version** and operating system
- **Minimal reproduction** example
- **Expected vs actual behavior**
- **Stack trace** if applicable

### Feature Requests

Use the feature request template and include:
- **Problem description** the feature would solve
- **Proposed solution** with examples
- **Alternative solutions** considered
- **Additional context** or screenshots

## Development Setup

### Prerequisites

- Go 1.21 or higher
- Git
- Make (for build automation)

### Local Development

```bash
# Clone and set up
git clone https://github.com/cuemby/gor.git
cd gor

# Install development tools
make tools

# Run tests
make test

# Start development server
make dev

# Run examples
make run-webapp
make run-auth
```

### Project Structure

```
gor/
├── pkg/gor/           # Core framework interfaces
├── internal/          # Implementation packages
├── cmd/gor/          # CLI application
├── examples/         # Example applications
├── docs/            # Documentation
├── .github/         # GitHub workflows and templates
└── coverage_output/ # Test coverage reports
```

### Make Targets

- `make build` - Build the CLI binary
- `make test` - Run all tests
- `make test-coverage` - Generate coverage report
- `make fmt` - Format all code
- `make vet` - Run go vet
- `make ci` - Full CI pipeline
- `make clean` - Clean build artifacts

## Architecture Guidelines

### Adding New Features

1. **Define interfaces** in `pkg/gor/`
2. **Implement** in appropriate `internal/` package
3. **Add tests** with good coverage
4. **Create examples** in `examples/`
5. **Update documentation**

### Rails Inspiration

Gor follows Rails conventions where appropriate:
- **Convention over configuration**
- **RESTful routing patterns**
- **MVC architecture**
- **ActiveRecord-style ORM**
- **Middleware patterns**

### Performance Considerations

- **Minimize allocations** in hot paths
- **Use sync.Pool** for reusable objects
- **Benchmark** performance-critical code
- **Profile** memory usage
- **Consider goroutine lifecycle**

## Community

### Getting Help

- **GitHub Discussions** for general questions
- **GitHub Issues** for bugs and feature requests
- **Discord/Slack** (link in README) for real-time chat

### Code of Conduct

This project adheres to the [Contributor Covenant Code of Conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code.

## Recognition

Contributors will be recognized in:
- Release notes for significant contributions
- README contributors section
- GitHub contributor graphs

## License

By contributing, you agree that your contributions will be licensed under the MIT License. See [LICENSE](LICENSE) for details.

---

Thank you for contributing to Gor! 🚀