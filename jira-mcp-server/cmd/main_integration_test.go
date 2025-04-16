//go:build integration

package main_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog" // Added for slog
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"jira-mcp-server/internal/handlers"
	"jira-mcp-server/internal/jira"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestServer initializes a mock JIRA server and the MCP server configured to use it.
// It returns the MCP server instance, the mock JIRA server instance, and a cleanup function.
func setupTestServer(t *testing.T) (*httptest.Server, *httptest.Server, func()) {
	// Mock JIRA Server
	mockJira := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Default handler: return 404 for unmocked endpoints
		t.Logf("Mock JIRA received request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, `{"error": "Mock JIRA endpoint not implemented: %s %s"}`, r.Method, r.URL.Path)
	}))

	// Set environment variables for the MCP server to use the mock JIRA
	t.Setenv("JIRA_URL", mockJira.URL)
	t.Setenv("JIRA_USER_EMAIL", "test-user@example.com")
	t.Setenv("JIRA_API_TOKEN", "test-token")
	// Ensure PORT is not set, so httptest can assign a random one
	os.Unsetenv("PORT") // Use t.Setenv if needing to restore later, but unset is fine here

	// Initialize JIRA client (will use the overridden JIRA_URL)
	// Pass the mock server's client to ensure requests go to the mock
	jiraClient, err := jira.NewClient(mockJira.Client())
	require.NoError(t, err, "Failed to create JIRA client for test")

	// Initialize handlers
	testLogger := slog.New(slog.NewJSONHandler(io.Discard, nil)) // Discard logs in integration tests
	jiraHandlers := handlers.NewJiraHandlers(jiraClient, testLogger)

	// Set up router (mirroring main.go)
	router := mux.NewRouter()
	router.HandleFunc("/create_jira_issue", jiraHandlers.CreateJiraIssueHandler).Methods("POST")
	router.HandleFunc("/search_jira_issues", jiraHandlers.SearchIssuesHandler).Methods("POST")
	router.HandleFunc("/jira_issue/{issueKey}", jiraHandlers.GetIssueDetailsHandler).Methods("GET")
	router.HandleFunc("/jira_epic/{epicKey}/issues", jiraHandlers.GetIssuesInEpicHandler).Methods("GET")

	// MCP Server using the router and configured client
	mcpServer := httptest.NewServer(router)

	cleanup := func() {
		mcpServer.Close()
		mockJira.Close()
	}

	return mcpServer, mockJira, cleanup
}

// --- Test Cases ---

func TestIntegrationCreateIssue(t *testing.T) {
	mcpServer, mockJira, cleanup := setupTestServer(t)
	defer cleanup()

	// --- Success Case ---
	t.Run("Success", func(t *testing.T) {
		// Configure Mock JIRA for this specific test
		mockJira.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Logf("Mock JIRA received request: %s %s", r.Method, r.URL.Path)
			if r.Method == http.MethodPost && r.URL.Path == "/rest/api/3/issue" {
				// Basic Auth check (optional but good practice)
				user, pass, ok := r.BasicAuth()
				assert.True(t, ok, "Basic auth expected")
				assert.Equal(t, "test-user@example.com", user)
				assert.Equal(t, "test-token", pass)

				// Content-Type check
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				// Read body to verify payload (optional)
				bodyBytes, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				var reqBody map[string]interface{}
				err = json.Unmarshal(bodyBytes, &reqBody)
				require.NoError(t, err)
				assert.Contains(t, reqBody, "fields")
				fields, _ := reqBody["fields"].(map[string]interface{})
				assert.Contains(t, fields, "summary")
				assert.Equal(t, "Test Summary", fields["summary"])
				assert.Contains(t, fields, "project")
				project, _ := fields["project"].(map[string]interface{})
				assert.Contains(t, project, "key")
				assert.Equal(t, "PROJ", project["key"])
				assert.Contains(t, fields, "issuetype")
				issueType, _ := fields["issuetype"].(map[string]interface{})
				assert.Contains(t, issueType, "name")
				assert.Equal(t, "Task", issueType["name"])

				// Send mock success response
				w.WriteHeader(http.StatusCreated)
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprintln(w, `{"id": "10001", "key": "TEST-1", "self": "http://mock-jira/rest/api/3/issue/10001"}`)
				return
			}
			// Fallback for unexpected requests
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, `{"error": "Mock JIRA endpoint not implemented for this test: %s %s"}`, r.Method, r.URL.Path)
		})

		// Prepare request for MCP server
		createReqBody := map[string]interface{}{
			"project_key": "PROJ", // Use snake_case to match struct tag
			"summary":     "Test Summary",
			"description": "Test Description",
			"issue_type":  "Task", // Use snake_case to match struct tag
		}
		reqBytes, _ := json.Marshal(createReqBody)

		// Send request to MCP server
		req, err := http.NewRequest("POST", mcpServer.URL+"/create_jira_issue", bytes.NewBuffer(reqBytes))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := mcpServer.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Assert MCP server response
		assert.Equal(t, http.StatusCreated, resp.StatusCode)
		respBodyBytes, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		var respBody map[string]string
		err = json.Unmarshal(respBodyBytes, &respBody)
		require.NoError(t, err)
		assert.Equal(t, "TEST-1", respBody["key"]) // Check for "key" instead of "issueKey"
	})

	// --- Error Case (e.g., JIRA returns 400) ---
	t.Run("JiraError", func(t *testing.T) {
		// Configure Mock JIRA to return an error
		mockJira.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Logf("Mock JIRA received request: %s %s", r.Method, r.URL.Path)
			if r.Method == http.MethodPost && r.URL.Path == "/rest/api/3/issue" {
				w.WriteHeader(http.StatusBadRequest)
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprintln(w, `{"errorMessages": ["Project key 'INVALID' does not exist."], "errors": {}}`)
				return
			}
			w.WriteHeader(http.StatusNotFound)
		})

		// Prepare request for MCP server
		createReqBody := map[string]interface{}{
			"project_key": "INVALID", // Use snake_case to match struct tag
			"summary":     "Test Summary Error",
			"description": "Test Description",
			"issue_type":  "Task", // Use snake_case to match struct tag
		}
		reqBytes, _ := json.Marshal(createReqBody)

		// Send request to MCP server
		req, err := http.NewRequest("POST", mcpServer.URL+"/create_jira_issue", bytes.NewBuffer(reqBytes))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := mcpServer.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Assert MCP server response (should map the JIRA 400 to our 400 with a user-friendly message)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		respBodyBytes, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		// Check for the specific user-friendly JSON error message
		require.JSONEq(t, `{"error":"Invalid request data sent to JIRA."}`, string(respBodyBytes))
	})

	// --- Error Case (Bad MCP Request Body) ---
	t.Run("BadMCPRequest", func(t *testing.T) {
		// Mock JIRA handler doesn't matter here as the request should fail before reaching it
		mockJira.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Errorf("Mock JIRA should not have been called for a bad MCP request")
			w.WriteHeader(http.StatusInternalServerError)
		})

		// Send request with invalid JSON to MCP server
		req, err := http.NewRequest("POST", mcpServer.URL+"/create_jira_issue", strings.NewReader("{invalid json"))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := mcpServer.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Assert MCP server response
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		respBodyBytes, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		// Check for the specific user-friendly JSON error message for bad decoding
		// Check for the specific user-friendly JSON error message for bad decoding
		require.JSONEq(t, `{"error":"Invalid request body"}`, string(respBodyBytes))
	})
}

func TestIntegrationGetIssue(t *testing.T) {
	mcpServer, mockJira, cleanup := setupTestServer(t)
	defer cleanup()

	const testIssueKey = "TEST-123"
	const mockIssueID = "10050"

	// --- Success Case ---
	t.Run("Success", func(t *testing.T) {
		// Configure Mock JIRA
		mockJira.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Logf("Mock JIRA received request: %s %s", r.Method, r.URL.Path)
			expectedPath := fmt.Sprintf("/rest/api/3/issue/%s", testIssueKey)
			if r.Method == http.MethodGet && r.URL.Path == expectedPath {
				// Basic Auth check
				user, pass, ok := r.BasicAuth()
				assert.True(t, ok, "Basic auth expected")
				assert.Equal(t, "test-user@example.com", user)
				assert.Equal(t, "test-token", pass)

				// Send mock success response
				w.WriteHeader(http.StatusOK)
				w.Header().Set("Content-Type", "application/json")
				// Respond with a more detailed structure matching a real JIRA issue
				fmt.Fprintf(w, `{
					"id": "%s",
					"key": "%s",
					"self": "%s/rest/api/3/issue/%s",
					"fields": {
						"summary": "Integration Test Issue Summary",
						"description": {
							"type": "doc",
							"version": 1,
							"content": [
								{
									"type": "paragraph",
									"content": [
										{
											"type": "text",
											"text": "This is the description."
										}
									]
								}
							]
						},
						"status": {
							"name": "To Do"
						},
						"issuetype": {
							"name": "Task"
						},
						"project": {
							"key": "PROJ",
							"name": "Project Name"
						}
					}
				}`, mockIssueID, testIssueKey, mockJira.URL, mockIssueID)
				return
			}
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, `{"error": "Mock JIRA endpoint not implemented for this test: %s %s"}`, r.Method, r.URL.Path)
		})

		// Send request to MCP server
		req, err := http.NewRequest("GET", mcpServer.URL+"/jira_issue/"+testIssueKey, nil)
		require.NoError(t, err)

		resp, err := mcpServer.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Assert MCP server response
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		respBodyBytes, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		// Basic check for key fields in the response
		assert.Contains(t, string(respBodyBytes), `"key":"TEST-123"`)
		assert.Contains(t, string(respBodyBytes), `"summary":"Integration Test Issue Summary"`)
		assert.Contains(t, string(respBodyBytes), `"status":{"name":"To Do"}`) // Check nested field
	})

	// --- Error Case (JIRA returns 404 Not Found) ---
	t.Run("JiraNotFound", func(t *testing.T) {
		// Configure Mock JIRA to return 404
		mockJira.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Logf("Mock JIRA received request: %s %s", r.Method, r.URL.Path)
			expectedPath := fmt.Sprintf("/rest/api/3/issue/%s", "NONEXIST-1")
			if r.Method == http.MethodGet && r.URL.Path == expectedPath {
				w.WriteHeader(http.StatusNotFound)
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprintln(w, `{"errorMessages": ["Issue does not exist or you do not have permission to see it."], "errors": {}}`)
				return
			}
			w.WriteHeader(http.StatusNotFound) // Fallback
		})

		// Send request to MCP server for a non-existent issue
		req, err := http.NewRequest("GET", mcpServer.URL+"/jira_issue/NONEXIST-1", nil)
		require.NoError(t, err)

		resp, err := mcpServer.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Assert MCP server response (should map the JIRA 404 to our 404 with a user-friendly message)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		respBodyBytes, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		// Check for the specific user-friendly JSON error message
		require.JSONEq(t, `{"error":"JIRA resource not found."}`, string(respBodyBytes))
	})
}
func TestIntegrationSearchIssues(t *testing.T) {
	mcpServer, mockJira, cleanup := setupTestServer(t)
	defer cleanup()

	// --- Success Case ---
	t.Run("Success", func(t *testing.T) {
		// Configure Mock JIRA
		mockJira.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Logf("Mock JIRA received request: %s %s", r.Method, r.URL.Path)
			if r.Method == http.MethodPost && r.URL.Path == "/rest/api/3/search" {
				// Auth check
				user, pass, ok := r.BasicAuth()
				assert.True(t, ok)
				assert.Equal(t, "test-user@example.com", user)
				assert.Equal(t, "test-token", pass)
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				// Read body to verify JQL
				bodyBytes, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				var reqBody map[string]interface{}
				err = json.Unmarshal(bodyBytes, &reqBody)
				require.NoError(t, err)
				assert.Contains(t, reqBody, "jql")
				assert.Equal(t, "project = PROJ ORDER BY created DESC", reqBody["jql"])
				// Optionally check maxResults, fields etc.

				// Send mock success response
				w.WriteHeader(http.StatusOK)
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprintln(w, `{
					"expand": "names,schema",
					"startAt": 0,
					"maxResults": 50,
					"total": 1,
					"issues": [
						{
							"id": "10001",
							"key": "PROJ-1",
							"self": "http://mock-jira/rest/api/3/issue/10001",
							"fields": {
								"summary": "First issue",
								"status": {"name": "Done"}
							}
						}
					]
				}`)
				return
			}
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, `{"error": "Mock JIRA endpoint not implemented for this test: %s %s"}`, r.Method, r.URL.Path)
		})

		// Prepare request for MCP server
		searchReqBody := map[string]interface{}{
			"jql": "project = PROJ ORDER BY created DESC",
		}
		reqBytes, _ := json.Marshal(searchReqBody)

		// Send request to MCP server
		req, err := http.NewRequest("POST", mcpServer.URL+"/search_jira_issues", bytes.NewBuffer(reqBytes))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := mcpServer.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Assert MCP server response
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		respBodyBytes, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		// Basic check for key fields in the response
		assert.Contains(t, string(respBodyBytes), `"total":1`)
		assert.Contains(t, string(respBodyBytes), `"key":"PROJ-1"`)
		assert.Contains(t, string(respBodyBytes), `"summary":"First issue"`)
	})

	// --- Error Case (Invalid JQL - JIRA returns 400) ---
	t.Run("JiraInvalidJQL", func(t *testing.T) {
		// Configure Mock JIRA to return 400 for bad JQL
		mockJira.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Logf("Mock JIRA received request: %s %s", r.Method, r.URL.Path)
			if r.Method == http.MethodPost && r.URL.Path == "/rest/api/3/search" {
				w.WriteHeader(http.StatusBadRequest)
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprintln(w, `{"errorMessages": ["Error in the JQL Query: The character '%' is not valid."], "errors": {}}`)
				return
			}
			w.WriteHeader(http.StatusNotFound)
		})

		// Prepare request for MCP server with invalid JQL
		searchReqBody := map[string]interface{}{
			"jql": "project = %INVALID%",
		}
		reqBytes, _ := json.Marshal(searchReqBody)

		// Send request to MCP server
		req, err := http.NewRequest("POST", mcpServer.URL+"/search_jira_issues", bytes.NewBuffer(reqBytes))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := mcpServer.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Assert MCP server response (should map the JIRA 400 to our 400 with a user-friendly message)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		respBodyBytes, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		// Check for the specific user-friendly JSON error message
		require.JSONEq(t, `{"error":"Invalid request data sent to JIRA."}`, string(respBodyBytes))
	})

	// --- Error Case (Bad MCP Request Body) ---
	t.Run("BadMCPRequest", func(t *testing.T) {
		mockJira.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Errorf("Mock JIRA should not have been called")
			w.WriteHeader(http.StatusInternalServerError)
		})

		req, err := http.NewRequest("POST", mcpServer.URL+"/search_jira_issues", strings.NewReader("not json"))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := mcpServer.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		respBodyBytes, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		// Check for the specific user-friendly JSON error message for bad decoding
		// Check for the specific user-friendly JSON error message for bad decoding
		require.JSONEq(t, `{"error":"Invalid request body"}`, string(respBodyBytes))
	})
}
func TestIntegrationGetEpicIssues(t *testing.T) {
	mcpServer, mockJira, cleanup := setupTestServer(t)
	defer cleanup()

	const testEpicKey = "EPIC-1"

	// --- Success Case ---
	t.Run("Success", func(t *testing.T) {
		// Configure Mock JIRA to handle the search for issues in the epic
		mockJira.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Logf("Mock JIRA received request: %s %s", r.Method, r.URL.Path)
			if r.Method == http.MethodPost && r.URL.Path == "/rest/api/3/search" {
				// Auth check
				user, pass, ok := r.BasicAuth()
				assert.True(t, ok)
				assert.Equal(t, "test-user@example.com", user)
				assert.Equal(t, "test-token", pass)
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				// Read body to verify JQL
				bodyBytes, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				var reqBody map[string]interface{}
				err = json.Unmarshal(bodyBytes, &reqBody)
				require.NoError(t, err)
				assert.Contains(t, reqBody, "jql")
				// JIRA uses a custom field for Epic Link. The exact field ID can vary.
				// Common ones are 'customfield_10014', 'customfield_10008', etc.
				// For testing, we assume the client constructs the correct JQL.
				// A more robust mock might inspect the JQL more closely.
				assert.Contains(t, reqBody["jql"], fmt.Sprintf("'%s' = '%s'", jira.EpicLinkFieldName, testEpicKey), "JQL should filter by Epic Link field with single quotes")

				// Send mock success response
				w.WriteHeader(http.StatusOK)
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprintln(w, `{
					"expand": "names,schema",
					"startAt": 0,
					"maxResults": 50,
					"total": 2,
					"issues": [
						{
							"id": "10010",
							"key": "STORY-1",
							"fields": {"summary": "Story in Epic 1"}
						},
						{
							"id": "10011",
							"key": "STORY-2",
							"fields": {"summary": "Another Story in Epic 1"}
						}
					]
				}`)
				return
			}
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, `{"error": "Mock JIRA endpoint not implemented for this test: %s %s"}`, r.Method, r.URL.Path)
		})

		// Send request to MCP server
		req, err := http.NewRequest("GET", mcpServer.URL+"/jira_epic/"+testEpicKey+"/issues", nil)
		require.NoError(t, err)

		resp, err := mcpServer.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Assert MCP server response
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		respBodyBytes, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		// Check response details
		assert.Contains(t, string(respBodyBytes), `"total":2`)
		assert.Contains(t, string(respBodyBytes), `"key":"STORY-1"`)
		assert.Contains(t, string(respBodyBytes), `"key":"STORY-2"`)
	})

	// --- Error Case (JIRA returns error during search) ---
	t.Run("JiraSearchError", func(t *testing.T) {
		// Configure Mock JIRA to return an error
		mockJira.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Logf("Mock JIRA received request: %s %s", r.Method, r.URL.Path)
			if r.Method == http.MethodPost && r.URL.Path == "/rest/api/3/search" {
				w.WriteHeader(http.StatusInternalServerError) // Simulate a server error
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprintln(w, `{"errorMessages": ["Internal server error occurred."], "errors": {}}`)
				return
			}
			w.WriteHeader(http.StatusNotFound)
		})

		// Send request to MCP server
		req, err := http.NewRequest("GET", mcpServer.URL+"/jira_epic/"+testEpicKey+"/issues", nil)
		require.NoError(t, err)

		resp, err := mcpServer.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Assert MCP server response (should reflect the JIRA error)
		// Assert MCP server response (should map the JIRA 500 to our 500 with a user-friendly message)
		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
		respBodyBytes, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		// Check for the specific user-friendly JSON error message for unhandled JIRA errors
		require.JSONEq(t, `{"error":"An unexpected error occurred while communicating with JIRA."}`, string(respBodyBytes))
	})

	// --- Edge Case (Epic exists but has no issues) ---
	t.Run("NoIssuesInEpic", func(t *testing.T) {
		// Configure Mock JIRA to return an empty list
		mockJira.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Logf("Mock JIRA received request: %s %s", r.Method, r.URL.Path)
			if r.Method == http.MethodPost && r.URL.Path == "/rest/api/3/search" {
				w.WriteHeader(http.StatusOK)
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprintln(w, `{
					"expand": "names,schema",
					"startAt": 0,
					"maxResults": 50,
					"total": 0,
					"issues": []
				}`)
				return
			}
			w.WriteHeader(http.StatusNotFound)
		})

		// Send request to MCP server
		req, err := http.NewRequest("GET", mcpServer.URL+"/jira_epic/"+testEpicKey+"/issues", nil)
		require.NoError(t, err)

		resp, err := mcpServer.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Assert MCP server response
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		respBodyBytes, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Contains(t, string(respBodyBytes), `"total":0`)
		assert.Contains(t, string(respBodyBytes), `"issues":[]`)
	})
}
