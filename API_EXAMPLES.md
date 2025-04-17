# JIRA MCP Server - API Examples

This document provides detailed examples for interacting with the JIRA MCP Server API endpoints. For general information about the server, see the main [`README.md`](./README.md).

## `/create_jira_issue` (POST)

Creates a new JIRA issue.

**Request:**

```bash
curl -X POST http://localhost:8080/create_jira_issue \
  -H "Content-Type: application/json" \
  -d '{
    "project_key": "PROJ",
    "summary": "Implement new feature via API",
    "issue_type": "Story",
    "description": "This is a detailed description of the story created through the MCP server.",
    "assignee_email": "user@example.com", # Optional
    "parent_key": "PROJ-100" # Optional, for sub-tasks
  }'
```

**Success Response (201 Created):**

```json
{
  "message": "JIRA issue created successfully",
  "key": "PROJ-123",
  "url": "https://your-domain.atlassian.net/browse/PROJ-123"
}
```

**Error Response (e.g., 400 Bad Request):**

```json
{
  "error": "Invalid request body: Missing required field 'project_key'"
}
```

## `/search_jira_issues` (POST)

Searches for JIRA issues using JIRA Query Language (JQL).

**Request:**

```bash
curl -X POST http://localhost:8080/search_jira_issues \
  -H "Content-Type: application/json" \
  -d '{
    "jql": "project = PROJ AND status = \"To Do\" ORDER BY created DESC",
    "max_results": 5,
    "fields": ["summary", "status", "assignee"]
  }'
```

**Success Response (200 OK):**

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

**Error Response (e.g., 400 Bad Request):**

```json
{
  "error": "Invalid JQL query: ..."
}
```

## `/jira_issue/{issueKey}` (GET)

Retrieves details for a specific JIRA issue.

**Request:**

```bash
# Get specific fields
curl -X GET "http://localhost:8080/jira_issue/PROJ-123?fields=summary,status,issuetype"

# Get default fields
curl -X GET "http://localhost:8080/jira_issue/PROJ-123"
```

**Success Response (200 OK):**

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
    // ... other requested or default fields ...
  }
}
```

**Error Response (e.g., 404 Not Found):**

```json
{
  "error": "Issue not found: PROJ-999"
}
```

## `/jira_epic/{epicKey}/issues` (GET)

Retrieves all issues belonging to a specific Epic. Requires the `JIRA_MCP_EPIC_LINK_FIELD_ID` configuration to be set correctly.

**Request:**

```bash
curl -X GET http://localhost:8080/jira_epic/PROJ-456/issues
```

**Success Response (200 OK):**

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

**Error Response (e.g., 404 Not Found or 500 Internal Server Error if Epic Link Field ID is wrong):**

```json
{
  "error": "Epic not found or error retrieving issues: PROJ-999"
}