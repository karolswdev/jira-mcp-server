# Plan: Making jira-mcp-server GitHub Ready (`up-to-github.md`)

**Goal:** Prepare the `jira-mcp-server` project for public release on GitHub, ensuring code quality, maintainability, testability, clear documentation, and adherence to Go best practices.

**Target Audience:** Development team working on `jira-mcp-server`.

**Current State:** Functional MCP server with basic JIRA interaction (create, search, get issue, get epic issues), Docker support, initial documentation, but lacks robust testing and has potential layout issues.

**Key Concerns Addressed:**
*   Lack of meaningful tests.
*   Need for mocking external dependencies (JIRA API).
*   Project file layout requires review and cleanup.

---

## Phase 1: Foundational Cleanup & Structure (Priority: High)

*Focus: Address immediate structural issues, establish code quality standards, and set up basic tooling.*

1.  [COMPLETED] **Fix Project File Layout:**
    *   **Why:** The current layout (`jira-mcp-server/jira-mcp-server/internal/...`) indicates a problematic nesting. The `jira-mcp-server-app` directory is also unclear. The empty `pkg/` directory needs a decision. A clean, standard layout improves navigation and understanding.
    *   **Action:**
        *   Move the contents of `jira-mcp-server/jira-mcp-server/*` up one level into `jira-mcp-server/`.
        *   Remove the now-empty inner `jira-mcp-server` directory.
        *   Investigate `jira-mcp-server-app`: Determine its purpose. If it's redundant or belongs elsewhere, move or remove it.
        *   Address `pkg/`:
            *   If it's intended for reusable library code (that *other* projects might import), keep it and plan what goes there.
            *   If *all* code is application-specific, remove the `pkg/` directory and keep code within `internal/`. (This seems more likely for this type of server).
        *   Ensure the final top-level structure looks something like this (assuming `pkg` is removed and `jira-mcp-server-app` is resolved):
            ```
            .
            ├── .github/         # (Added later for CI/Templates)
            ├── build.sh         # (Or Makefile/Taskfile)
            ├── cmd/
            │   └── main.go
            ├── docs/
            │   └── architecture.md
            ├── internal/
            │   ├── handlers/
            │   │   └── jira_handlers.go
            │   └── jira/
            │       └── client.go
            ├── testdata/        # (Added later for test fixtures)
            ├── .dockerignore
            ├── .gitignore
            ├── docker-compose.yml
            ├── Dockerfile
            ├── go.mod
            ├── go.sum
            ├── LICENSE
            ├── README.md
            └── up-to-github.md  # This plan
            ```

2.  [COMPLETED] **Introduce Standard Build/Task Runner:**
    *   **Why:** `build.sh` is basic. A `Makefile` or `Taskfile.yaml` provides standardized commands for common tasks (build, test, lint, run, docker-build) across different environments.
    *   **Action:**
        *   Create a `Makefile` or `Taskfile.yaml` at the project root.
        *   Define targets/tasks for:
            *   `build`: `go build -o jira-mcp-server ./cmd/main.go`
            *   `run`: `go run ./cmd/main.go` (requires env vars set)
            *   `test`: `go test ./...` (will add flags later)
            *   `lint`: (Integrate linter from next step)
            *   `fmt`: `go fmt ./... && goimports -w .`
            *   `docker-build`: `docker build -t jira-mcp-server .`
            *   `docker-run`: `docker-compose up`
        *   Remove or update `build.sh` accordingly.

3.  [COMPLETED] **Implement Linting and Formatting:**
    *   **Why:** Enforces consistent code style and catches potential errors early. Essential for collaboration and maintainability.
    *   **Action:**
        *   Add `golangci-lint` to the project (e.g., via `go install` or include in dev environment).
        *   Create a `.golangci.yml` configuration file with sensible defaults (e.g., enable `govet`, `errcheck`, `staticcheck`, `unused`, `goimports`, `gofmt`).
        *   Integrate the lint command into the `Makefile`/`Taskfile`.
        *   Run `go fmt ./...` and `goimports -w .` on the entire codebase.
        *   Run `golangci-lint run` and fix reported issues.

4.  [COMPLETED] **Dependency Review:**
    *   **Why:** Ensure dependencies are necessary and up-to-date. Clean up `go.mod` and `go.sum`.
    *   **Action:**
        *   Review `go.mod`. Are all listed dependencies actively used?
        *   Run `go mod tidy` to clean up `go.mod` and `go.sum`.

---

## Phase 2: Robust Testing (Priority: High)

*Focus: Address the core testing concerns by implementing unit and integration tests with appropriate mocking.*

1.  [x] **Refactor for Testability (Dependency Injection):**
    *   **Why:** Handlers currently create the `jira.Client` directly (`jira.NewClient()`). This makes it hard to test handlers without hitting the real API. We need to inject dependencies.
    *   **Action:**
        *   Define an interface for the `jira.Client`, capturing the methods used by the handlers (e.g., `CreateIssue`, `SearchIssues`, `GetIssue`).
            ```go
            // internal/jira/client.go or a new internal/ports/jira.go
            type JiraService interface {
                CreateIssue(req CreateIssueRequest) (*CreateIssueResponse, error)
                SearchIssues(jql string, maxResults int, fields []string) (*SearchResponse, error)
                GetIssue(issueKey string, fields []string) (*Issue, error)
                // Add other methods as needed
            }
            ```
        *   Ensure `jira.Client` implicitly satisfies this interface.
        *   Modify handlers to accept this `JiraService` interface, likely via a struct holding the dependency.
            ```go
            // internal/handlers/jira_handlers.go
            type JiraHandlers struct {
                JiraService jira.JiraService
            }

            func NewJiraHandlers(service jira.JiraService) *JiraHandlers {
                return &JiraHandlers{JiraService: service}
            }

            func (h *JiraHandlers) CreateJiraIssueHandler(w http.ResponseWriter, r *http.Request) {
                // ... parse request ...
                // Use h.JiraService instead of jira.NewClient()
                resp, err := h.JiraService.CreateIssue(req)
                // ... handle response ...
            }
            // Adapt other handlers similarly
            ```
        *   Update `cmd/main.go` to initialize the real `jira.Client` and inject it into the `JiraHandlers` struct when setting up routes.

2.  [x] **Implement Unit Tests for Handlers:**
    *   **Why:** Test the logic within each handler function (request parsing, validation, calling the service interface, response writing) *without* involving the real JIRA client or HTTP transport.
    *   **Action:**
        *   Create `_test.go` files (e.g., `internal/handlers/jira_handlers_test.go`).
        *   Use the `net/http/httptest` package to create mock HTTP requests (`httptest.NewRequest`) and response recorders (`httptest.NewRecorder`).
        *   Create mock implementations of the `JiraService` interface that return predefined responses or errors for testing different scenarios (e.g., success, JIRA error, invalid input).
        *   Instantiate `JiraHandlers` with the mock service.
        *   Call the handler function directly (`handler.CreateJiraIssueHandler(rr, req)`).
        *   Assert the HTTP status code and response body recorded in the `httptest.ResponseRecorder`.
        *   Cover success cases, error cases (bad input, service errors), and edge cases for each handler.

3.  [x] **Implement Unit Tests for JIRA Client:**
    *   **Why:** Test the `jira.Client`'s logic (constructing API requests, handling auth, parsing responses) *without* making actual external HTTP calls.
    *   **Action:**
        *   Create `internal/jira/client_test.go`.
        *   Modify `jira.Client` to allow injecting an `*http.Client`. If not already done, make `httpClient *http.Client` a field set in `NewClient`.
        *   Use Go's `net/http/httptest` package to create a mock HTTP server (`httptest.NewServer`).
        *   In your tests, configure this mock server to expect specific requests (URL, method, headers, body) and return predefined responses (JSON payloads mimicking JIRA API, or error statuses).
        *   Create a `jira.Client` instance in your test, configuring its `baseURL` to the mock server's URL and potentially providing a custom `http.Client` that interacts with the test server.
        *   Call the `jira.Client` methods (e.g., `client.CreateIssue`).
        *   Assert that the client methods return the expected data structures or errors based on the mock server's responses.
        *   Test correct request formation, auth header presence, handling of different JIRA response codes (200, 201, 400, 404, 500), and response body parsing.

4.  [x] **Implement Integration Tests (API Layer with Mocked JIRA):**
    *   **Why:** Test the full flow from an incoming HTTP request through the handlers and the JIRA client *interacting with a mocked JIRA API*. This directly addresses the "mock a JIRA API" concern.
    *   **Action:**
        *   Create tests (potentially in `cmd/main_test.go` or a separate `test/integration/` package) that:
            *   Start a test instance of your *entire* MCP server (`http.Server`) listening on a random available port (`httptest.NewServer` can wrap your main router).
            *   Start a *second* `httptest.NewServer` acting as the **Mock JIRA API**. Configure its handlers to mimic JIRA endpoints (`/rest/api/3/issue`, `/rest/api/3/search`, etc.) based on the requests received.
            *   Configure the MCP server instance (via env vars passed to the test process or modified config) to point its `JIRA_URL` to the Mock JIRA API's URL.
            *   Use a standard Go `http.Client` to send requests (POST, GET) to your running MCP server instance's endpoints (e.g., `/create_jira_issue`, `/jira_issue/TEST-1`).
            *   Assert the HTTP responses received from your MCP server.
            *   Optionally, assert that the Mock JIRA API received the expected requests from the MCP server.

5.  [x] **Establish Test Coverage:**
    *   **Why:** Measure the effectiveness of the tests and identify untested code paths.
    *   **Action:**
        *   Update the `test` target in `Makefile`/`Taskfile` to include coverage: `go test -cover -coverprofile=coverage.out ./...`
        *   Optionally add a target to view coverage: `go tool cover -html=coverage.out`
        *   Aim for a meaningful coverage percentage (e.g., >70-80%), focusing on critical logic. Don't just chase numbers; ensure important scenarios are tested.

---

## Phase 3: Enhancements & Polish (Priority: Medium)

*Focus: Improve robustness, configuration, logging, and documentation.*

1.  [x] **Enhance Error Handling:**
    *   **Why:** Current error handling often returns generic 500 errors. Map specific errors (e.g., JIRA 404 Not Found) to appropriate HTTP status codes. Provide clearer error messages.
    *   **Action:**
        *   In `internal/jira/client.go`, parse JIRA API error responses more granularly if possible. Return custom error types or check status codes.
        *   In `internal/handlers/jira_handlers.go`, check for specific error types returned by the `JiraService`. Map JIRA 404 to HTTP 404, JIRA 400 to HTTP 400, etc., using `respondWithError`.
        *   Avoid exposing raw internal error details in production responses unless intended. Log detailed errors internally.

2.  [x] **Implement Structured Logging:**
    *   **Why:** `log.Printf` is basic. Structured logging (e.g., JSON format) makes logs easier to parse, filter, and analyze, especially in containerized environments.
    *   **Action:**
        *   Replace standard `log` calls with a structured logging library (e.g., Go's built-in `log/slog` (Go 1.21+) or a library like `zerolog` or `zap`).
        *   Log key information with requests (method, path, status code, duration) and critical events/errors with relevant context (e.g., issue key, JQL).

3.  [x] **Refine Configuration Management:**
    *   **Why:** Relying solely on env vars can be limiting. Consider libraries for more flexible configuration (e.g., from files, flags).
    *   **Action:**
        *   Evaluate libraries like `Viper` or `cleanenv`.
        *   Update `cmd/main.go` to load configuration using the chosen library. Continue supporting environment variables as the primary method for Docker/cloud-native deployments, but potentially allow overrides via flags or a config file.
        *   Document the configuration options clearly in the README.

4.  [x] **Context Propagation:**
    *   **Why:** Allows for request cancellation, deadlines, and passing request-scoped values through the call chain, improving resilience and observability.
    *   **Action:**
        *   Add `context.Context` as the first parameter to handler functions and `JiraService` interface methods.
        *   Pass `r.Context()` from the handler down through the service calls.
        *   In `internal/jira/client.go`, use `http.NewRequestWithContext` to associate the context with outgoing JIRA API calls.

5.  [COMPLETED] **Update Documentation:**
    *   **Why:** Documentation must reflect the current state, including new endpoints, testing strategy, and configuration.
    *   **Action:**
        *   **README.md:** Update setup instructions (Makefile/Taskfile), configuration details, API endpoint descriptions (ensure request/response examples are accurate), add sections on testing and contributing (later).
        *   **docs/architecture.md:** Update component interactions (reflecting DI), update sequence diagrams (remove TODOs, ensure they match current handlers/endpoints). Add a note about the testing strategy (unit, integration, mocking).
        *   **Code Comments:** Add `godoc` comments to exported types, functions, and methods, explaining their purpose.

---

## Phase 4: GitHub & Community Readiness (Priority: Medium-Low)

*Focus: Prepare the repository for public visibility and potential contributions.*

1.  [COMPLETED] **Implement CI/CD Pipeline (GitHub Actions):**
    *   **Why:** Automates testing, linting, and building on every push/PR, ensuring code quality and stability.
    *   **Action:**
        *   Create `.github/workflows/ci.yml`.
        *   Define jobs for:
            *   Checking formatting (`go fmt`, `goimports`).
            *   Running linters (`golangci-lint run`).
            *   Running tests (`go test -cover ./...`).
            *   Building the binary (`go build`).
            *   (Optional) Building Docker image.
            *   (Optional) Uploading coverage reports (e.g., Codecov, Coveralls).
        *   Ensure workflows trigger on pushes to `main` and on pull requests.

2.  [COMPLETED] **Add Contribution Guidelines:**
    *   **Why:** Sets expectations for potential contributors.
    *   **Action:**
        *   Create a `CONTRIBUTING.md` file.
        *   Outline the process for reporting bugs (issues), suggesting features (issues), and submitting changes (pull requests). Mention required checks (linting, tests passing).

3.  [COMPLETED] **Add Code of Conduct:**
    *   **Why:** Fosters a welcoming and inclusive community.
    *   **Action:**
        *   Create a `CODE_OF_CONDUCT.md` file. Use a standard template like the Contributor Covenant.

4.  [COMPLETED] **Add Issue and PR Templates:**
    *   **Why:** Standardizes bug reports, feature requests, and pull request descriptions.
    *   **Action:**
        *   Create templates in `.github/ISSUE_TEMPLATE/` (for bug reports, feature requests) and `.github/PULL_REQUEST_TEMPLATE.md`.

5.  **Final Repository Polish:**
    *   **Why:** First impressions matter on GitHub.
    *   **Action:**
        *   Write a clear and concise repository description.
        *   Add relevant repository topics/tags (e.g., `golang`, `jira`, `mcp`, `api`, `http-server`).
        *   Ensure `LICENSE` and `README.md` are present and accurate.
        *   [x] Finalize CHANGELOG.md for v0.1.0 release. (Tag creation pending).
        *   [x] Add badges to the README (build status, coverage, Go report card).