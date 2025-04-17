# Contributing to jira-mcp-server

Thank you for considering contributing to the `jira-mcp-server` project! We welcome bug reports, feature suggestions, and pull requests. Please note that this project adheres to a [Code of Conduct](./CODE_OF_CONDUCT.md).

## Reporting Bugs

If you encounter a bug, please help us fix it by submitting an issue to our [GitHub Issues page](https://github.com/karolswdev/jira-mcp-server/issues). <!-- TODO: Replace with actual link when available -->

When reporting a bug, please include the following details:

*   **Steps to Reproduce:** Clear and concise steps to replicate the issue.
*   **Expected Behavior:** What you expected to happen.
*   **Actual Behavior:** What actually happened.
*   **Environment:** Details about your operating system, Go version, and any other relevant configuration.
*   **Logs:** Relevant log output, if applicable.

## Suggesting Enhancements

If you have an idea for a new feature or an improvement to an existing one, please submit an issue to our [GitHub Issues page](https://github.com/karolswdev/jira-mcp-server/issues). <!-- TODO: Replace with actual link when available -->

Clearly describe the enhancement:

*   **Motivation:** Why is this enhancement needed? What problem does it solve?
*   **Proposed Solution:** A clear description of the change you envision.
*   **Alternatives:** Any alternative solutions or features you've considered.

## Pull Request Process

We welcome contributions via Pull Requests (PRs). Here's how to submit one:

1.  **Fork the Repository:** Create your own fork of the `jira-mcp-server` repository on GitHub.
2.  **Create a Branch:** Create a new branch in your fork for your feature or bug fix (e.g., `git checkout -b feature/your-feature-name` or `git checkout -b fix/issue-123`).
3.  **Make Changes:** Implement your changes, adhering to the project's coding style and conventions.
4.  **Formatting:** Ensure your code is correctly formatted by running `make fmt` in the `jira-mcp-server/` directory.
5.  **Linting:** Ensure your code passes linting checks by running `make lint` in the `jira-mcp-server/` directory.
6.  **Testing:** Ensure all unit and integration tests pass by running `make test` and `make test-integration` in the `jira-mcp-server/` directory. Add new tests for your changes where appropriate.
7.  **Documentation:** Update the `README.md` or other documentation files if your changes require it.
8.  **Changelog:** Add an entry to the `CHANGELOG.md` file under the `[Unreleased]` section, briefly describing your change.
9.  **Commit Changes:** Commit your changes with a clear and descriptive commit message.
10. **Push Changes:** Push your branch to your fork on GitHub.
11. **Submit Pull Request:** Open a pull request from your branch to the `main` branch of the original `jira-mcp-server` repository.
12. **Describe PR:** Provide a clear title and description for your pull request, explaining the purpose and details of your changes. Link any relevant issues.

Your pull request will be reviewed, and we may provide feedback or request changes. Thank you for your contribution!