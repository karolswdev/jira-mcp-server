package jira

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	// Added for URL parsing in error handling
)

// EpicLinkFieldName holds the JIRA custom field ID typically used for "Epic Link".
// NOTE: This ID can vary between JIRA instances. Common values include 'customfield_10014', 'customfield_10008'.
// Verify the correct ID for your specific JIRA Cloud instance if filtering by Epic Link fails.
const EpicLinkFieldName = "customfield_10014"

// JiraService defines the interface for interacting with the JIRA API.
// This allows for dependency injection and easier testing by mocking the JIRA client.

// JiraService defines the interface for interacting with the JIRA API.
// This allows for dependency injection and easier testing.
type JiraService interface {
	CreateIssue(ctx context.Context, req CreateIssueRequest) (*CreateIssueResponse, error)
	SearchIssues(ctx context.Context, jql string, maxResults int, fields []string) (*SearchResponse, error)
	GetIssue(ctx context.Context, issueKey string, fields []string) (*Issue, error)
}

// Client implements the JiraService interface and provides methods
// for interacting with the JIRA Cloud REST API.

type Client struct {
	baseURL    string
	userEmail  string
	apiToken   string
	httpClient *http.Client
}

// NewClient creates a new JIRA API client.
// It reads configuration (JIRA_URL, JIRA_USER_EMAIL, JIRA_API_TOKEN) from Viper.
// An optional custom http.Client can be provided for testing or specific transport configurations;
// if httpClient is nil, http.DefaultClient will be used.
// It returns an error if required configuration is missing.

// NewClient creates a new JIRA API client.
// It reads configuration from environment variables (JIRA_URL, JIRA_USER_EMAIL, JIRA_API_TOKEN).
// An optional custom http.Client can be provided for testing or specific transport configurations.
// If httpClient is nil, http.DefaultClient will be used.
func NewClient(httpClient *http.Client) (*Client, error) {
	baseURL := os.Getenv("JIRA_URL")
	userEmail := os.Getenv("JIRA_USER_EMAIL")
	apiToken := os.Getenv("JIRA_API_TOKEN")

	if baseURL == "" || userEmail == "" || apiToken == "" {
		return nil, fmt.Errorf("missing required JIRA credentials in environment variables (JIRA_URL, JIRA_USER_EMAIL, JIRA_API_TOKEN)")
	}

	client := httpClient
	if client == nil {
		client = http.DefaultClient // Use default client if none provided
	}

	return &Client{
		baseURL:    baseURL,
		userEmail:  userEmail,
		apiToken:   apiToken,
		httpClient: client,
	}, nil
}

// CreateIssueRequest defines the structure for the request body when creating a JIRA issue.
// It includes required fields like ProjectKey, Summary, IssueType, and optional fields.

type CreateIssueRequest struct {
	ProjectKey    string `json:"project_key"`
	Summary       string `json:"summary"`
	IssueType     string `json:"issue_type"`
	Description   string `json:"description,omitempty"`
	AssigneeEmail string `json:"assignee_email,omitempty"`
	ParentKey     string `json:"parent_key,omitempty"`
}

// CreateIssueResponse defines the structure for the successful response body
// when creating a JIRA issue, containing the new issue's Key and Self URL.

type CreateIssueResponse struct {
	Key  string `json:"key"`
	Self string `json:"self"`
}

// SearchResponse represents the structure of the response from JIRA's /rest/api/3/search endpoint,
// containing pagination details and a slice of found Issues.

// SearchResponse represents the response from JIRA's /rest/api/3/search endpoint
type SearchResponse struct {
	Expand     string  `json:"expand"`
	StartAt    int     `json:"startAt"`
	MaxResults int     `json:"maxResults"`
	Total      int     `json:"total"`
	Issues     []Issue `json:"issues"`
}

// Issue represents a simplified structure for a JIRA issue, commonly returned in search results
// or when getting issue details. It includes basic identifiers and a map for arbitrary fields.

// Issue represents a JIRA issue with common fields
type Issue struct {
	Expand string                 `json:"expand"`
	ID     string                 `json:"id"`
	Key    string                 `json:"key"`
	Self   string                 `json:"self"`
	Fields map[string]interface{} `json:"fields"`
}

// JiraAPIError represents an error returned specifically from the JIRA API.
// It includes the HTTP status code, the raw error message or body from JIRA,
// and the URL that was called.

// JiraAPIError represents an error returned by the JIRA API, including the status code.
type JiraAPIError struct {
	StatusCode int
	Message    string // Raw error message or body from JIRA
	URL        string // The URL that caused the error
}

func (e *JiraAPIError) Error() string {
	return fmt.Sprintf("JIRA API error: status %d, message: %s (URL: %s)", e.StatusCode, e.Message, e.URL)
}

// CreateIssue sends a request to the JIRA API to create a new issue.
// It validates required fields in the CreateIssueRequest, constructs the API payload
// (including handling the description format), and sends an authenticated POST request.
// It returns a CreateIssueResponse on success or an error (potentially a JiraAPIError).

func (c *Client) CreateIssue(ctx context.Context, req CreateIssueRequest) (*CreateIssueResponse, error) {
	// Validate required fields
	if req.ProjectKey == "" || req.Summary == "" || req.IssueType == "" {
		return nil, fmt.Errorf("project_key, summary, and issue_type are required")
	}

	// Construct the JIRA API payload using the fields from the request struct
	fields := map[string]interface{}{
		"project":   map[string]string{"key": req.ProjectKey},
		"summary":   req.Summary,
		"issuetype": map[string]string{"name": req.IssueType},
	}

	// Add optional fields if provided
	if req.Description != "" {
		// JIRA description often expects a specific document format (Atlassian Document Format)
		// For simplicity here, we'll send it as a plain string, but a real implementation
		// might need to structure it correctly.
		// Example for plain text (might not render correctly in newer JIRA versions):
		// fields["description"] = req.Description
		// Example for ADF:
		fields["description"] = map[string]interface{}{
			"type":    "doc",
			"version": 1,
			"content": []map[string]interface{}{
				{
					"type": "paragraph",
					"content": []map[string]interface{}{
						{
							"type": "text",
							"text": req.Description,
						},
					},
				},
			},
		}
	}
	// Assignee logic was removed as email assignment is less reliable and account ID is preferred.
	// If needed, re-add logic here using account ID.
	if req.ParentKey != "" {
		fields["parent"] = map[string]string{"key": req.ParentKey}
	}

	payload := map[string]interface{}{
		"fields": fields,
	}

	// Marshal payload to JSON
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request payload: %v", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/rest/api/3/issue", c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %v", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.SetBasicAuth(c.userEmail, c.apiToken)

	// Send request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to JIRA API: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 { // Check for non-2xx status
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, &JiraAPIError{
			StatusCode: resp.StatusCode,
			Message:    string(bodyBytes),
			URL:        url, // Use the request URL
		}
	}

	// Parse successful response
	var issueResponse CreateIssueResponse
	if err := json.NewDecoder(resp.Body).Decode(&issueResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}
	return &issueResponse, nil
}

// SearchIssues sends a request to the JIRA API's search endpoint (/rest/api/3/search).
// It takes a JQL query string, maximum results count, and optional fields list.
// It returns a SearchResponse containing the matching issues or an error (potentially a JiraAPIError).

// SearchIssues searches for JIRA issues using JQL query
func (c *Client) SearchIssues(ctx context.Context, jql string, maxResults int, fields []string) (*SearchResponse, error) {
	if jql == "" {
		return nil, fmt.Errorf("JQL query cannot be empty")
	}

	// Construct request payload
	payload := map[string]interface{}{
		"jql":        jql,
		"maxResults": maxResults,
	}

	if len(fields) > 0 {
		payload["fields"] = fields
	}

	// Marshal payload to JSON
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal search request: %v", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/rest/api/3/search", c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, fmt.Errorf("failed to create search request: %v", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.SetBasicAuth(c.userEmail, c.apiToken)

	// Send request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send search request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 { // Check for non-2xx status
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, &JiraAPIError{
			StatusCode: resp.StatusCode,
			Message:    string(bodyBytes),
			URL:        url, // Use the request URL
		}
	}

	// Parse successful response
	var searchResponse SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResponse); err != nil {
		return nil, fmt.Errorf("failed to decode search response: %v", err)
	}
	return &searchResponse, nil
}

// GetIssue sends a request to the JIRA API to retrieve details for a single issue by its key.
// It takes the issueKey and an optional list of fields to retrieve.
// It returns an Issue struct containing the details or an error (potentially a JiraAPIError).

// GetIssue retrieves a single JIRA issue by key
func (c *Client) GetIssue(ctx context.Context, issueKey string, fields []string) (*Issue, error) {
	if issueKey == "" {
		return nil, fmt.Errorf("issue key cannot be empty")
	}

	// Construct URL
	url := fmt.Sprintf("%s/rest/api/3/issue/%s", c.baseURL, issueKey)

	// Add fields query parameter if specified
	if len(fields) > 0 {
		url = fmt.Sprintf("%s?fields=%s", url, fieldsCommaSeparated(fields))
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Set headers
	httpReq.Header.Set("Accept", "application/json")
	httpReq.SetBasicAuth(c.userEmail, c.apiToken)

	// Send request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 { // Check for non-2xx status
		bodyBytes, _ := io.ReadAll(resp.Body)
		// Attempt to get the original URL from the request if available
		requestURL := url // Default to the constructed URL
		if httpReq != nil && httpReq.URL != nil {
			requestURL = httpReq.URL.String()
		}
		return nil, &JiraAPIError{
			StatusCode: resp.StatusCode,
			Message:    string(bodyBytes),
			URL:        requestURL,
		}
	}

	// Parse successful response
	var issue Issue
	if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return &issue, nil
}

// fieldsCommaSeparated joins field names with commas for the query parameter
func fieldsCommaSeparated(fields []string) string {
	var sb strings.Builder
	for i, field := range fields {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(field)
	}
	return sb.String()
}
