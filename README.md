# JIRA MCP Server 🚀

<!-- Badges - TODO: Replace <USER>/<REPO> and add real token/links -->
[![Go CI](https://github.com/[TODO: Replace with GitHub Username]/[TODO: Replace with GitHub Repo Name]/actions/workflows/ci.yml/badge.svg)](https://github.com/[TODO: Replace with GitHub Username]/[TODO: Replace with GitHub Repo Name]/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/[TODO: Replace with GitHub Username]/[TODO: Replace with GitHub Repo Name])](https://goreportcard.com/report/github.com/[TODO: Replace with GitHub Username]/[TODO: Replace with GitHub Repo Name])
[![codecov](https://codecov.io/gh/[TODO: Replace with GitHub Username]/[TODO: Replace with GitHub Repo Name]/branch/main/graph/badge.svg)](https://codecov.io/gh/[TODO: Replace with GitHub Username]/[TODO: Replace with GitHub Repo Name]) <!-- Codecov token set via GitHub Secret: CODECOV_TOKEN -->
![Go Version](https://img.shields.io/badge/go-1.20+-blue.svg)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](./jira-mcp-server/LICENSE)

**A flexible Go server implementing the Model Context Protocol (MCP) to interact with the JIRA Cloud REST API, enabling seamless integration between LLMs/tools and your JIRA projects.**

## What is this? 🤔

This project provides a bridge between systems that speak the Model Context Protocol (like certain AI assistants or development tools) and the powerful JIRA Cloud API. It allows you to perform common JIRA actions (creating issues, searching, retrieving details) programmatically through a standardized MCP interface, abstracting away the complexities of direct JIRA API calls.

## ✨ Features

*   **MCP Interface:** Exposes JIRA actions via standard MCP endpoints.
*   **JIRA Cloud Integration:** Create issues, search using JQL, retrieve issue details, and fetch issues within an Epic.
*   **Flexible Configuration:** Uses Viper for configuration via environment variables, config files, or defaults.
*   **Docker Support:** Ready for containerized deployment using Docker and Docker Compose.
*   **Robust Testing:** Includes comprehensive unit and integration tests.
*   **Structured Logging:** Uses `slog` for clear, structured logging.
*   **Dependency Injection:** Built with clean architecture principles using `wire` for dependency injection.

## Prerequisites

*   Go 1.20+ (for building/running locally)
*   Docker & Docker Compose (optional, for containerized deployment)
*   A JIRA Cloud instance
*   A JIRA API Token associated with a user email

## 🚀 Getting Started

1.  **Clone the repository:**
    ```bash
    git clone https://github.com/[TODO: Replace with GitHub Username]/[TODO: Replace with GitHub Repo Name].git # TODO: Replace with actual URL
    cd [TODO: Replace with GitHub Repo Name] # e.g., cd jira-mcp
    ```
2.  **Navigate to the server directory:**
    ```bash
    cd jira-mcp-server
    ```
    **➡️ Important:** Most subsequent commands (`make ...`, `go run ...`, `docker ...`) should be run from *within* the `jira-mcp-server/` directory.

3.  **Configure the server:** Set the required environment variables or create a `config.yaml`. See the [Configuration](#⚙️-configuration) section below.

4.  **Run the server:** See the [Running the Server](#▶️-running-the-server) section below.

## ⚙️ Configuration

Configuration is managed using [Viper](https://github.com/spf13/viper) and loaded from the following sources in order of precedence:

1.  **Environment Variables:** Prefixed with `JIRA_MCP_` (e.g., `JIRA_MCP_JIRA_URL`). **Highest precedence.**
2.  **Configuration File:** `config.yaml` (or `.json`, `.toml`) located within the `jira-mcp-server/` directory. See [`config.yaml.example`](./jira-mcp-server/config.yaml.example) for structure and all options.
3.  **Defaults:** Default values defined within the application code.

**Required Configuration:**

These values *must* be provided via environment variables or the config file:

*   `JIRA_MCP_JIRA_URL`: Your JIRA Cloud instance base URL (e.g., `https://your-domain.atlassian.net`).
*   `JIRA_MCP_JIRA_USER_EMAIL`: The email address of the JIRA user associated with the API token.
*   `JIRA_MCP_JIRA_API_TOKEN`: Your JIRA API token. **Treat this like a password!**

**Optional Configuration:**

*   `JIRA_MCP_PORT`: Port for the server to listen on (Default: `8080`).
*   `JIRA_MCP_LOG_LEVEL`: Logging level (`debug`, `info`, `warn`, `error`) (Default: `info`).
*   `JIRA_MCP_EPIC_LINK_FIELD_ID`: The custom field ID for the "Epic Link" in your JIRA instance (e.g., `customfield_10014`). **Required** for the `/jira_epic/{epicKey}/issues` endpoint to function correctly. Find this ID via your JIRA API or administration settings.

**Example (Environment Variables):**

```bash
export JIRA_MCP_JIRA_URL="https://your-domain.atlassian.net"
export JIRA_MCP_JIRA_USER_EMAIL="your.email@example.com"
export JIRA_MCP_JIRA_API_TOKEN="your-api-token-secret"
export JIRA_MCP_PORT="9000" # Optional
export JIRA_MCP_EPIC_LINK_FIELD_ID="customfield_10014" # Optional but needed for Epic endpoint
```

## ▶️ Running the Server

Ensure you are inside the `jira-mcp-server/` directory and have configured the required settings.

**Option 1: Run Directly (using Go)**

```bash
# Ensure required JIRA_MCP_... environment variables are set
make run
# Or: go run ./cmd/main.go
```
The server will start listening on the configured port (default `8080`).

**Option 2: Run with Docker**

1.  **Build the Docker image:**
    ```bash
    make docker-build
    ```
2.  **Create `.env` file:** Create a file named `.env` *inside the `jira-mcp-server/` directory* containing your `JIRA_MCP_` environment variables (one per line, e.g., `JIRA_MCP_JIRA_URL=...`).
    ```dotenv
    # .env file content example
    JIRA_MCP_JIRA_URL=https://your-domain.atlassian.net
    JIRA_MCP_JIRA_USER_EMAIL=your.email@example.com
    JIRA_MCP_JIRA_API_TOKEN=your-api-token-secret
    JIRA_MCP_PORT=8080 # Optional, ensure it matches docker-compose.yml mapping if changed
    JIRA_MCP_EPIC_LINK_FIELD_ID=customfield_10014 # Optional
    ```
3.  **Run using Docker Compose:**
    ```bash
    make docker-run
    # This command uses docker-compose up -d
    ```
    To stop the container: `docker-compose down`

## 🔌 API Endpoints

The server exposes the following primary endpoints:

*   `POST /create_jira_issue`: Creates a new JIRA issue.
*   `POST /search_jira_issues`: Searches for JIRA issues using JQL.
*   `GET /jira_issue/{issueKey}`: Retrieves details for a specific JIRA issue.
*   `GET /jira_epic/{epicKey}/issues`: Retrieves all issues belonging to a specific Epic (requires `JIRA_MCP_EPIC_LINK_FIELD_ID` configuration).

## Example Requests & Responses

For detailed request and response examples for each endpoint, please see:

➡️ **[`API_EXAMPLES.md`](./API_EXAMPLES.md)**

## ✅ Testing

The project includes both unit and integration tests. Run these commands from the `jira-mcp-server/` directory:

*   **Run Unit Tests:** Tests individual components in isolation (mocks JIRA API).
    ```bash
    make test
    ```
*   **Run Integration Tests:** Tests the HTTP API layer against a mock JIRA server.
    ```bash
    make test-integration
    ```
*   **View Unit Test Coverage:** Generates `coverage.html` and opens it in your browser.
    ```bash
    make coverage
    ```
*   **View Integration Test Coverage:** Generates `coverage-integration.html` and opens it in your browser.
    ```bash
    make coverage-integration
    ```

## 🏗️ Architecture

This server follows clean architecture principles, utilizing dependency injection (`wire`), structured logging (`slog`), and layered separation of concerns (handlers, JIRA client, core logic).

For a more detailed explanation, please see the **[Architecture Document](./jira-mcp-server/docs/architecture.md)**.

## 🤝 Contributing

Contributions are welcome and greatly appreciated! Whether it's reporting bugs, suggesting features, improving documentation, or submitting pull requests, your help makes this project better.

Please read our **[Contributing Guidelines](./CONTRIBUTING.md)** to get started.

Also, please note that this project is released with a **[Contributor Code of Conduct](./CODE_OF_CONDUCT.md)**. By participating in this project you agree to abide by its terms.

## 📜 License

This project is licensed under the MIT License. See the **[LICENSE](./jira-mcp-server/LICENSE)** file for details.

## ⚠️ Security Note

**Never commit your JIRA API token or other sensitive credentials directly into your code or configuration files in version control.** Use environment variables or a secure secrets management system, especially for production deployments. Ensure your `.env` file (if used for Docker) is included in your `.gitignore`.