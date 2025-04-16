# JIRA MCP Server

A Go implementation of a Model Context Protocol (MCP) server for interacting with JIRA Cloud REST API.
[![Go CI](https://github.com/<USER>/<REPO>/actions/workflows/ci.yml/badge.svg)](https://github.com/<USER>/<REPO>/actions/workflows/ci.yml) [![Go Report Card](https://goreportcard.com/badge/github.com/<USER>/<REPO>)](https://goreportcard.com/report/github.com/<USER>/<REPO>) [![codecov](https://codecov.io/gh/<USER>/<REPO>/branch/main/graph/badge.svg?token=<CODECOV_TOKEN>)](https://codecov.io/gh/<USER>/<REPO>) <!-- Replace with actual Codecov setup later --> ![Go Version](https://img.shields.io/badge/go-1.23.x-blue.svg) <!-- Update version if needed -->

## Prerequisites

- Go 1.20+ installed (for development)
- Docker (optional, for containerized deployment)
- JIRA Cloud account with API access
- API token for your JIRA account

## Getting Started

1.  **Clone the repository:**
    ```bash
    git clone <repository-url>
    cd jira-mcp # Or the name of your cloned directory
    ```
    **Note:** Most subsequent commands (like `make ...`) should be run from within the `jira-mcp-server` subdirectory.
    ```bash
    ```
2.  **Configure:** Set the required environment variables (see [Configuration](#configuration) below) or create a `config.yaml`.
3.  **Build (inside `jira-mcp-server` directory):**
    ```bash
    make build
    ```
    This compiles the server binary to `jira-mcp-server`.

## Configuration

Configuration is managed using [Viper](https://github.com/spf13/viper) and loaded from the following sources in order of precedence:

1.  **Environment Variables:** Prefixed with `JIRA_MCP_`.
2.  **Configuration File:** `config.yaml` (or `.json`, `.toml`) in the `jira-mcp-server` directory (optional). See `jira-mcp-server/config.yaml.example`.
3.  **Defaults:** Default values set within the application.

**Required Configuration:**

The following values must be provided either via environment variables (highest precedence) or the configuration file:

*   `JIRA_MCP_JIRA_URL`: The base URL of your JIRA Cloud instance (e.g., `https://your-domain.atlassian.net`).
*   `JIRA_MCP_JIRA_USER_EMAIL`: The email address associated with the API token user.
*   `JIRA_MCP_JIRA_API_TOKEN`: Your JIRA API token.

**Optional Configuration:**

*   `JIRA_MCP_PORT`: The port the server listens on. Defaults to `8080`.

**Example Environment Variables:**

```bash
export JIRA_MCP_JIRA_URL="https://your-domain.atlassian.net"
export JIRA_MCP_JIRA_USER_EMAIL="your.email@example.com"
export JIRA_MCP_JIRA_API_TOKEN="your-api-token"
export JIRA_MCP_PORT="8888" # Optional, defaults to 8080
```

Refer to `jira-mcp-server/config.yaml.example` for the structure of the optional configuration file. Remember that environment variables will always override values set in the file.

4.  **Run (inside `jira-mcp-server` directory):**
    ```bash
    make run
    ```
    Alternatively, run the compiled binary directly:
    ```bash
    ./jira-mcp-server
    ```
    The server will listen on the configured port (default: 8080).

### Running with Docker (inside `jira-mcp-server` directory)

1.  **Build the image (inside `jira-mcp-server` directory):**
    ```bash
    make docker-build
    ```
2.  **Create `.env` file:** Create a `.env` file *inside the `jira-mcp-server` directory* with your `JIRA_MCP_` environment variables (see [Configuration](#configuration)).
3.  **Run using Docker (inside `jira-mcp-server` directory):**
    ```bash
    make docker-run
    ```
    This uses the `.env` file and maps the default port 8080.
4.  **Run using Docker Compose (inside `jira-mcp-server` directory):**
    ```bash
    make docker-compose-up
    ```
    (Ensure your `.env` file is present). To stop: `make docker-compose-down`.

## Testing (run commands inside `jira-mcp-server` directory)

The project includes both unit and integration tests.

*   **Unit Tests:** Test individual functions and components in isolation, often using mocks (e.g., for the JIRA client). Run with:
    ```bash
    make test
    ```
*   **Integration Tests:** Test the interaction between components, including the HTTP API layer using `httptest` against a mock JIRA server. Run with:
    ```bash
    make test-integration
    ```

**Test Coverage:**

You can generate and view test coverage reports:

*   Unit Test Coverage:
    ```bash
    make coverage
    # This generates coverage.html. Open it in your browser.
    ```
*   Integration Test Coverage:
    ```bash
    make coverage-integration
    # This generates coverage-integration.html. Open it in your browser.
    ```


## Architecture

For a detailed overview of the server's architecture, see the [Architecture Document](./jira-mcp-server/docs/architecture.md).


## API Endpoint

- **POST /create_jira_issue**: Creates a new JIRA issue
  - Required parameters:
    - `project_key`: The JIRA project key (e.g., "PROJ")
    - `summary`: The issue title
    - `issue_type`: The type of issue (e.g., "Story", "Task", "Bug")
  - Optional parameters:
    - `description`: Detailed description
    - `assignee_email`: Email of assignee
    - `parent_key`: Key of parent issue (for sub-tasks)

- **POST /search_jira_issues**: Searches for JIRA issues using JQL.
  - Request Body (JSON):
    - `jql` (string, required): The JIRA Query Language string.
    - `max_results` (int, optional): Maximum number of issues to return.
    - `fields` ([]string, optional): List of fields to return for each issue (e.g., `["summary", "status"]`). Defaults to a standard set if omitted.
  - Example Success Response (JSON): `jira.SearchResponse` containing a list of issues matching the JQL.

- **GET /jira_issue/{issueKey}**: Retrieves details for a specific JIRA issue.
  - Path Parameter:
    - `issueKey` (string, required): The key of the issue (e.g., "PROJ-123").
  - Query Parameter:
    - `fields` (string, optional): Comma-separated list of fields to return (e.g., `fields=summary,status,assignee`). Defaults to a standard set if omitted.
  - Example Success Response (JSON): `jira.Issue` containing the details of the requested issue.

- **GET /jira_epic/{epicKey}/issues**: Retrieves all issues belonging to a specific Epic.
  - Path Parameter:
    - `epicKey` (string, required): The key of the Epic issue (e.g., "PROJ-456").
  - Example Success Response (JSON): `jira.SearchResponse` containing a list of issues linked to the specified Epic.

## Example Request (/create_jira_issue)

```bash
curl -X POST http://localhost:8080/create_jira_issue \
  -H "Content-Type: application/json" \
  -d '{
    "project_key": "PROJ",
    "summary": "Implement new feature",
    "issue_type": "Story",
    "description": "Detailed description here"
  }'
```

## Example Response (/create_jira_issue)

```json
{
  "message": "JIRA issue created successfully",
  "key": "PROJ-123",
  "url": "https://your-domain.atlassian.net/browse/PROJ-123"
}
```

## Example Request (/search_jira_issues)

```bash
curl -X POST http://localhost:8080/search_jira_issues \
  -H "Content-Type: application/json" \
  -d '{
    "jql": "project = PROJ AND status = \"To Do\" ORDER BY created DESC",
    "max_results": 5,
    "fields": ["summary", "status", "assignee"]
  }'
```

## Example Response (/search_jira_issues)

```json
{
  "expand": "names,schema",
  "startAt": 0,
  "maxResults": 5,
  "total": 2,
  "issues": [
    {
      "expand": "operations,versionedRepresentations,editmeta,changelog,renderedFields",
      "id": "10001",
      "self": "https://your-domain.atlassian.net/rest/api/3/issue/10001",
      "key": "PROJ-124",
      "fields": {
        "summary": "Another task",
        "status": {
          "self": "https://your-domain.atlassian.net/rest/api/3/status/10000",
          "description": "",
          "iconUrl": "https://your-domain.atlassian.net/",
          "name": "To Do",
          "id": "10000",
          "statusCategory": {
            "self": "https://your-domain.atlassian.net/rest/api/3/statuscategory/2",
            "id": 2,
            "key": "new",
            "colorName": "blue-gray",
            "name": "To Do"
          }
        },
        "assignee": null
      }
    },
    {
      "expand": "operations,versionedRepresentations,editmeta,changelog,renderedFields",
      "id": "10000",
      "self": "https://your-domain.atlassian.net/rest/api/3/issue/10000",
      "key": "PROJ-123",
      "fields": {
        "summary": "Implement new feature",
        "status": {
          "self": "https://your-domain.atlassian.net/rest/api/3/status/10000",
          "description": "",
          "iconUrl": "https://your-domain.atlassian.net/",
          "name": "To Do",
          "id": "10000",
          "statusCategory": {
            "self": "https://your-domain.atlassian.net/rest/api/3/statuscategory/2",
            "id": 2,
            "key": "new",
            "colorName": "blue-gray",
            "name": "To Do"
          }
        },
        "assignee": null
      }
    }
  ]
}
```

## Example Request (/jira_issue/{issueKey})

```bash
curl -X GET "http://localhost:8080/jira_issue/PROJ-123?fields=summary,status,issuetype"
```

## Example Response (/jira_issue/{issueKey})

```json
{
  "expand": "renderedFields,names,schema,operations,editmeta,changelog,versionedRepresentations",
  "id": "10000",
  "self": "https://your-domain.atlassian.net/rest/api/3/issue/10000",
  "key": "PROJ-123",
  "fields": {
    "summary": "Implement new feature",
    "issuetype": {
      "self": "https://your-domain.atlassian.net/rest/api/3/issuetype/10001",
      "id": "10001",
      "description": "A task that needs to be done.",
      "iconUrl": "https://your-domain.atlassian.net/rest/api/2/universal_avatar/view/type/issuetype/avatar/10318?size=medium",
      "name": "Task",
      "subtask": false,
      "avatarId": 10318,
      "entityId": "uuid-goes-here",
      "hierarchyLevel": 0
    },
    "status": {
      "self": "https://your-domain.atlassian.net/rest/api/3/status/10000",
      "description": "",
      "iconUrl": "https://your-domain.atlassian.net/",
      "name": "To Do",
      "id": "10000",
      "statusCategory": {
        "self": "https://your-domain.atlassian.net/rest/api/3/statuscategory/2",
        "id": 2,
        "key": "new",
        "colorName": "blue-gray",
        "name": "To Do"
      }
    }
  }
}
```

## Example Request (/jira_epic/{epicKey}/issues)

```bash
curl -X GET http://localhost:8080/jira_epic/PROJ-456/issues
```

## Example Response (/jira_epic/{epicKey}/issues)

```json
{
  "expand": "names,schema",
  "startAt": 0,
  "maxResults": 50,
  "total": 1,
  "issues": [
    {
      "expand": "operations,versionedRepresentations,editmeta,changelog,renderedFields",
      "id": "10005",
      "self": "https://your-domain.atlassian.net/rest/api/3/issue/10005",
      "key": "PROJ-457",
      "fields": {
        "summary": "Story within the epic",
        "status": {
          "self": "https://your-domain.atlassian.net/rest/api/3/status/10001",
          "description": "",
          "iconUrl": "https://your-domain.atlassian.net/",
          "name": "In Progress",
          "id": "10001",
          "statusCategory": {
            "self": "https://your-domain.atlassian.net/rest/api/3/statuscategory/4",
            "id": 4,
            "key": "indeterminate",
            "colorName": "yellow",
            "name": "In Progress"
          }
        },
        "issuetype": {
           "self": "https://your-domain.atlassian.net/rest/api/3/issuetype/10002",
           "id": "10002",
           "description": "A user story.",
           "iconUrl": "https://your-domain.atlassian.net/rest/api/2/universal_avatar/view/type/issuetype/avatar/10315?size=medium",
           "name": "Story",
           "subtask": false,
           "avatarId": 10315,
           "hierarchyLevel": 0
        }
        // ... other fields ...
      }
    }
    // ... potentially more issues ...
  ]
}
```


## Security Note

Never commit your JIRA API token or other sensitive credentials to version control.