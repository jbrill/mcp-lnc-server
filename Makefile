.DEFAULT_GOAL := build

PKG := github.com/jbrill/mcp-lnc-server

GOTEST := GO111MODULE=on go test -v

GO_BIN := ${GOPATH}/bin
GOBUILD := GO111MODULE=on go build -v
GOINSTALL := GO111MODULE=on go install -v
GOMOD := GO111MODULE=on go mod

COMMIT := $(shell git describe --abbrev=40 --always --dirty)
LDFLAGS := -ldflags "-X $(PKG).Commit=$(COMMIT)"
DEV_TAGS = dev

GOFILES_NOVENDOR = $(shell find . -type f -name '*.go' -not -path "./vendor/*")
GOLIST := go list $(PKG)/... | grep -v '/vendor/'

TEST_FLAGS = -test.timeout=20m

UNIT := $(GOLIST) | xargs -L 1 env $(GOTEST) $(TEST_FLAGS)

GREEN := "\\033[0;32m"
NC := "\\033[0m"
define print
	echo $(GREEN)$1$(NC)
endef

# =======
# TESTING
# =======

unit:
	@$(call print, "Running unit tests.")
	$(UNIT)

test-docker:
	@$(call print, "Running unit tests inside golang:1.24.5 container.")
	docker run --rm \
		-v $(PWD):/workspace \
		-w /workspace \
		-e GO111MODULE=on \
		golang:1.24.5 go test ./...

lint-docker:
	@$(call print, "Running linter in Docker container.")
	docker build -f Dockerfile.lint -t mcp-lnc-lint .
	docker run --rm mcp-lnc-lint

unit-cover:
	@$(call print, "Running unit tests with coverage.")
	$(GOTEST) -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# ==========
# FORMATTING
# ==========

fmt:
	@$(call print, "Formatting source.")
	gofmt -l -w -s $(GOFILES_NOVENDOR)

fmt-check:
	@$(call print, "Checking formatting.")
	@gofmt -l $(GOFILES_NOVENDOR) | tee /tmp/fmt-issues
	@test ! -s /tmp/fmt-issues || (echo "Code is not formatted. Please run 'make fmt'" && exit 1)

lint:
	@$(call print, "Linting source.")
	golangci-lint run -v

# =======
# MODULES
# =======

mod-tidy:
	@$(call print, "Tidying modules.")
	$(GOMOD) tidy

mod-check:
	@$(call print, "Checking modules.")
	$(GOMOD) tidy
	if test -n "$$(git status --porcelain | grep -e "go.mod\|go.sum")"; then echo "Running go mod tidy changes go.mod/go.sum"; git status; git diff; exit 1; fi

# ============
# INSTALLATION
# ============

build:
	@$(call print, "Building MCP LNC server.")
	$(GOBUILD) -tags="$(DEV_TAGS)" -o mcp-lnc-server $(LDFLAGS) .

build-release:
	@$(call print, "Building release MCP LNC server.")
	$(GOBUILD) -o mcp-lnc-server $(LDFLAGS) .

install:
	@$(call print, "Installing MCP LNC server.")
	$(GOINSTALL) -tags="${tags}" $(LDFLAGS) .

# =======
# CLEANUP
# =======

clean:
	@$(call print, "Cleaning build artifacts.")
	rm -f mcp-lnc-server
	rm -f coverage.out coverage.html

# =======
# QUALITY
# =======

check: fmt lint mod-check unit
	@$(call print, "All checks passed!")

# ==============
# CONTAINERIZATION
# ==============

docker-build:
	@$(call print, "Building Docker image.")
	docker build -t mcp-lnc-server .

docker-run:
	@$(call print, "Running Docker container.")
	docker run --rm -p 8080:8080 mcp-lnc-server

# =======
# HELPERS
# =======

help:
	@echo "Available targets:"
	@echo "  build         - Build the MCP LNC server binary"
	@echo "  build-release - Build optimized release binary"
	@echo "  install       - Install the binary to GOPATH/bin"
	@echo "  unit          - Run unit tests"
	@echo "  test-docker   - Run unit tests inside golang:1.24.5 container"
	@echo "  unit-cover    - Run unit tests with coverage"
	@echo "  fmt           - Format Go source code"
	@echo "  fmt-check     - Check if code is formatted"
	@echo "  lint          - Run golangci-lint"
	@echo "  lint-docker   - Run golangci-lint in Docker container"
	@echo "  mod-tidy      - Tidy Go modules"
	@echo "  mod-check     - Check if modules are tidy"
	@echo "  check         - Run all checks (fmt, lint, mod-check, unit)"
	@echo "  clean         - Clean build artifacts"
	@echo "  docker-build  - Build Docker image"
	@echo "  docker-run    - Run Docker container"
	@echo "  help          - Show this help message"

# Instruct make to not interpret these as file/folder related targets
.PHONY: unit test-docker lint-docker unit-cover fmt fmt-check lint mod-tidy mod-check build build-release install clean check docker-build docker-run help
