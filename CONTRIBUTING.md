# Contributing to Go-CacheX

Thank you for your interest in contributing to Go-CacheX! This document provides guidelines and information for contributors.

## Code of Conduct

This project and everyone participating in it is governed by our Code of Conduct. By participating, you are expected to uphold this code.

## How Can I Contribute?

### Reporting Bugs

- Use the GitHub issue tracker
- Include a clear and descriptive title
- Provide detailed steps to reproduce the bug
- Include system information (OS, Go version, etc.)
- Include error messages and stack traces

### Suggesting Enhancements

- Use the GitHub issue tracker with the "enhancement" label
- Describe the feature and its benefits
- Provide use cases and examples
- Consider implementation complexity

### Pull Requests

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Add tests for new functionality
5. Ensure all tests pass (`go test -race ./...`)
6. Run linting (`golangci-lint run`)
7. Commit your changes (`git commit -m 'Add amazing feature'`)
8. Push to the branch (`git push origin feature/amazing-feature`)
9. Open a Pull Request

## Development Setup

### Prerequisites

- Go 1.24 or later
- Redis (for integration tests)
- Docker (for running examples)

### Local Development

1. Clone the repository:
   ```bash
   git clone https://github.com/SeaSBee/go-cachex.git
   cd go-cachex
   ```

2. Install dependencies:
   ```bash
   go mod download
   ```

3. Run tests:
   ```bash
   go test -race ./...
   ```

4. Run linting:
   ```bash
   golangci-lint run
   ```

5. Run integration tests:
   ```bash
   go test -tags=integration ./tests/integration/...
   ```

### Running Examples

1. Start Redis:
   ```bash
   docker run -d -p 6379:6379 redis:7-alpine
   ```

2. Run the basic example:
   ```bash
   go run example/basic/main.go
   ```

3. Or use Docker Compose:
   ```bash
   docker-compose -f deployments/docker-compose.yml up -d
   ```

## Code Style

### Go Code

- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use `gofmt` for formatting
- Follow the project's linting rules
- Write clear, descriptive commit messages

### Testing

- Write unit tests for all new functionality
- Use table-driven tests where appropriate
- Test error conditions and edge cases
- Aim for high test coverage
- Use `testify` for assertions

### Documentation

- Document all exported functions and types
- Include examples in documentation
- Update README.md for new features
- Add comments for complex logic

## Commit Message Format

Use conventional commit format:

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes
- `refactor`: Code refactoring
- `test`: Test changes
- `chore`: Build/tooling changes

Examples:
```
feat(cache): add ReadThrough pattern support
fix(redis): handle connection timeouts properly
docs(readme): add quick start guide
```

## Release Process

1. Update version in relevant files
2. Update CHANGELOG.md
3. Create a release tag
4. Push to GitHub
5. GitHub Actions will build and test

## Questions?

If you have questions about contributing, please:

1. Check existing issues and discussions
2. Open a new issue with the "question" label
3. Join our community discussions

Thank you for contributing to Go-CacheX!
