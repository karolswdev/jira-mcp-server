package handlers

import (
	"context" // Added for request context
	"encoding/json"
	"errors" // Added for errors.As
	"fmt"
	"log/slog" // Added for structured logging
	"net/http"
	"strings"

	// "strconv" // No longer needed for parsing error string
	// "strings" // No longer needed for parsing error string

	"jira-mcp-server/internal/jira"

	"github.com/gorilla/mux" // Added for path parameter extraction
)

// JiraService defines the interface for interacting with the JIRA service.
// This allows for mocking in unit tests.
// It's defined here to avoid circular dependencies with the jira package.
type JiraService interface {
	CreateIssue(ctx context.Context, req jira.CreateIssueRequest) (*jira.CreateIssueResponse, error)
	SearchIssues(ctx context.Context, jql string, maxResults int, fields []string) (*jira.SearchResponse, error)
	GetIssue(ctx context.Context, issueKey string, fields []string) (*jira.Issue, error)
	// GetEpicIssues is implicitly covered by SearchIssues
}

// JiraHandlers holds dependencies for JIRA related HTTP handlers.
type JiraHandlers struct {
	JiraSvc jira.JiraService
	// JiraHandlers holds dependencies for JIRA related HTTP handlers, such as the
	// JiraService implementation and a structured logger.

	Logger *slog.Logger // Added logger field
}

// NewJiraHandlers creates a new JiraHandlers instance.
func NewJiraHandlers(service jira.JiraService, logger *slog.Logger) *JiraHandlers {
	return &JiraHandlers{
		// NewJiraHandlers creates a new JiraHandlers instance with the provided JiraService
		// implementation and structured logger.

		JiraSvc: service,
		Logger:  logger, // Assign logger
	}
}

func (h *JiraHandlers) CreateJiraIssueHandler(w http.ResponseWriter, r *http.Request) {
	h.Logger.Info("Request received", "method", r.Method, "path", r.URL.Path)
	if r.Method != http.MethodPost {
		// CreateJiraIssueHandler handles POST requests to /create_jira_issue.
		// It parses the request body, calls the JiraService's CreateIssue method,
		// and returns the created issue's key and URL or an error response.

		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	var req jira.CreateIssueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.Logger.Error("Failed to decode request body", "error", err)
		// Use the helper for consistent JSON error responses
		respondWithError(w, http.StatusBadRequest, "Invalid request body") // Keep user message generic
		return
	}

	// Get context from request
	ctx := r.Context()
	// Create issue
	resp, err := h.JiraSvc.CreateIssue(ctx, req)
	if err != nil {
		statusCode, userMessage := mapJiraError(err)
		// Log the detailed error internally
		h.Logger.Error("Error creating JIRA issue", "error", err)
		respondWithError(w, statusCode, userMessage) // Use user-friendly message
		return
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(w).Encode(map[string]string{
		"message": "JIRA issue created successfully",
		"key":     resp.Key,
		"url":     resp.Self,
	})
	if err != nil {
		// Log error, but can't change header after WriteHeader
		h.Logger.Error("Error encoding success response", "error", err)
	}
}

// Helper struct for SearchIssuesHandler request body
type SearchRequest struct {
	JQL string `json:"jql"`
	// SearchRequest defines the expected JSON structure for the request body
	// of the SearchIssuesHandler.

	MaxResults int      `json:"maxResults"`
	Fields     []string `json:"fields"`
}

// Helper function to write JSON error responses
func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

// Helper function to write JSON success responses
func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if payload != nil {
		if err := json.NewEncoder(w).Encode(payload); err != nil {
			// Log the encoding error, but we can't write another header
			// Note: Can't use the injected logger here as it's a helper function.
			// Using the default slog logger instead.
			slog.Error("Error encoding JSON response", "error", err)
		}
	}
}

// mapJiraError maps errors from the JiraService (especially JiraAPIErrors)
// to an appropriate HTTP status code and a user-friendly error message.
func mapJiraError(err error) (int, string) {
	if err == nil {
		return http.StatusOK, "" // Should not happen if called on non-nil error
	}

	var jiraAPIError *jira.JiraAPIError
	if errors.As(err, &jiraAPIError) {
		// We have a specific error from the JIRA API client
		switch jiraAPIError.StatusCode {
		case http.StatusBadRequest: // 400
			// Consider parsing jiraAPIError.Message for more specific user feedback if safe
			return http.StatusBadRequest, "Invalid request data sent to JIRA."
		case http.StatusUnauthorized: // 401
			return http.StatusUnauthorized, "Authentication failed with JIRA."
		case http.StatusForbidden: // 403
			return http.StatusForbidden, "Permission denied by JIRA."
		case http.StatusNotFound: // 404
			return http.StatusNotFound, "JIRA resource not found."
		default:
			// Log the detailed error internally
			// Note: Can't use the injected logger here as it's a helper function.
			// Using the default slog logger instead.
			slog.Error("Unhandled JIRA API Error", "status_code", jiraAPIError.StatusCode, "message", jiraAPIError.Message, "original_error", err)
			// For other 4xx or 5xx errors from JIRA, return a generic server error
			return http.StatusInternalServerError, "An unexpected error occurred while communicating with JIRA."
		}
	} else {
		// Check for specific client-side validation errors before defaulting
		// Example: Check for errors defined within the client package itself
		// if errors.Is(err, someSpecificClientValidationError) {
		// 	return http.StatusBadRequest, "Invalid input: specific reason."
		// }

		// Log the detailed error internally
		// Note: Can't use the injected logger here as it's a helper function.
		// Using the default slog logger instead.
		slog.Error("Internal Server Error (non-JIRA API)", "error", err)
		// Default for non-JiraAPIError types (e.g., network issues, internal validation)
		return http.StatusInternalServerError, "An internal server error occurred."
	}
}

// SearchIssuesHandler handles requests to search for JIRA issues.
func (h *JiraHandlers) SearchIssuesHandler(w http.ResponseWriter, r *http.Request) {
	h.Logger.Info("Request received", "method", r.Method, "path", r.URL.Path)
	// SearchIssuesHandler handles POST requests to /search_jira_issues.
	// It parses the request body containing JQL, maxResults, and fields,
	// calls the JiraService's SearchIssues method, and returns the search results
	// or an error response.

	if r.Method != http.MethodPost {
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.Logger.Error("Failed to decode request body", "error", err)
		respondWithError(w, http.StatusBadRequest, "Invalid request body") // Keep user message generic
		return
	}
	defer func() { _ = r.Body.Close() }() // Ensure body is closed

	// Basic validation
	if req.JQL == "" {
		respondWithError(w, http.StatusBadRequest, "Missing required field: jql")
		return
	}

	// Get context from request
	ctx := r.Context()
	// Default maxResults if not provided or zero
	maxResults := req.MaxResults
	if maxResults <= 0 {
		maxResults = 50 // Default to 50 if not specified or invalid
	}

	resp, err := h.JiraSvc.SearchIssues(ctx, req.JQL, maxResults, req.Fields)
	if err != nil {
		statusCode, userMessage := mapJiraError(err)
		// Log the detailed error internally
		h.Logger.Error("Error searching JIRA issues", "jql", req.JQL, "error", err)
		respondWithError(w, statusCode, userMessage) // Use user-friendly message
		return
	}

	respondWithJSON(w, http.StatusOK, resp)
}

// GetIssueDetailsHandler handles requests to get details for a specific JIRA issue.
func (h *JiraHandlers) GetIssueDetailsHandler(w http.ResponseWriter, r *http.Request) {
	h.Logger.Info("Request received", "method", r.Method, "path", r.URL.Path)
	// GetIssueDetailsHandler handles GET requests to /jira_issue/{issueKey}.
	// It extracts the issueKey from the URL path, optionally parses requested fields
	// from query parameters, calls the JiraService's GetIssue method, and returns
	// the issue details or an error response.

	if r.Method != http.MethodGet {
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Extract issueKey from path parameter using mux
	vars := mux.Vars(r)
	issueKey := vars["issueKey"]
	if issueKey == "" {
		respondWithError(w, http.StatusBadRequest, "Missing issue key in URL path")
		return
	}

	// Optional: Parse fields from query parameter
	fieldsQuery := r.URL.Query().Get("fields")
	var fields []string
	if fieldsQuery != "" {
		// Basic split, consider more robust parsing if needed
		fields = strings.Split(fieldsQuery, ",")
	}

	// Get context from request
	ctx := r.Context()
	issue, err := h.JiraSvc.GetIssue(ctx, issueKey, fields)
	if err != nil {
		statusCode, userMessage := mapJiraError(err)
		// Log the detailed error internally
		h.Logger.Error("Error getting JIRA issue details", "issueKey", issueKey, "error", err)
		respondWithError(w, statusCode, userMessage) // Use user-friendly message
		return
	}

	respondWithJSON(w, http.StatusOK, issue)
}

// GetIssuesInEpicHandler handles requests to find issues within a specific epic.
func (h *JiraHandlers) GetIssuesInEpicHandler(w http.ResponseWriter, r *http.Request) {
	h.Logger.Info("Request received", "method", r.Method, "path", r.URL.Path)
	// GetIssuesInEpicHandler handles GET requests to /jira_epic/{epicKey}/issues.
	// It extracts the epicKey from the URL path, constructs a JQL query to find
	// issues linked to the epic, calls the JiraService's SearchIssues method,
	// and returns the found issues or an error response.

	if r.Method != http.MethodGet {
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Extract epicKey from path parameter using mux
	vars := mux.Vars(r)
	epicKey := vars["epicKey"]
	if epicKey == "" {
		respondWithError(w, http.StatusBadRequest, "Missing epic key in URL path")
		return
	}

	// Construct JQL using the EpicLinkFieldName constant from the jira package.
	// Note the single quotes around the field name, which is often required for custom fields in JQL.
	jql := fmt.Sprintf("'%s' = '%s'", jira.EpicLinkFieldName, epicKey) // Use single quotes for JQL string literal

	// Get context from request
	ctx := r.Context()
	// Using default search options for simplicity, could allow overrides via query params
	defaultMaxResults := 50
	var defaultFields []string // Or specify default fields: []string{"summary", "status", "assignee"}

	resp, err := h.JiraSvc.SearchIssues(ctx, jql, defaultMaxResults, defaultFields)
	if err != nil {
		statusCode, userMessage := mapJiraError(err)
		// Log the detailed error internally
		h.Logger.Error("Error getting issues in epic", "epicKey", epicKey, "jql", jql, "error", err)
		respondWithError(w, statusCode, userMessage) // Use user-friendly message
		return
	}

	respondWithJSON(w, http.StatusOK, resp)
}
