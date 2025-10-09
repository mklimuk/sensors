# Makefile for building the custom build tool
# This replaces the mage-based build system

BINARY_NAME=dev
SOURCE_DIR=cmd/dev

# Build for multiple platforms
build-all: build build-linux build-darwin build-windows

# Build for current platform
build:
	@echo "Building $(BINARY_NAME) for current platform..."
	go build -o $(BINARY_NAME) ./$(SOURCE_DIR)

build-linux:
	@echo "Building $(BINARY_NAME) for Linux..."
	GOOS=linux GOARCH=amd64 go build -o $(BINARY_NAME)-linux-amd64 ./$(SOURCE_DIR)

build-darwin:
	@echo "Building $(BINARY_NAME) for macOS..."
	GOOS=darwin GOARCH=amd64 go build -o $(BINARY_NAME)-darwin-amd64 ./$(SOURCE_DIR)
	GOOS=darwin GOARCH=arm64 go build -o $(BINARY_NAME)-darwin-arm64 ./$(SOURCE_DIR)

build-windows:
	@echo "Building $(BINARY_NAME) for Windows..."
	GOOS=windows GOARCH=amd64 go build -o $(BINARY_NAME)-windows-amd64.exe ./$(SOURCE_DIR)

# Clean build artifacts
clean:
	@echo "Cleaning $(BINARY_NAME) artifacts..."
	rm $(BINARY_NAME)*

# Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod tidy
	go mod download

# Generate changelog
changelog:
	@echo "Generating changelog..."
	./dev changelog

# Generate changelog for next version
changelog-next:
	@echo "Generating changelog for next version..."
	@read -p "Next version (e.g., v1.2.0): " version; \
	./dev changelog --next $$version

# Validate commit messages follow conventional commits
validate-commits:
	@echo "Validating commit messages..."
	@git log --oneline origin/main..HEAD 2>/dev/null | while read line; do \
		if ! echo "$$line" | grep -qE "^[a-f0-9]+ (feat|fix|docs|refactor|test|perf|build|ci|chore)(\(.+\))?:"; then \
			echo "❌ Invalid commit: $$line"; \
			echo "   Commits must follow format: <type>[scope]: <description>"; \
			exit 1; \
		fi; \
	done || true
	@echo "✅ All commits follow conventional format"

# Setup changelog tooling (git-chglog)
setup-changelog:
	@echo "Setting up changelog tooling..."
	@./scripts/setup-changelog.sh

# Debian packaging targets
deb-all:
	@echo "Building Debian packages for all architectures and versions..."
	@./scripts/build-deb-docker.sh 1.0.0 all all

deb-amd64:
	@echo "Building Debian package for amd64 (bookworm)..."
	@./scripts/build-deb-docker.sh 1.0.0 amd64 bookworm

deb-arm64:
	@echo "Building Debian package for arm64 (bookworm)..."
	@./scripts/build-deb-docker.sh 1.0.0 arm64 bookworm

deb-armhf:
	@echo "Building Debian package for armhf (bookworm)..."
	@./scripts/build-deb-docker.sh 1.0.0 armhf bookworm

deb-custom:
	@echo "Building custom Debian package..."
	@read -p "Version (e.g., 1.0.0): " version; \
	read -p "Architecture (amd64/arm64/armhf): " arch; \
	read -p "Debian version (buster/bookworm/trixie): " debian_ver; \
	./scripts/build-deb-docker.sh $$version $$arch $$debian_ver

deb-clean:
	@echo "Cleaning Debian build artifacts..."
	@rm -rf build/deb dist/deb
	@echo "✅ Debian build artifacts cleaned"

# Show help
help:
	@echo "Available targets:"
	@echo ""
	@echo "Build Tool:"
	@echo "  build            - Build dev tool for current platform"
	@echo "  build-all        - Build dev tool for all platforms"
	@echo "  build-linux      - Build dev tool for Linux"
	@echo "  build-darwin     - Build dev tool for macOS"
	@echo "  build-windows    - Build dev tool for Windows"
	@echo "  clean            - Clean build artifacts"
	@echo "  deps             - Install dependencies"
	@echo ""
	@echo "Debian Packaging:"
	@echo "  deb-all          - Build .deb packages for all architectures and Debian versions"
	@echo "  deb-amd64        - Build .deb for amd64 (Debian Bookworm)"
	@echo "  deb-arm64        - Build .deb for arm64 (Debian Bookworm)"
	@echo "  deb-armhf        - Build .deb for armhf (Debian Bookworm)"
	@echo "  deb-custom       - Build .deb with custom version/arch/debian version"
	@echo "  deb-clean        - Clean Debian build artifacts"
	@echo ""
	@echo "Documentation:"
	@echo "  changelog        - Generate changelog from git history"
	@echo "  changelog-next   - Generate changelog for next version"
	@echo "  validate-commits - Validate commit messages"
	@echo "  setup-changelog  - Setup changelog tooling"
	@echo "  help             - Show this help"

.PHONY: build build-all build-linux build-darwin build-windows clean deps changelog changelog-next validate-commits setup-changelog deb-all deb-amd64 deb-arm64 deb-armhf deb-custom deb-clean help