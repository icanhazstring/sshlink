# sshlink Makefile

# Variables
BINARY_NAME=sshlink
BUILD_DIR=build
MAIN_FILES=*.go

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Build flags
LDFLAGS=-ldflags "-s -w"

.PHONY: all build test clean deps help

# Default target - show help
all: help

# Build the binary
build: deps
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_FILES)
	@echo "✅ Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Run tests
test: deps
	@echo "Running tests..."
	$(GOTEST) -v ./...
	@echo "✅ Tests complete"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	@rm -rf $(BUILD_DIR)
	@echo "✅ Clean complete"

# Download dependencies and initialize module
deps:
	@echo "Setting up Go module..."
	@if [ ! -f go.mod ]; then \
		echo "Initializing Go module..."; \
		$(GOMOD) init github.com/icanhazstring/sshlink; \
	fi
	@echo "Downloading dependencies and tidying..."
	$(GOMOD) download
	$(GOMOD) tidy
	@echo "✅ Dependencies ready"

# Install the binary to $GOPATH/bin
install: build
	@echo "Installing $(BINARY_NAME)..."
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/$(BINARY_NAME)
	@echo "✅ Installed to $(GOPATH)/bin/$(BINARY_NAME)"

# Run the application
run:
	@echo "Running $(BINARY_NAME)..."
	$(GOCMD) run $(MAIN_FILES)

# Build for multiple platforms
build-all: deps
	@echo "Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)

	# macOS
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_FILES)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_FILES)

	# Linux
	#GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_FILES)
	#GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(MAIN_FILES)

	# Windows
	#GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_FILES)

	@echo "✅ Multi-platform build complete"

# Show help
help:
	@echo "Available targets:"
	@echo "  build      - Build the binary to $(BUILD_DIR)/$(BINARY_NAME)"
	@echo "  test       - Run tests"
	@echo "  clean      - Clean build artifacts"
	@echo "  deps       - Download and tidy dependencies"
	@echo "  install    - Install binary to \$$GOPATH/bin"
	@echo "  run        - Run the application directly"
	@echo "  build-all  - Build for multiple platforms"
	@echo "  help       - Show this help message"
