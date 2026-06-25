APP_NAME := secrets
CMD_PATH := ./cmd/secrets
BIN_DIR := ./bin

# Version info
VERSION_PKG := github.com/bdryanovski/secrets/internal/version
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
YEAR := $(shell date -u '+%Y')
MONTH := $(shell date -u '+%-m')

# Read RC from file, default to 0
RC_FILE := .rc
RC := $(shell cat $(RC_FILE) 2>/dev/null || echo "0")
VERSION := $(YEAR).$(MONTH).$(RC)-rc$(GIT_COMMIT)

LDFLAGS := -X $(VERSION_PKG).GitCommit=$(GIT_COMMIT) \
           -X $(VERSION_PKG).BuildDate=$(BUILD_DATE) \
           -X $(VERSION_PKG).Version=$(VERSION)

# CGO is required for SQLCipher
CGO_ENABLED := 1

.PHONY: all build clean run build-linux build-darwin test lint fmt version bump-rc

all: build

## build: Build the binary for the current platform
build:
	@echo "Building $(APP_NAME) $(VERSION)..."
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=$(CGO_ENABLED) go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(APP_NAME) $(CMD_PATH)
	@echo "Done: $(BIN_DIR)/$(APP_NAME)"

## build-linux: Cross-compile for Linux amd64
build-linux:
	@echo "Building $(APP_NAME) for Linux amd64..."
	@mkdir -p $(BIN_DIR)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=$(CGO_ENABLED) go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(APP_NAME)-linux-amd64 $(CMD_PATH)
	@echo "Done: $(BIN_DIR)/$(APP_NAME)-linux-amd64"

## build-darwin: Cross-compile for macOS (amd64 + arm64)
build-darwin:
	@echo "Building $(APP_NAME) for macOS amd64..."
	@mkdir -p $(BIN_DIR)
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=$(CGO_ENABLED) go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(APP_NAME)-darwin-amd64 $(CMD_PATH)
	@echo "Done: $(BIN_DIR)/$(APP_NAME)-darwin-amd64"
	@echo "Building $(APP_NAME) for macOS arm64..."
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=$(CGO_ENABLED) go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(APP_NAME)-darwin-arm64 $(CMD_PATH)
	@echo "Done: $(BIN_DIR)/$(APP_NAME)-darwin-arm64"

## clean: Remove build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BIN_DIR)
	@echo "Done."

## run: Build and run the application
run: build
	@$(BIN_DIR)/$(APP_NAME)

## test: Run all tests
test:
	CGO_ENABLED=$(CGO_ENABLED) go test ./... -v

## lint: Run go vet
lint:
	go vet ./...

## fmt: Format all Go files
fmt:
	gofmt -s -w .

## version: Print the current version
version:
	@echo "$(VERSION)"

## bump-rc: Increment the RC number
bump-rc:
	@echo $$(( $(RC) + 1 )) > $(RC_FILE)
	@echo "RC bumped to $$(cat $(RC_FILE))"

## help: Show this help
help:
	@echo "Available targets:"
	@grep -E '^## ' Makefile | sed 's/## /  /'
