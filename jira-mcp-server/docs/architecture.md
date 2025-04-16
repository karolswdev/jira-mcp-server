# JIRA MCP Server Architecture

This document provides an overview of the JIRA Model Context Protocol (MCP) server's architecture.

## Overview

The JIRA MCP server acts as a bridge between a Large Language Model (LLM) and the JIRA Cloud REST API. It listens for specific MCP commands via HTTP requests, translates these commands into appropriate JIRA API calls, executes them, and returns the results to the LLM. It now supports creating issues (`POST /create_jira_issue`), searching issues via JQL (`POST /search_jira_issues`), retrieving specific issues (`GET /jira_issue/{issueKey}`), and getting issues within an epic (`GET /jira_epic/{epicKey}/issues`). Routing is handled by `gorilla/mux` to support path parameters.

Authentication with the JIRA API is handled using a JIRA Cloud API Token, with credentials (URL, User Email, API Token) securely read from environment variables.

## Key Components

1.  **HTTP Server (`cmd/main.go`):**
    *   Initializes configuration (Viper).
    *   Initializes structured logging (`slog`).
    *   Initializes the `jira.Client` (which implements `handlers.JiraService`) using configuration.
    *   Initializes the `handlers.JiraHandlers` struct, injecting the logger and JIRA service.
    *   Initializes the `gorilla/mux` router and registers routes, associating them with methods on the `JiraHandlers` struct.
    *   Starts the HTTP server using `net/http`, listening on the configured port.

2.  **Request Handlers (`internal/handlers/jira_handlers.go`):**
    *   Defines the `JiraHandlers` struct which holds dependencies like the `JiraService` and `*slog.Logger`.
    *   Provides `NewJiraHandlers` to create an instance with injected dependencies.
    *   Contains handler methods (e.g., `CreateJiraIssueHandler`, `SearchJiraIssuesHandler`) attached to the `JiraHandlers` struct.
    *   Each handler method is responsible for:
        *   Logging the incoming request using the injected logger.
        *   Parsing the HTTP request (body, path parameters, query parameters).
        *   Calling the appropriate method on the injected `JiraService` (e.g., `jh.service.CreateIssue`).
        *   Handling potential errors from the service call.
        *   Formatting and writing the JSON response (success or error) back using helper functions.

3.  **JIRA Client (`internal/jira/client.go`):**
    *   Defines the `Client` struct which encapsulates the logic for interacting with the JIRA Cloud REST API.
    *   Implements the `handlers.JiraService` interface.
    *   Provides a `NewClient` function, typically called in `main.go`, to initialize the client with configuration (URL, credentials) and an `http.Client`.
    *   Contains methods corresponding to JIRA API operations:
        *   `CreateIssue`: Creates a new issue.
        *   `SearchIssues`: Searches issues using JQL.
        *   `GetIssue`: Retrieves a single issue by key.
        *   `GetEpicIssues`: Retrieves issues belonging to an epic (implemented via a specific JQL search).
    *   Handles constructing API requests, making authenticated HTTP calls using the underlying `http.Client`, and parsing JIRA API responses.


## Diagrams

### High-Level Component Diagram

```mermaid
graph TD
    LLM -->|MCP Command (HTTP)| Router[Router (gorilla/mux)];
    Router -->|Request| Handlers[Request Handlers (handlers pkg)];
    Handlers -->|Client Call| JiraClient[JIRA Client (jira pkg)];
    JiraClient -->|REST API Call (HTTPS)| JiraAPI[JIRA Cloud API];
    JiraAPI -->|API Response| JiraClient;
    JiraClient -->|Result| Handlers;
    Handlers -->|MCP Response (JSON)| Router;
    Router -->|Response| LLM;

    subgraph JiraMCP [JIRA MCP Server (Go)]
        direction LR
        Router
        Handlers
        JiraClient
    end
```

### Sequence Diagrams


#### `create_jira_issue` Sequence Diagram (Illustrative)

```mermaid
sequenceDiagram
    participant LLM
    participant Router as Mux Router (main.go)
    participant Handler as JiraHandlers.CreateIssueHandler (handlers.go)
    participant Service as JiraService (jira.Client)
    participant JiraAPI as JIRA Cloud API

    Note over main.go: Initializes Logger, JiraService (Client), JiraHandlers (with deps)

    LLM->>+Router: POST /create_jira_issue (JSON Body)
    Router->>+Handler: jh.CreateIssueHandler(w, r)
    Handler->>Handler: Log Request (using injected logger)
    Handler->>Handler: Parse Request Body
    Handler->>+Service: CreateIssue(request)
    Service->>Service: Validate Parameters
    Service->>Service: Construct API Payload (JSON)
    Service->>+JiraAPI: POST /rest/api/3/issue (Payload, Auth Header)
    JiraAPI-->>-Service: Response (Success/Error)
    Service->>Service: Parse Response
    alt Success
        Service-->>-Handler: Success: IssueResponse{Key, Self}
    else Error
        Service-->>-Handler: Error: Failed to create issue (...)
    end
    alt Success
        Handler->>Handler: Write 201 Created (JSON: {message, key, url})
    else Error
        Handler->>Handler: Write 500 Internal Server Error (JSON: {error})
    end
    Handler-->>-Router: Return from handler func
    Router-->>-LLM: HTTP Response
```
## Testing Strategy

The project employs a two-tiered testing approach:

1.  **Unit Tests (`make test`):**
    *   Focus on testing individual components in isolation.
    *   Handlers (`internal/handlers`) are tested using mock implementations of the `JiraService` interface to verify handler logic without making real API calls.
    *   The JIRA client (`internal/jira`) methods are tested by mocking the underlying `http.Client` transport layer to simulate various JIRA API responses (success, errors, specific data).

2.  **Integration Tests (`make test-integration`):**
    *   Focus on testing the interaction between the HTTP server/router and the handlers.
    *   Uses the standard Go `net/http/httptest` package to create test requests against the API endpoints.
    *   A mock JIRA server (also using `httptest`) is often employed to provide controlled responses for the JIRA client during these tests, ensuring the full request-response flow through the MCP server is validated without external dependencies.

Coverage reports can be generated separately for unit (`make coverage`) and integration (`make coverage-integration`) tests.



*Note: Similar sequences apply for other endpoints like `SearchJiraIssuesHandler`, `GetJiraIssueHandler`, etc., calling the corresponding `JiraService` methods.*

## Configuration

Configuration is managed via Viper, primarily using environment variables prefixed with `JIRA_MCP_` or a `config.yaml` file. See the main `README.md` for full details. Key variables include:

*   `JIRA_MCP_JIRA_URL`: The base URL of the JIRA Cloud instance.
*   `JIRA_MCP_JIRA_USER_EMAIL`: The email address associated with the API token.
*   `JIRA_MCP_JIRA_API_TOKEN`: The generated JIRA API token.
*   `JIRA_MCP_PORT` (Optional): The port the server should listen on (defaults to `8080`).