# Dev Tool

A custom build tool for the RackMonitor project that simplifies common development tasks including building, testing, and deploying the application.

## Overview

The `dev` tool is a Cobra-based CLI application that provides a consistent interface for project build operations. It supports both native builds using `go build` and cross-compilation using Docker containers.

## Installation

### Building the Dev Tool

The dev tool itself is built using the project's Makefile:

```bash
# Build for current platform
make build

# Build for all platforms
make build-all

# Build for specific platforms
make build-linux
make build-darwin
make build-windows
```

The binary will be created in the project root as `dev` (or `dev-{os}-{arch}` for cross-platform builds).

### Quick Start

```bash
# Build the dev tool
make build

# Use the dev tool
./dev --help
```

## Usage

### Global Flags

- `--debug`: Enable debug logging
- `--version`: Specify version for build operations (default: "latest")

### Commands

#### `build`

Build the rackmonitor application.

```bash
# Build for current platform (native build using go build)
./dev build

# Build for a specific platform
./dev build --os linux --arch amd64

# Build with a specific version
./dev build --version 1.0.0

# Cross-platform build without cache
./dev build --os linux --arch arm64 --no-cache
```

**Flags:**
- `--os`: Target operating system (default: current OS)
- `--arch`: Target architecture (default: current architecture)
- `--version`: Version to inject into the binary (default: "latest")
- `--no-cache`: Disable Docker cache when building (cross-compilation only)

**Build Behavior:**
- **Native builds** (same OS and architecture): Uses `go build` directly for fast compilation
- **Cross-compilation**: Uses Docker with the `mklimuk/gobuild:1.24-cross` image for cross-platform builds

**Output:**
- Native builds: `dist/rackmonitor`
- Cross-compilation: `./dev-{os}-{arch}` in the project root

#### `changelog`

Generate or update CHANGELOG.md from git commit history.

```bash
# Generate full changelog
./dev changelog

# Generate changelog for next version
./dev changelog --next v1.2.0

# Generate for specific tag
./dev changelog --tag v1.0.0

# Output to different file
./dev changelog --output CHANGES.md
```

**Flags:**
- `--next`: Next version tag (e.g., v1.2.0) to include in changelog
- `--output`: Output file path (default: CHANGELOG.md)
- `--tag`: Generate changelog for a specific tag

**Prerequisites:**
This command requires `git-chglog` to be installed. Run the setup script first:
```bash
./scripts/setup-changelog.sh
# or
make setup-changelog
```

**Commit Message Format:**
The changelog is generated from git commits following the Conventional Commits specification:
```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

**Supported Types:**
- `feat` - New features
- `fix` - Bug fixes
- `docs` - Documentation changes
- `refactor` - Code refactoring
- `test` - Test additions/changes
- `perf` - Performance improvements
- `build` - Build system changes
- `ci` - CI/CD changes
- `chore` - Maintenance tasks

**Example Commits:**
```bash
git commit -m "feat(sensor): add temperature threshold validation"
git commit -m "fix(api): correct response format for alerts endpoint"
git commit -m "docs: update README with build instructions"
```

## Architecture

### Build Strategy

The dev tool implements an intelligent build strategy:

1. **Native Builds**: When building for the same OS and architecture as the host, it uses `go build` directly
   - Fast compilation
   - No Docker required
   - Leverages local Go toolchain
   - Supports CGo (enabled by default)

2. **Cross-Compilation**: When building for a different platform, it uses Docker
   - Uses `mklimuk/gobuild:1.24-cross` image
   - Supports multiple target platforms
   - Consistent build environment
   - Optional cache control

### Version Injection

The dev tool automatically injects version information into the binary using Go build flags:

```go
-ldflags "-X {ConfigPackage}.Version={version}"
```

The version is injected into the `github.com/satsysoft/rackmonitor/pkg/config` package by default.

### Dependencies

The dev tool uses the following external libraries:
- **github.com/spf13/cobra**: CLI framework [[memory:4643114]]
- **github.com/charmbracelet/log**: Structured logging with colored output
- **github.com/gophertribe/devtool/build**: Build utilities (provides `GoBuild` and `Docker` functions)

## Examples

### Common Development Workflows

#### Generate changelog for release
```bash
# Generate changelog for next version
./dev changelog --next v1.2.0

# Or use interactive Makefile target
make changelog-next

# Review the generated CHANGELOG.md

# Commit the changelog
git add CHANGELOG.md
git commit -m "docs: update changelog for v1.2.0"

# Tag the release
git tag -a v1.2.0 -m "Release v1.2.0"
```

#### Build and run locally
```bash
# Build the dev tool
make build

# Build the application
./dev build

# Run the application
./dist/rackmonitor serve
```

#### Build for production deployment
```bash
# Build with version tag
./dev build --version 1.0.0

# Build for Linux server
./dev build --os linux --arch amd64 --version 1.0.0
```

#### Build for Raspberry Pi
```bash
# Raspberry Pi 4 (64-bit)
./dev build --os linux --arch arm64

# Raspberry Pi 3/Zero (32-bit)
./dev build --os linux --arch arm
```

#### Clean build (no cache)
```bash
# Useful when dependencies change
./dev build --os linux --arch amd64 --no-cache
```

## Makefile Integration

The project includes a Makefile that wraps common dev tool operations:

```bash
# Build the dev tool itself
make build

# Build for all platforms
make build-all

# Clean build artifacts
make clean

# Install dependencies
make deps

# Show help
make help
```

## Configuration

### Logging

The dev tool uses structured logging with color output powered by Charm Bracelet's log library:

- **Default level**: INFO
- **Debug mode**: Enable with `--debug` flag
- **Output format**: Human-readable with colors in development
- **Timestamp format**: DateTime format (YYYY-MM-DD HH:MM:SS)

### Docker Image

Cross-compilation uses the `mklimuk/gobuild:1.24-cross` Docker image, which includes:
- Go 1.24 toolchain
- Cross-compilation tools for multiple platforms
- CGo support for various architectures

## Troubleshooting

### "command not found: dev"

Make sure you've built the dev tool:
```bash
make build
./dev --help
```

### Docker permission errors

If you encounter permission errors during cross-compilation:
```bash
# Add your user to the docker group (Linux)
sudo usermod -aG docker $USER
# Log out and back in for changes to take effect
```

### Build failures

Enable debug logging to see detailed build information:
```bash
./dev --debug build --os linux --arch amd64
```

### Version not injected correctly

Verify the config package path matches your project structure. The dev tool injects the version into `github.com/satsysoft/rackmonitor/pkg/config`. If your package structure is different, you'll need to update the `ConfigPackage` field in the build command.

## Changelog Workflow

The dev tool integrates changelog generation based on git history:

1. **Setup** (one-time):
   ```bash
   ./scripts/setup-changelog.sh
   ```
   This installs `git-chglog` and sets up a git hook to validate commit messages.

2. **Commit with convention**:
   ```bash
   git commit -m "feat(sensor): add humidity sensor support"
   ```
   The git hook will validate your commit message format.

3. **Generate changelog**:
   ```bash
   # For a new release
   ./dev changelog --next v1.2.0
   
   # Or just update from history
   ./dev changelog
   ```

4. **Validate commits** before pushing:
   ```bash
   make validate-commits
   ```

## Future Enhancements

Planned additions to the dev tool:

- [ ] `test` command - Run tests with coverage reporting
- [ ] `lint` command - Run linters and formatters
- [ ] `deploy` command - Deploy to remote servers
- [ ] `watch` command - Auto-rebuild on file changes
- [ ] `run` command - Build and run in one step
- [ ] `release` command - Automated release workflow (build + changelog + tag)
- [ ] Configuration file support (`.devtool.yaml`)
- [ ] Plugin system for custom build steps

## Contributing

When adding new commands to the dev tool:

1. Create a new file in `cmd/dev/cmd/` (e.g., `test.go`)
2. Follow the existing pattern using Cobra
3. Use `cmd.Flags()` to retrieve flag values (not `args` by index) [[memory:4643114]]
4. Add structured logging with `slog`
5. Register the command in `main.go` using `rootCmd.AddCommand()`
6. Update this README with the new command documentation

## License

This tool is part of the RackMonitor project and shares the same license.

