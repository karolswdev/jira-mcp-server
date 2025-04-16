package jira_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"jira-mcp-server/internal/jira"
)

// Helper function to create a mock JIRA server
func setupTestServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *jira.Client) {
	t.Helper()
	server := httptest.NewServer(handler)

	// Create a client configured to talk to the test server
	// Note: We pass server.Client() to ensure the client uses the test server's transport.
	// We also need to provide dummy credentials, though they won't be validated by the mock server.
	t.Setenv("JIRA_URL", server.URL)
	t.Setenv("JIRA_USER_EMAIL", "test@example.com")
	t.Setenv("JIRA_API_TOKEN", "test-token")

	client, err := jira.NewClient(server.Client())
	require.NoError(t, err, "Failed to create test JIRA client")

	return server, client
}

func TestClient_CreateIssue(t *testing.T) {
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		expectedReqBody := `{
			"fields": {
				"project": { "key": "TEST" },
				"summary": "Test Summary",
				"issuetype": { "name": "Task" },
				"description": {
					"type": "doc",
					"version": 1,
					"content": [
						{
							"type": "paragraph",
							"content": [
								{
									"type": "text",
									"text": "Test Desc"
								}
							]
						}
					]
				}
			}
		}`
		mockResponse := jira.CreateIssueResponse{Key: "TEST-123", Self: "http://fakejira.com/rest/api/3/issue/TEST-123"}
		mockRespBody, _ := json.Marshal(mockResponse)

		handler := func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method, "Expected POST method")
			assert.Equal(t, "/rest/api/3/issue", r.URL.Path, "Expected correct API path")
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"), "Expected Content-Type header")
			assert.Equal(t, "application/json", r.Header.Get("Accept"), "Expected Accept header")
			authHeader := r.Header.Get("Authorization")
			assert.NotEmpty(t, authHeader, "Expected Authorization header")
			assert.True(t, strings.HasPrefix(authHeader, "Basic "), "Expected Basic auth scheme")

			bodyBytes, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			assert.JSONEq(t, expectedReqBody, string(bodyBytes), "Request body mismatch")

			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write(mockRespBody)
		}

		server, client := setupTestServer(t, handler)
		defer server.Close()

		req := jira.CreateIssueRequest{
			ProjectKey:  "TEST",
			Summary:     "Test Summary",
			IssueType:   "Task",
			Description: "Test Desc",
		}

		resp, err := client.CreateIssue(ctx, req)

		require.NoError(t, err, "CreateIssue should not return an error on success")
		require.NotNil(t, resp, "Response should not be nil on success")
		assert.Equal(t, mockResponse.Key, resp.Key)
		assert.Equal(t, mockResponse.Self, resp.Self)
	})

	t.Run("Error 400 Bad Request", func(t *testing.T) {
		mockErrorResp := `{"errorMessages":["Request validation failed"],"errors":{}}`
		handler := func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "/rest/api/3/issue", r.URL.Path)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(mockErrorResp))
		}

		server, client := setupTestServer(t, handler)
		defer server.Close()

		// Use a valid request structure but expect the server to reject it
		req := jira.CreateIssueRequest{
			ProjectKey: "TEST",
			Summary:    "Valid Summary",
			IssueType:  "Task",
		}

		resp, err := client.CreateIssue(ctx, req)

		require.Error(t, err, "CreateIssue should return an error on 400")
		require.Nil(t, resp, "Response should be nil on error")

		// Check if the error is the expected JiraAPIError type
		var jiraErr *jira.JiraAPIError
		require.ErrorAs(t, err, &jiraErr, "Error should be a JiraAPIError")
		assert.Equal(t, http.StatusBadRequest, jiraErr.StatusCode, "Status code should be 400")
		assert.Contains(t, jiraErr.Message, "Request validation failed", "Error message should contain JIRA error body")
		assert.Contains(t, jiraErr.Error(), "JIRA API error: status 400", "Formatted error string should contain status")
	})

	t.Run("Error Missing Required Fields Client Side", func(t *testing.T) {
		// No server needed as validation happens client-side
		t.Setenv("JIRA_URL", "http://dummy.com")
		t.Setenv("JIRA_USER_EMAIL", "test@example.com")
		t.Setenv("JIRA_API_TOKEN", "test-token")
		client, err := jira.NewClient(nil)
		require.NoError(t, err)

		req := jira.CreateIssueRequest{
			ProjectKey: "", // Missing project key
			Summary:    "Test Summary",
			IssueType:  "Task",
		}

		resp, err := client.CreateIssue(ctx, req)
		require.Error(t, err, "CreateIssue should return an error for missing fields")
		require.Nil(t, resp, "Response should be nil on validation error")
		assert.Contains(t, err.Error(), "project_key, summary, and issue_type are required")
	})
}

func TestClient_SearchIssues(t *testing.T) {
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		expectedJQL := "project = TEST AND status = Done"
		expectedMaxResults := 50
		expectedFields := []string{"summary", "status"}
		expectedReqBody := fmt.Sprintf(`{"fields":["summary","status"],"jql":"%s","maxResults":%d}`, expectedJQL, expectedMaxResults)

		mockResponse := jira.SearchResponse{
			StartAt:    0,
			MaxResults: expectedMaxResults,
			Total:      1,
			Issues: []jira.Issue{
				{
					Key: "TEST-1",
					Fields: map[string]interface{}{
						"summary": "Found issue",
						"status":  map[string]interface{}{"name": "Done"},
					},
				},
			},
		}
		mockRespBody, _ := json.Marshal(mockResponse)

		handler := func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "/rest/api/3/search", r.URL.Path)
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
			assert.Equal(t, "application/json", r.Header.Get("Accept"))
			assert.NotEmpty(t, r.Header.Get("Authorization"))

			bodyBytes, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			assert.JSONEq(t, expectedReqBody, string(bodyBytes))

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(mockRespBody)
		}

		server, client := setupTestServer(t, handler)
		defer server.Close()

		resp, err := client.SearchIssues(ctx, expectedJQL, expectedMaxResults, expectedFields)

		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, mockResponse.Total, resp.Total)
		assert.Equal(t, mockResponse.MaxResults, resp.MaxResults)
		require.Len(t, resp.Issues, 1)
		assert.Equal(t, "TEST-1", resp.Issues[0].Key)
		assert.Equal(t, "Found issue", resp.Issues[0].Fields["summary"])
	})

	t.Run("Error 401 Unauthorized", func(t *testing.T) {
		handler := func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "/rest/api/3/search", r.URL.Path)
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"errorMessages":["Authentication failed"]}`))
		}

		server, client := setupTestServer(t, handler)
		defer server.Close()

		resp, err := client.SearchIssues(ctx, "project = TEST", 10, nil)

		require.Error(t, err)
		require.Nil(t, resp)

		// Check if the error is the expected JiraAPIError type
		var jiraErr *jira.JiraAPIError
		require.ErrorAs(t, err, &jiraErr, "Error should be a JiraAPIError")
		assert.Equal(t, http.StatusUnauthorized, jiraErr.StatusCode, "Status code should be 401")
		assert.Contains(t, jiraErr.Message, "Authentication failed", "Error message should contain JIRA error body")
		assert.Contains(t, jiraErr.Error(), "JIRA API error: status 401", "Formatted error string should contain status")
	})

	t.Run("Error Empty JQL", func(t *testing.T) {
		// No server needed
		t.Setenv("JIRA_URL", "http://dummy.com")
		t.Setenv("JIRA_USER_EMAIL", "test@example.com")
		t.Setenv("JIRA_API_TOKEN", "test-token")
		client, err := jira.NewClient(nil)
		require.NoError(t, err)

		resp, err := client.SearchIssues(ctx, "", 10, nil)
		require.Error(t, err)
		require.Nil(t, resp)
		assert.Contains(t, err.Error(), "JQL query cannot be empty")
	})
}

func TestClient_GetIssue(t *testing.T) {
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		issueKey := "TEST-456"
		expectedFields := []string{"summary", "status", "assignee"}
		expectedURL := fmt.Sprintf("/rest/api/3/issue/%s?fields=summary,status,assignee", issueKey)

		mockResponse := jira.Issue{
			Key: issueKey,
			Fields: map[string]interface{}{
				"summary": "Specific Issue",
				"status":  map[string]interface{}{"name": "In Progress"},
				"assignee": map[string]interface{}{
					"displayName":  "Test User",
					"emailAddress": "test@example.com",
				},
			},
		}
		mockRespBody, _ := json.Marshal(mockResponse)

		handler := func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "GET", r.Method)
			assert.Equal(t, expectedURL, r.URL.RequestURI()) // Check path and query params
			assert.Equal(t, "application/json", r.Header.Get("Accept"))
			assert.NotEmpty(t, r.Header.Get("Authorization"))

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(mockRespBody)
		}

		server, client := setupTestServer(t, handler)
		defer server.Close()

		resp, err := client.GetIssue(ctx, issueKey, expectedFields)

		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, mockResponse.Key, resp.Key)
		assert.Equal(t, "Specific Issue", resp.Fields["summary"])
		statusMap, ok := resp.Fields["status"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "In Progress", statusMap["name"])
	})

	t.Run("Success No Fields", func(t *testing.T) {
		issueKey := "TEST-789"
		expectedURL := fmt.Sprintf("/rest/api/3/issue/%s", issueKey) // No fields param

		mockResponse := jira.Issue{Key: issueKey, Fields: map[string]interface{}{"summary": "Default Fields"}}
		mockRespBody, _ := json.Marshal(mockResponse)

		handler := func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "GET", r.Method)
			assert.Equal(t, expectedURL, r.URL.RequestURI())
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(mockRespBody)
		}

		server, client := setupTestServer(t, handler)
		defer server.Close()

		resp, err := client.GetIssue(ctx, issueKey, nil) // Pass nil for fields

		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, mockResponse.Key, resp.Key)
	})

	t.Run("Error 404 Not Found", func(t *testing.T) {
		issueKey := "NOTFOUND-1"
		handler := func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "GET", r.Method)
			assert.True(t, strings.HasPrefix(r.URL.Path, "/rest/api/3/issue/"))
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"errorMessages":["Issue does not exist or you do not have permission to see it."]}`))
		}

		server, client := setupTestServer(t, handler)
		defer server.Close()

		resp, err := client.GetIssue(ctx, issueKey, nil)

		require.Error(t, err)
		require.Nil(t, resp)

		// Check if the error is the expected JiraAPIError type
		var jiraErr *jira.JiraAPIError
		require.ErrorAs(t, err, &jiraErr, "Error should be a JiraAPIError")
		assert.Equal(t, http.StatusNotFound, jiraErr.StatusCode, "Status code should be 404")
		assert.Contains(t, jiraErr.Message, "Issue does not exist", "Error message should contain JIRA error body")
		assert.Contains(t, jiraErr.Error(), "JIRA API error: status 404", "Formatted error string should contain status")
	})

	t.Run("Error Empty Issue Key", func(t *testing.T) {
		// No server needed
		t.Setenv("JIRA_URL", "http://dummy.com")
		t.Setenv("JIRA_USER_EMAIL", "test@example.com")
		t.Setenv("JIRA_API_TOKEN", "test-token")
		client, err := jira.NewClient(nil)
		require.NoError(t, err)

		resp, err := client.GetIssue(ctx, "", nil)
		require.Error(t, err)
		require.Nil(t, resp)
		assert.Contains(t, err.Error(), "issue key cannot be empty")
	})
}

// Note: GetEpicIssues is not implemented in client.go, so no tests for it yet.
