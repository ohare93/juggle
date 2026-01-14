# Contributing to Juggle

Thank you for your interest in contributing to Juggle! This document provides guidelines and instructions for contributing.

## Getting Started

1. **Fork the repository** on GitHub
2. **Clone your fork** locally:
   ```bash
   git clone https://github.com/YOUR_USERNAME/juggle.git
   cd juggle
   ```
3. **Create a branch** for your changes:
   ```bash
   git checkout -b feature/your-feature-name
   ```

## Development Setup

Juggle uses [devbox](https://www.jetify.com/devbox) to manage development dependencies.

1. **Install devbox** (if not already installed):
   ```bash
   curl -fsSL https://get.jetify.com/devbox | bash
   ```

2. **Enter the devbox shell**:
   ```bash
   devbox shell
   ```
   This sets up Go and all required tooling automatically.

3. **Build the project**:
   ```bash
   devbox run build
   # or: go build -o juggle ./cmd/juggle
   ```

4. **Run tests**:
   ```bash
   devbox run test-quiet   # Quick pass/fail summary
   devbox run test-all     # Full verbose output
   ```

## Pull Request Process

1. **Ensure tests pass** before submitting:
   ```bash
   devbox run test-quiet
   ```

2. **Keep changes focused** - one feature or fix per PR

3. **Write clear commit messages** describing what changed and why

4. **Update documentation** if your changes affect user-facing behavior

5. **Submit the PR** against the `main` branch

### What Happens After Submission

- GitHub Actions automatically runs tests on Linux, Windows, and macOS
- A maintainer will review your PR
- Address any feedback and push updates to your branch
- Once approved, your PR will be merged

## Code Style

- Follow existing patterns in the codebase
- Use `go fmt` to format code
- Keep changes minimal and focused
- Add tests for new functionality

## Questions?

Open an issue on GitHub if you have questions or need help getting started.
