# Changelog

## [Unreleased]

### Changed
- Moved `README.md` from `jira-mcp-server/` to project root.
- Updated `README.md` command examples and paths to reflect the move.

### Added
- Added `/memory-bank/` to root `.gitignore`.

## [0.1.0] - 2025-04-16

### Added
- Created standard issue templates (`.github/ISSUE_TEMPLATE/bug_report.md`, `.github/ISSUE_TEMPLATE/feature_request.md`) and a pull request template (`.github/PULL_REQUEST_TEMPLATE.md`). (Corresponds to TODO Phase 4, Task 4).
- Added placeholder badges (Build Status, Go Report Card, Codecov, Go Version) to `jira-mcp-server/README.md`. (Corresponds to TODO Phase 4, Task 5).


- `github.com/spf13/viper` dependency for configuration management.
- `config.yaml.example` file demonstrating optional file-based configuration.

- Implemented structured JSON logging using `log/slog`. (Corresponds to TODO Phase 3, Task 2).
- Created `.github/workflows/ci.yml` GitHub Actions workflow to automate checks (format, lint), tests (unit, integration), and builds (binary, Docker) on push/PR to `main`. (Corresponds to TODO Phase 4, Task 1).
- Created `CODE_OF_CONDUCT.md` using the Contributor Covenant v2.1 template. (Corresponds to TODO Phase 4, Task 3).


- Created `CONTRIBUTING.md` outlining bug reporting, feature requests, and pull request process. (Corresponds to TODO Phase 4, Task 2).


- Created `internal/jira/client_test.go` with unit tests for `jira.Client` methods (`CreateIssue`, `SearchIssues`, `GetIssue`).
- Used `net/http/httptest` to mock the JIRA API server responses for testing.
- Added test cases for success, API errors (4xx), auth header checks, and request validation.
- Unit tests for all HTTP handlers in `internal/handlers/jira_handlers_test.go`.
- Mock implementation (`mockJiraService`) for `jira.JiraService` using `testify/mock`.
- Test cases covering success, bad requests (invalid JSON, missing params), and service errors for `CreateJiraIssueHandler`, `SearchIssuesHandler`, `GetIssueDetailsHandler`, and `GetIssuesInEpicHandler`.
- `github.com/stretchr/testify` dependency for assertions and mocking.
- Integration tests (`cmd/main_integration_test.go`) using `httptest` for MCP server and mock JIRA API. (Corresponds to TODO Phase 2, Task 4).
- Build tag `integration` to separate integration tests.
- `test-integration` and `test-all` targets in `Makefile`.
- `EpicLinkFieldName` constant in `internal/jira` package for epic link JQL.

- Added `coverage` and `coverage-integration` targets to `Makefile` for generating and viewing unit and integration test coverage reports. (Corresponds to TODO Phase 2, Task 5).


### Changed
- Refactored `cmd/main.go` to use Viper for loading configuration from defaults, environment variables (prefixed with `JIRA_MCP_`), and optional `config.yaml` file. Removed direct `os.Getenv` usage for configuration keys. (Corresponds to TODO Phase 3, Task 3).
- Updated `README.md` configuration section to explain Viper usage, precedence, required keys (`JIRA_MCP_` prefix), and reference `config.yaml.example`. (Corresponds to TODO Phase 3, Task 3).

- Replaced standard `log` calls (`log.Printf`, `log.Fatalf`, `log.Fatal`) with `slog` equivalents (`slog.Info`, `slog.Error`) in `cmd/main.go` and `internal/handlers/jira_handlers.go`. (Corresponds to TODO Phase 3, Task 2).

- Refactored `jira.NewClient` to accept an optional `*http.Client` for dependency injection, defaulting to `http.DefaultClient` if nil. Updated usage in `cmd/main.go`.
- Injected `*slog.Logger` dependency into `JiraHandlers`. (Corresponds to TODO Phase 3, Task 2).

- Corrected JSON request body structure in `CreateJiraIssueHandler` tests.
- Updated unit and integration tests to provide a discard logger to `NewJiraHandlers`. (Corresponds to TODO Phase 3, Task 2).

- Corrected HTTP method and request structure in `SearchIssuesHandler` tests (POST with JSON body).
- Updated mock expectations to handle `nil` vs empty slices correctly for default `fields` arguments.
- Updated mock expectations to use `mock.Anything` for context matching.
- Updated `Makefile` `test` target to implicitly exclude integration tests.
- Updated `Makefile` `test-integration` target to clean test cache (`go clean -testcache`).
- Improved error status code mapping in handlers (`mapJiraErrorToHTTPStatus`) to better reflect JIRA API errors.
- Corrected JQL generation in `GetIssuesInEpicHandler` to use `jira.EpicLinkFieldName` and single quotes for string literals.
- Corrected payload construction logic in `jira.Client.CreateIssue` to use struct fields correctly.

- Updated `test` and `test-integration` targets in `Makefile` to generate coverage profiles (`coverage.out`, `coverage-integration.out`). (Corresponds to TODO Phase 2, Task 5).
- Enhanced error handling in `internal/handlers`:
  - Introduced `jira.JiraAPIError` in `internal/jira/client.go` to wrap JIRA API errors and include status code.
  - Updated `mapJiraError` in `internal/handlers/jira_handlers.go` to check for `*jira.JiraAPIError` and map specific JIRA status codes (400, 401, 403, 404) to corresponding HTTP status codes (`http.StatusBadRequest`, `http.StatusUnauthorized`, `http.StatusForbidden`, `http.StatusNotFound`).
  - Updated handlers to return user-friendly JSON error messages instead of exposing raw internal errors.

 - Verified context propagation (`context.Context`) from handlers (`r.Context()`) through the `JiraService` interface and `jira.Client` methods, including usage of `http.NewRequestWithContext` for outgoing requests. (Corresponds to TODO Phase 3, Task 4).
- Updated `README.md` with `make` commands, accurate configuration/API details, examples, and a new Testing section. (Corresponds to TODO Phase 3, Task 5).
- Updated `docs/architecture.md` to reflect DI, updated sequence diagram, and added Testing Strategy section. (Corresponds to TODO Phase 3, Task 5).
- Added GoDoc comments to exported types/functions/methods in `internal/handlers/jira_handlers.go` and `internal/jira/client.go`. (Corresponds to TODO Phase 3, Task 5).





### Fixed
- Linting error (SA9003 empty branch) in `internal/jira/client.go` by removing unused `if req.AssigneeEmail != ""` block.
- Integration test failures caused by JSON key mismatches (`projectKey` vs `project_key`), incorrect JQL assertions (quoting), incorrect response key assertions (`issueKey` vs `key`), and error handling/propagation issues.

- Fixed unit and integration test assertions that failed due to changes in error message format after implementing structured logging. (Corresponds to TODO Phase 3, Task 2).

- Fixed unit test failures in `jira_handlers_test.go` (incorrect JQL expectation for epic issues) and `client_test.go` (incorrect request body expectation for description field) identified during coverage setup.
- Updated unit tests (`internal/handlers/jira_handlers_test.go`) to mock `*jira.JiraAPIError` and assert correct mapped HTTP status codes and user-friendly error messages.
- Updated integration tests (`cmd/main_integration_test.go`) to assert correct mapped HTTP status codes and user-friendly error messages for API error scenarios (400, 404, 500).
- Fixed unit tests in `internal/jira/client_test.go` to correctly assert `*jira.JiraAPIError` type and properties instead of old error string format.



All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial project structure for `jira-mcp-server`.
- Created `jira-mcp-server/Makefile` with standard targets (`build`, `run`, `test`, `lint` placeholder, `fmt`, `docker-build`, `docker-run`) to standardize common development tasks. (Corresponds to TODO Phase 1, Task 2).
- Added `jira-mcp-server/.golangci.yml` with default linters (`govet`, `errcheck`, `staticcheck`, `unused`, `goimports`, `gofmt`). (Corresponds to TODO Phase 1, Task 3).



### Changed
- Refactored `jira-mcp-server` for testability using dependency injection: Defined `JiraService` interface, updated `jira.Client` methods to accept `context.Context` and use `http.NewRequestWithContext`, modified `JiraHandlers` to accept `JiraService`, and updated `main.go` to inject dependencies. (Corresponds to TODO Phase 2, Task 1).
- Verified and cleaned up `jira-mcp-server` directory structure according to standard Go layout. Removed extraneous `repomix-output.txt`. Confirmed Go import paths are correct. Ran `go mod tidy`. (Corresponds to TODO Phase 1, Task 1).

- Updated `jira-mcp-server/Makefile` `lint` target to execute `golangci-lint run ./...`. (Corresponds to TODO Phase 1, Task 3).
- Reviewed dependencies in `jira-mcp-server/go.mod` and ran `go mod tidy` to ensure consistency and remove unused entries. (Corresponds to TODO Phase 1, Task 4).

### Fixed
- N/A

- Fixed `errcheck` linting errors in `internal/jira/client.go` and `internal/handlers/jira_handlers.go` reported by `golangci-lint`. (Corresponds to TODO Phase 1, Task 3).

### Removed
- `jira-mcp-server/repomix-output.txt`


<!-- Link Definitions -->
[Unreleased]: https://github.com/<USER>/<REPO>/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/<USER>/<REPO>/releases/tag/v0.1.0