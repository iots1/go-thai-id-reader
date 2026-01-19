# Go Thai ID API - Build Makefile
# Supports: Windows (amd64), macOS (Apple Silicon & Intel)
# Note: This project requires CGO due to scard library dependency

APP_NAME := go-thai-id-api
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_DIR := dist
LDFLAGS := -s -w -X main.Version=$(VERSION)

# Detect current OS and architecture
CURRENT_OS := $(shell uname -s | tr '[:upper:]' '[:lower:]')
CURRENT_ARCH := $(shell uname -m)

# Default target
.PHONY: all
all: clean build

# Clean build directory
.PHONY: clean
clean:
	rm -rf $(BUILD_DIR)
	mkdir -p $(BUILD_DIR)

# Build for current platform only (CGO required)
.PHONY: build
build:
	@echo "Building for current platform ($(CURRENT_OS)/$(CURRENT_ARCH))..."
ifeq ($(CURRENT_OS),darwin)
ifeq ($(CURRENT_ARCH),arm64)
	CGO_ENABLED=1 go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-darwin-arm64 .
	@echo "✓ macOS ARM64 build complete: $(BUILD_DIR)/$(APP_NAME)-darwin-arm64"
else
	CGO_ENABLED=1 go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-darwin-amd64 .
	@echo "✓ macOS Intel build complete: $(BUILD_DIR)/$(APP_NAME)-darwin-amd64"
endif
else ifeq ($(CURRENT_OS),linux)
	CGO_ENABLED=1 go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-linux-amd64 .
	@echo "✓ Linux build complete: $(BUILD_DIR)/$(APP_NAME)-linux-amd64"
else
	CGO_ENABLED=1 go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-windows-amd64.exe .
	@echo "✓ Windows build complete: $(BUILD_DIR)/$(APP_NAME)-windows-amd64.exe"
endif

# macOS Apple Silicon (M1/M2/M3) - must run on Apple Silicon Mac
.PHONY: build-darwin-arm64
build-darwin-arm64:
	@echo "Building for macOS (Apple Silicon)..."
	@echo "Note: Must be run on Apple Silicon Mac (CGO required)"
	CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-darwin-arm64 .
	@echo "✓ macOS ARM64 build complete: $(BUILD_DIR)/$(APP_NAME)-darwin-arm64"

# macOS Intel - must run on macOS
.PHONY: build-darwin-amd64
build-darwin-amd64:
	@echo "Building for macOS (Intel)..."
	@echo "Note: Must be run on macOS (CGO required)"
	CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-darwin-amd64 .
	@echo "✓ macOS Intel build complete: $(BUILD_DIR)/$(APP_NAME)-darwin-amd64"

# Windows - must run on Windows
.PHONY: build-windows
build-windows:
	@echo "Building for Windows (amd64)..."
	@echo "Note: Must be run on Windows (CGO required)"
	CGO_ENABLED=1 GOOS=windows GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-windows-amd64.exe .
	@echo "✓ Windows build complete: $(BUILD_DIR)/$(APP_NAME)-windows-amd64.exe"

# Create release archive for current platform
.PHONY: package
package: build
	@echo "Creating release package..."
ifeq ($(CURRENT_OS),darwin)
ifeq ($(CURRENT_ARCH),arm64)
	cd $(BUILD_DIR) && tar -czvf $(APP_NAME)-darwin-arm64.tar.gz $(APP_NAME)-darwin-arm64
	@echo "✓ Package created: $(BUILD_DIR)/$(APP_NAME)-darwin-arm64.tar.gz"
else
	cd $(BUILD_DIR) && tar -czvf $(APP_NAME)-darwin-amd64.tar.gz $(APP_NAME)-darwin-amd64
	@echo "✓ Package created: $(BUILD_DIR)/$(APP_NAME)-darwin-amd64.tar.gz"
endif
else ifeq ($(CURRENT_OS),linux)
	cd $(BUILD_DIR) && tar -czvf $(APP_NAME)-linux-amd64.tar.gz $(APP_NAME)-linux-amd64
	@echo "✓ Package created: $(BUILD_DIR)/$(APP_NAME)-linux-amd64.tar.gz"
else
	cd $(BUILD_DIR) && zip $(APP_NAME)-windows-amd64.zip $(APP_NAME)-windows-amd64.exe
	@echo "✓ Package created: $(BUILD_DIR)/$(APP_NAME)-windows-amd64.zip"
endif

# Run locally
.PHONY: run
run:
	go run .

# Show help
.PHONY: help
help:
	@echo "Go Thai ID API - Build Commands"
	@echo ""
	@echo "Note: This project requires CGO. Cross-compilation is not supported."
	@echo "      Each platform must be built on its native OS."
	@echo ""
	@echo "Usage:"
	@echo "  make build            Build for current platform"
	@echo "  make package          Build and create archive for current platform"
	@echo "  make clean            Clean build directory"
	@echo "  make run              Run locally"
	@echo "  make help             Show this help"
	@echo ""
	@echo "Platform-specific (run on native OS only):"
	@echo "  make build-darwin-arm64  Build for macOS Apple Silicon"
	@echo "  make build-darwin-amd64  Build for macOS Intel"
	@echo "  make build-windows       Build for Windows (64-bit)"
