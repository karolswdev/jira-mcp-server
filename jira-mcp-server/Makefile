# Makefile for jira-mcp-server

# Default target
.PHONY: all
all: build

# Build the Go application
.PHONY: build
build:
	@echo "Building jira-mcp-server..."
	go build -o jira-mcp-server ./cmd/main.go

# Run the Go application
# Note: Requires environment variables (e.g., JIRA_URL, JIRA_USER, JIRA_TOKEN) to be set.
.PHONY: run
run:
	@echo "Running jira-mcp-server (ensure env vars are set)..."
	go run ./cmd/main.go

# Run unit tests (excludes tests with build tags like 'integration')
.PHONY: test
test:
	@echo "Running unit tests..."
	go test -cover -coverprofile=coverage.out ./...

# Run integration tests (only tests tagged with 'integration' in the cmd directory)
.PHONY: test-integration
test-integration:
	@echo "Cleaning test cache..."
	go clean -testcache
	@echo "Running integration tests..."
	go test -tags=integration -cover -coverprofile=coverage-integration.out ./cmd/...

# Run all tests (unit and integration)
.PHONY: test-all
test-all:
	@echo "Running all tests (unit + integration)..."
	go test -tags=integration ./...

# Generate and view unit test coverage report
.PHONY: coverage
coverage: test ## Generate and view unit test coverage report
	@echo "Opening unit test coverage report..."
	@go tool cover -html=coverage.out

# Generate and view integration test coverage report
.PHONY: coverage-integration
coverage-integration: test-integration ## Generate and view integration test coverage report
	@echo "Opening integration test coverage report..."
	@go tool cover -html=coverage-integration.out

# Lint the code (Placeholder - to be implemented in Task 3)
.PHONY: lint
lint:
	@echo "Running linter..."
	golangci-lint run ./...

# Format the code using go fmt and goimports
# Prerequisite: goimports (install with: go install golang.org/x/tools/cmd/goimports@latest)
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	go fmt ./...
	goimports -w .

# Build the Docker image
.PHONY: docker-build
docker-build:
	@echo "Building Docker image..."
	docker build -t jira-mcp-server .

# Run the application using Docker Compose
.PHONY: docker-run
docker-run:
	@echo "Running with Docker Compose..."
	docker-compose up