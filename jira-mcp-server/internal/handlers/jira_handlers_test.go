package handlers

import (
	"context"
	"errors"
	"io"       // Added for io.Discard
	"log/slog" // Added for slog
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	// Added for error testing
	// Needed for path variables

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"jira-mcp-server/internal/jira" // Corrected import path based on go.mod
)

// mockJiraService is a mock implementation of the JiraService interface.
type mockJiraService struct {
	mock.Mock // Embed mock.Mock
}

func (m *mockJiraService) CreateIssue(ctx context.Context, issueData jira.CreateIssueRequest) (*jira.CreateIssueResponse, error) { // Corrected types
	args := m.Called(ctx, issueData)
	res, _ := args.Get(0).(*jira.CreateIssueResponse) // Corrected type, Allow nil return for error case
	return res, args.Error(1)
}

func (m *mockJiraService) SearchIssues(ctx context.Context, jql string, maxResults int, fields []string) (*jira.SearchResponse, error) { // Corrected signature to match interface
	args := m.Called(ctx, jql, maxResults, fields) // Corrected arguments
	res, _ := args.Get(0).(*jira.SearchResponse)   // Corrected type, Allow nil return for error case
	return res, args.Error(1)
}

func (m *mockJiraService) GetIssue(ctx context.Context, issueKey string, fields []string) (*jira.Issue, error) { // Corrected type
	args := m.Called(ctx, issueKey, fields)
	res, _ := args.Get(0).(*jira.Issue) // Corrected type, Allow nil return for error case
	return res, args.Error(1)
}

// GetEpicIssues removed as it's not part of the JiraService interface used by handlers

// --- Test Cases Start Here ---

// --- CreateJiraIssueHandler Tests ---

func TestCreateJiraIssueHandler_Success(t *testing.T) {
	mockService := new(mockJiraService)
	testLogger := slog.New(slog.NewJSONHandler(io.Discard, nil)) // Discard logs in tests
	handlers := NewJiraHandlers(mockService, testLogger)

	// Corrected reqBody JSON to match jira.CreateIssueRequest struct
	reqBody := `{"project_key": "PROJ", "summary": "Test Issue", "issue_type": "Task"}`
	req := httptest.NewRequest(http.MethodPost, "/create_jira_issue", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	// Use mock.Anything for context matching
	expectedReq := jira.CreateIssueRequest{ // Corrected type
		// NOTE: The test request body doesn't match the CreateIssueRequest struct fields.
		// Adjusting test data to match the struct definition.
		ProjectKey: "PROJ",
		Summary:    "Test Issue",
		IssueType:  "Task",
	}
	expectedResp := &jira.CreateIssueResponse{ // Corrected type
		// ID field removed as it doesn't exist in the struct
		Key:  "PROJ-123",
		Self: "http://jira.example.com/rest/api/2/issue/10001",
	}

	mockService.On("CreateIssue", mock.Anything, expectedReq).Return(expectedResp, nil) // Use mock.Anything for context

	handlers.CreateJiraIssueHandler(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)
	// Use require for fatal assertions on critical checks
	// Expect the actual map returned by the handler
	require.JSONEq(t, `{"message":"JIRA issue created successfully", "key":"PROJ-123", "url":"http://jira.example.com/rest/api/2/issue/10001"}`, rr.Body.String())
	mockService.AssertExpectations(t)
}

func TestCreateJiraIssueHandler_BadRequest_InvalidJSON(t *testing.T) {
	mockService := new(mockJiraService) // Service shouldn't be called
	testLogger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	handlers := NewJiraHandlers(mockService, testLogger)

	reqBody := `{"fields": {"project": {"key": "PROJ"}, "summary": "Test Issue", "issuetype": {"name": "Task"}}` // Invalid JSON
	req := httptest.NewRequest(http.MethodPost, "/create_jira_issue", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handlers.CreateJiraIssueHandler(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	// Check for the specific user-friendly JSON error message
	require.JSONEq(t, `{"error":"Invalid request body"}`, rr.Body.String())
	mockService.AssertNotCalled(t, "CreateIssue", mock.Anything, mock.Anything) // Verify service wasn't called
}

func TestCreateJiraIssueHandler_ServiceError(t *testing.T) {
	mockService := new(mockJiraService)
	testLogger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	handlers := NewJiraHandlers(mockService, testLogger)

	// Corrected reqBody JSON to match jira.CreateIssueRequest struct
	reqBody := `{"project_key": "PROJ", "summary": "Test Issue", "issue_type": "Task"}`
	req := httptest.NewRequest(http.MethodPost, "/create_jira_issue", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	expectedReq := jira.CreateIssueRequest{ // Corrected type
		// NOTE: Adjusting test data to match the struct definition.
		ProjectKey: "PROJ",
		Summary:    "Test Issue",
		IssueType:  "Task",
	}
	// Simulate a generic internal error first
	serviceErr := errors.New("some internal processing error")

	mockService.On("CreateIssue", mock.Anything, expectedReq).Return(nil, serviceErr)

	handlers.CreateJiraIssueHandler(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	// Check for the generic user-friendly message for non-JiraAPIErrors
	require.JSONEq(t, `{"error":"An internal server error occurred."}`, rr.Body.String())
	mockService.AssertExpectations(t)
}

func TestCreateJiraIssueHandler_ServiceError_JiraBadRequest(t *testing.T) {
	mockService := new(mockJiraService)
	testLogger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	handlers := NewJiraHandlers(mockService, testLogger)

	reqBody := `{"project_key": "PROJ", "summary": "Bad Data", "issue_type": "Bug"}`
	req := httptest.NewRequest(http.MethodPost, "/create_jira_issue", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	expectedReq := jira.CreateIssueRequest{
		ProjectKey: "PROJ",
		Summary:    "Bad Data",
		IssueType:  "Bug",
	}
	// Simulate a JIRA API 400 Bad Request error
	serviceErr := &jira.JiraAPIError{
		StatusCode: http.StatusBadRequest,
		Message:    `{"errorMessages":["Field 'priority' is required."],"errors":{}}`, // Example JIRA error body
		URL:        "http://jira.example.com/rest/api/3/issue",
	}

	mockService.On("CreateIssue", mock.Anything, expectedReq).Return(nil, serviceErr)

	handlers.CreateJiraIssueHandler(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	// Check for the specific user-friendly message mapped from 400
	require.JSONEq(t, `{"error":"Invalid request data sent to JIRA."}`, rr.Body.String())
	mockService.AssertExpectations(t)
}

// --- SearchJiraIssuesHandler Tests ---

func TestSearchJiraIssuesHandler_Success(t *testing.T) {
	mockService := new(mockJiraService)
	testLogger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	handlers := NewJiraHandlers(mockService, testLogger)

	// Handler expects POST with JSON body
	reqBody := `{"jql": "project=PROJ ORDER BY created DESC", "maxResults": 10, "fields": ["summary", "status"]}`
	req := httptest.NewRequest(http.MethodPost, "/search_jira_issues", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	expectedJQL := "project=PROJ ORDER BY created DESC"
	expectedMaxResults := 10
	expectedFields := []string{"summary", "status"}
	expectedResp := &jira.SearchResponse{
		StartAt:    0,
		MaxResults: 10,
		Total:      1,
		Issues: []jira.Issue{
			{
				Key:  "PROJ-1",
				Self: "http://jira.example.com/rest/api/2/issue/10000",
				Fields: map[string]interface{}{
					"summary": "First issue",
					"status":  map[string]interface{}{"name": "To Do"},
				},
			},
		},
	}

	mockService.On("SearchIssues", mock.Anything, expectedJQL, expectedMaxResults, expectedFields).Return(expectedResp, nil) // Use mock.Anything for context

	handlers.SearchIssuesHandler(rr, req) // Corrected method name

	assert.Equal(t, http.StatusOK, rr.Code)
	// Using require.JSONEq for better diffs on failure
	require.JSONEq(t, `{"expand":"","startAt":0,"maxResults":10,"total":1,"issues":[{"expand":"","id":"","key":"PROJ-1","self":"http://jira.example.com/rest/api/2/issue/10000","fields":{"summary":"First issue","status":{"name":"To Do"}}}]}`, rr.Body.String())
	mockService.AssertExpectations(t)
}

func TestSearchJiraIssuesHandler_BadRequest_MissingJQL(t *testing.T) {
	mockService := new(mockJiraService)
	testLogger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	handlers := NewJiraHandlers(mockService, testLogger)

	// Handler expects POST with JSON body, send body missing 'jql'
	reqBody := `{"maxResults": 10}`
	req := httptest.NewRequest(http.MethodPost, "/search_jira_issues", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handlers.SearchIssuesHandler(rr, req) // Corrected method name

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "Missing required field: jql") // Match handler's error message
	mockService.AssertNotCalled(t, "SearchIssues", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestSearchJiraIssuesHandler_ServiceError(t *testing.T) {
	mockService := new(mockJiraService)
	testLogger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	handlers := NewJiraHandlers(mockService, testLogger)

	// Handler expects POST with JSON body
	expectedJQL := "project=PROJ"
	reqBody := `{"jql": "` + expectedJQL + `"}` // Only JQL provided, handler defaults others
	req := httptest.NewRequest(http.MethodPost, "/search_jira_issues", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	// Default maxResults is used when not provided
	// Default fields (empty slice) is used when not provided
	// Simulate a JIRA API 401 Unauthorized error
	serviceErr := &jira.JiraAPIError{
		StatusCode: http.StatusUnauthorized,
		Message:    "Client must be authenticated to access this resource.",
		URL:        "http://jira.example.com/rest/api/3/search",
	}

	mockService.On("SearchIssues", mock.Anything, expectedJQL, 50, []string(nil)).Return(nil, serviceErr)

	handlers.SearchIssuesHandler(rr, req) // Corrected method name

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	// Check for the specific user-friendly message mapped from 401
	require.JSONEq(t, `{"error":"Authentication failed with JIRA."}`, rr.Body.String())
	mockService.AssertExpectations(t)
}

// --- GetIssueDetailsHandler Tests ---

func TestGetIssueDetailsHandler_Success(t *testing.T) {
	mockService := new(mockJiraService)
	testLogger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	handlers := NewJiraHandlers(mockService, testLogger)

	issueKey := "PROJ-456"
	expectedFields := []string{"summary", "status"}
	req := httptest.NewRequest(http.MethodGet, "/jira_issue/"+issueKey+"?fields=summary,status", nil)
	rr := httptest.NewRecorder()

	// Simulate gorilla/mux path variables
	req = mux.SetURLVars(req, map[string]string{"issueKey": issueKey})

	expectedResp := &jira.Issue{
		Key:  issueKey,
		Self: "http://jira.example.com/rest/api/2/issue/" + issueKey,
		Fields: map[string]interface{}{
			"summary": "Another test issue",
			"status":  map[string]interface{}{"name": "In Progress"},
		},
	}

	mockService.On("GetIssue", mock.Anything, issueKey, expectedFields).Return(expectedResp, nil) // Use mock.Anything for context

	handlers.GetIssueDetailsHandler(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	require.JSONEq(t, `{"expand":"","id":"","key":"PROJ-456","self":"http://jira.example.com/rest/api/2/issue/PROJ-456","fields":{"summary":"Another test issue","status":{"name":"In Progress"}}}`, rr.Body.String())
	mockService.AssertExpectations(t)
}

func TestGetIssueDetailsHandler_BadRequest_MissingKey(t *testing.T) {
	mockService := new(mockJiraService)
	testLogger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	handlers := NewJiraHandlers(mockService, testLogger)

	// Request without setting the mux var
	req := httptest.NewRequest(http.MethodGet, "/jira_issue/", nil) // Path might differ based on router setup, assuming mux handles empty var
	rr := httptest.NewRecorder()

	// Simulate gorilla/mux path variables with empty key
	req = mux.SetURLVars(req, map[string]string{"issueKey": ""})

	handlers.GetIssueDetailsHandler(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "Missing issue key in URL path")
	mockService.AssertNotCalled(t, "GetIssue", mock.Anything, mock.Anything, mock.Anything)
}

func TestGetIssueDetailsHandler_ServiceError(t *testing.T) {
	mockService := new(mockJiraService)
	testLogger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	handlers := NewJiraHandlers(mockService, testLogger)

	issueKey := "PROJ-789"
	// Test without specific fields
	req := httptest.NewRequest(http.MethodGet, "/jira_issue/"+issueKey, nil)
	rr := httptest.NewRecorder()

	// Simulate gorilla/mux path variables
	req = mux.SetURLVars(req, map[string]string{"issueKey": issueKey})

	// Simulate a JIRA API 404 Not Found error
	serviceErr := &jira.JiraAPIError{
		StatusCode: http.StatusNotFound,
		Message:    `{"errorMessages":["Issue does not exist or you do not have permission to see it."],"errors":{}}`,
		URL:        "http://jira.example.com/rest/api/3/issue/" + issueKey,
	}

	// Expect call with empty fields slice when query param is absent
	mockService.On("GetIssue", mock.Anything, issueKey, []string(nil)).Return(nil, serviceErr)

	handlers.GetIssueDetailsHandler(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	// Check for the specific user-friendly message mapped from 404
	require.JSONEq(t, `{"error":"JIRA resource not found."}`, rr.Body.String())
	mockService.AssertExpectations(t)
}

// --- GetIssuesInEpicHandler Tests ---

func TestGetIssuesInEpicHandler_Success(t *testing.T) {
	mockService := new(mockJiraService)
	testLogger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	handlers := NewJiraHandlers(mockService, testLogger)

	epicKey := "EPIC-1"
	// The handler constructs this specific JQL
	expectedJQL := `'customfield_10014' = 'EPIC-1'` // Corrected JQL based on handler implementation
	// The handler uses default maxResults (50) and fields ([])
	expectedMaxResults := 50
	// expectedFields := []string{} // Removed as it's unused now

	req := httptest.NewRequest(http.MethodGet, "/jira_epic/"+epicKey+"/issues", nil)
	rr := httptest.NewRecorder()

	// Simulate gorilla/mux path variables
	req = mux.SetURLVars(req, map[string]string{"epicKey": epicKey})

	expectedResp := &jira.SearchResponse{
		StartAt:    0,
		MaxResults: 50,
		Total:      1,
		Issues: []jira.Issue{
			{
				Key:  "STORY-101",
				Self: "http://jira.example.com/rest/api/2/issue/10101",
				Fields: map[string]interface{}{
					"summary": "Story within the epic",
				},
			},
		},
	}

	mockService.On("SearchIssues", mock.Anything, expectedJQL, expectedMaxResults, []string(nil)).Return(expectedResp, nil) // Expect nil slice for default fields, corrected JQL

	handlers.GetIssuesInEpicHandler(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	require.JSONEq(t, `{"expand":"","startAt":0,"maxResults":50,"total":1,"issues":[{"expand":"","id":"","key":"STORY-101","self":"http://jira.example.com/rest/api/2/issue/10101","fields":{"summary":"Story within the epic"}}]}`, rr.Body.String())
	mockService.AssertExpectations(t)
}

func TestGetIssuesInEpicHandler_BadRequest_MissingKey(t *testing.T) {
	mockService := new(mockJiraService)
	testLogger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	handlers := NewJiraHandlers(mockService, testLogger)

	// Request without setting the mux var
	req := httptest.NewRequest(http.MethodGet, "/jira_epic//issues", nil)
	rr := httptest.NewRecorder()

	// Simulate gorilla/mux path variables with empty key
	req = mux.SetURLVars(req, map[string]string{"epicKey": ""})

	handlers.GetIssuesInEpicHandler(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "Missing epic key in URL path")
	mockService.AssertNotCalled(t, "SearchIssues", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestGetIssuesInEpicHandler_ServiceError(t *testing.T) {
	mockService := new(mockJiraService)
	testLogger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	handlers := NewJiraHandlers(mockService, testLogger)

	epicKey := "EPIC-FAIL"
	expectedJQL := `'customfield_10014' = 'EPIC-FAIL'` // Corrected JQL based on handler implementation
	expectedMaxResults := 50
	// expectedFields := []string{} // Removed as it's unused now

	req := httptest.NewRequest(http.MethodGet, "/jira_epic/"+epicKey+"/issues", nil)
	rr := httptest.NewRecorder()

	// Simulate gorilla/mux path variables
	req = mux.SetURLVars(req, map[string]string{"epicKey": epicKey})

	// Simulate a JIRA API 403 Forbidden error (via SearchIssues)
	serviceErr := &jira.JiraAPIError{
		StatusCode: http.StatusForbidden,
		Message:    "User does not have permission to perform this operation.",
		URL:        "http://jira.example.com/rest/api/3/search",
	}

	mockService.On("SearchIssues", mock.Anything, expectedJQL, expectedMaxResults, []string(nil)).Return(nil, serviceErr)

	handlers.GetIssuesInEpicHandler(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
	// Check for the specific user-friendly message mapped from 403
	require.JSONEq(t, `{"error":"Permission denied by JIRA."}`, rr.Body.String())
	mockService.AssertExpectations(t)
}
