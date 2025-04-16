package main

import (
	"log/slog" // Added for structured logging
	"net/http"
	"os"

	"jira-mcp-server/internal/handlers"
	"jira-mcp-server/internal/jira"

	"github.com/gorilla/mux" // Added mux import
	"github.com/spf13/viper" // Added viper import
)

func main() {
	// Initialize structured logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// --- Configuration Setup using Viper ---
	viper.SetDefault("PORT", "8080")
	viper.SetDefault("JIRA_URL", "")        // No sensible default
	viper.SetDefault("JIRA_USER_EMAIL", "") // No sensible default
	viper.SetDefault("JIRA_API_TOKEN", "")  // No sensible default

	viper.SetConfigName("config") // Name of config file (without extension)
	viper.SetConfigType("yaml")   // REQUIRED if the config file does not have the extension in the name
	viper.AddConfigPath(".")      // Look for config in the working directory
	// viper.AddConfigPath("$HOME/.appname") // Optionally look in home directory
	// viper.AddConfigPath("/etc/appname/") // Optionally look in /etc

	// Attempt to read the config file but ignore errors if it's not found
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			slog.Info("Config file not found, using defaults and environment variables.")
		} else {
			// Config file was found but another error was produced
			slog.Error("Error reading config file", "error", err)
			os.Exit(1)
		}
	}

	viper.SetEnvPrefix("JIRA_MCP") // Env vars will be JIRA_MCP_PORT, JIRA_MCP_JIRA_URL, etc.
	viper.AutomaticEnv()           // Read in environment variables that match

	// Verify required configuration values are present (after loading defaults, file, env)
	requiredKeys := []string{"JIRA_URL", "JIRA_USER_EMAIL", "JIRA_API_TOKEN"}
	for _, key := range requiredKeys {
		// Viper keys are case-insensitive, but we use uppercase for consistency
		if viper.GetString(key) == "" {
			// Construct the expected env var name for the error message
			envVarName := viper.GetEnvPrefix() + "_" + key
			slog.Error("Required configuration value not set. Set it via config file or environment variable.", "key", key, "env_var", envVarName)
			os.Exit(1)
		}
	}
	// --- End Configuration Setup ---

	// Initialize JIRA client
	jiraClient, err := jira.NewClient(nil) // Pass nil to use http.DefaultClient
	if err != nil {
		slog.Error("Failed to create JIRA client", "error", err)
		os.Exit(1)
	}

	// Initialize handlers with dependencies
	jiraHandlers := handlers.NewJiraHandlers(jiraClient, logger) // Pass logger

	// Set up router
	r := mux.NewRouter()

	// Register handlers
	r.HandleFunc("/create_jira_issue", jiraHandlers.CreateJiraIssueHandler).Methods("POST")
	r.HandleFunc("/search_jira_issues", jiraHandlers.SearchIssuesHandler).Methods("POST")
	r.HandleFunc("/jira_issue/{issueKey}", jiraHandlers.GetIssueDetailsHandler).Methods("GET")
	r.HandleFunc("/jira_epic/{epicKey}/issues", jiraHandlers.GetIssuesInEpicHandler).Methods("GET")

	port := viper.GetString("PORT") // Get port from Viper (checks env: JIRA_MCP_PORT, config: port, default: 8080)

	serverAddr := ":" + port
	slog.Info("Starting JIRA MCP server", "address", serverAddr)
	err = http.ListenAndServe(serverAddr, r) // Use mux router
	if err != nil {
		slog.Error("Failed to start server", "error", err)
		os.Exit(1)
	}
}
