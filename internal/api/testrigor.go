package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/benvon/testrigor-ci-tool/internal/config"
)

// TestRigorClient handles interactions with the TestRigor API
type TestRigorClient struct {
	cfg        *config.Config
	debugMode  bool
	httpClient HTTPClient
}

// NewTestRigorClient creates a new TestRigor API client
func NewTestRigorClient(cfg *config.Config, client ...HTTPClient) *TestRigorClient {
	var httpClient HTTPClient
	if len(client) > 0 && client[0] != nil {
		httpClient = client[0]
	} else {
		httpClient = NewDefaultHTTPClient()
	}
	return &TestRigorClient{cfg: cfg, httpClient: httpClient}
}

// SetDebugMode enables or disables debug output
func (c *TestRigorClient) SetDebugMode(debug bool) {
	c.debugMode = debug
}

// TestRunOptions contains options for starting a test run
type TestRunOptions struct {
	ForceCancelPreviousTesting bool
	TestCaseUUIDs              []string
	BranchName                 string
	CommitHash                 string
	URL                        string
	Labels                     []string
	ExcludedLabels             []string
	CustomName                 string
	MakeXrayReports            bool
}

// TestStatus represents the status of a test run
type TestStatus struct {
	Status     string
	DetailsURL string
	TaskID     string
	Errors     []TestError
	Results    TestResults
}

// TestError represents an error that occurred during a test run
type TestError struct {
	Category    string
	Error       string
	Occurrences int
	Severity    string
	DetailsURL  string
}

// TestResults represents the overall results of a test run
type TestResults struct {
	Total      int
	InQueue    int
	InProgress int
	Failed     int
	Passed     int
	Canceled   int
	NotStarted int
	Crash      int
}

// TestRunResult contains the result of starting a test run
type TestRunResult struct {
	TaskID     string
	BranchName string
}

// HTTPClient interface for making HTTP requests
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// DefaultHTTPClient is the default HTTP client implementation
type DefaultHTTPClient struct {
	client *http.Client
}

// NewDefaultHTTPClient creates a new default HTTP client
func NewDefaultHTTPClient() *DefaultHTTPClient {
	return &DefaultHTTPClient{
		client: &http.Client{},
	}
}

// Do implements the HTTPClient interface
func (c *DefaultHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return c.client.Do(req)
}

// requestOptions contains options for making HTTP requests
type requestOptions struct {
	method      string
	url         string
	body        interface{}
	headers     map[string]string
	debugMode   bool
	contentType string
}

// makeRequest handles making HTTP requests with common error handling and debug output
func (c *TestRigorClient) makeRequest(opts requestOptions) ([]byte, error) {
	var bodyReader io.Reader
	if opts.body != nil {
		jsonBody, err := json.MarshalIndent(opts.body, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("error marshaling request body: %v", err)
		}
		bodyReader = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(opts.method, opts.url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	// Set default headers
	req.Header.Set("Content-Type", opts.contentType)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("auth-token", c.cfg.TestRigor.AuthToken)

	// Set additional headers
	for key, value := range opts.headers {
		req.Header.Set(key, value)
	}

	if c.debugMode {
		fmt.Printf("\nSending %s request to: %s\n", opts.method, req.URL)
		fmt.Printf("Request headers:\n%s", formatHeaders(req.Header))
		if opts.body != nil {
			jsonBody, _ := json.MarshalIndent(opts.body, "", "  ")
			fmt.Printf("Request body:\n%s\n", string(jsonBody))
		}
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	if c.debugMode {
		fmt.Printf("Response status: %s\n", resp.Status)
		fmt.Printf("Response headers:\n%s", formatHeaders(resp.Header))
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %v", err)
	}

	if c.debugMode {
		var prettyJSON bytes.Buffer
		if err := json.Indent(&prettyJSON, bodyBytes, "", "  "); err != nil {
			fmt.Printf("Response body (raw):\n%s\n", string(bodyBytes))
		} else {
			fmt.Printf("Response body:\n%s\n", prettyJSON.String())
		}
	}

	return bodyBytes, nil
}

// parseErrorResponse parses an error response from the API
func parseErrorResponse(bodyBytes []byte) error {
	var errResp struct {
		Status  int      `json:"status"`
		Message string   `json:"message"`
		Errors  []string `json:"errors"`
		TaskID  string   `json:"taskId,omitempty"`
	}
	if err := json.Unmarshal(bodyBytes, &errResp); err != nil {
		return fmt.Errorf("error parsing error response: %v", err)
	}

	if len(errResp.Errors) > 0 {
		return fmt.Errorf("API error: %s - %s", errResp.Message, strings.Join(errResp.Errors, ", "))
	}
	return fmt.Errorf("API error: %s", errResp.Message)
}

// formatHeaders formats HTTP headers for debug output
func formatHeaders(headers http.Header) string {
	var result strings.Builder
	for key, values := range headers {
		result.WriteString(fmt.Sprintf("%s: %s\n", key, strings.Join(values, ", ")))
	}
	return result.String()
}

// generateFakeCommitHash generates a fake Git commit hash that looks real but is obviously fake
func generateFakeCommitHash(timestamp string) string {
	// Remove hyphens from timestamp
	cleanTimestamp := strings.ReplaceAll(timestamp, "-", "")
	// Start with "fake" in hex (66616b65) followed by timestamp and padded with zeros
	// This ensures it's 40 characters and starts with "fake" in hex
	base := fmt.Sprintf("66616b65%s", cleanTimestamp)
	// Pad with zeros to reach 40 characters
	return fmt.Sprintf("%s%s", base, strings.Repeat("0", 40-len(base)))
}

// StartTestRun starts a new test run with the given options
func (c *TestRigorClient) StartTestRun(opts TestRunOptions) (*TestRunResult, error) {
	// Prepare request body
	body := map[string]interface{}{
		"forceCancelPreviousTesting": opts.ForceCancelPreviousTesting,
		"skipXrayCloud":              !opts.MakeXrayReports,
	}

	var branchName string

	// Add test case UUIDs if provided
	if len(opts.TestCaseUUIDs) > 0 {
		body["testCaseUuids"] = opts.TestCaseUUIDs
		if opts.URL != "" {
			body["url"] = opts.URL
		}
	} else {
		// Handle branch and commit information
		if opts.CommitHash != "" && opts.BranchName == "" {
			// Error: Commit hash provided without branch name
			return nil, fmt.Errorf("commit hash must be accompanied by a branch name")
		}

		if opts.BranchName != "" {
			if opts.CommitHash != "" {
				// Scenario 1: Real branch and commit provided
				branchName = opts.BranchName
				body["branch"] = map[string]string{
					"name":   branchName,
					"commit": opts.CommitHash,
				}
			} else {
				// Scenario 2: Branch name only - use fake branch/commit
				timestamp := time.Now().Format("20060102-150405")
				branchName = fmt.Sprintf("fake-branch-%s", timestamp)
				body["branch"] = map[string]string{
					"name":   branchName,
					"commit": generateFakeCommitHash(timestamp),
				}
			}
		} else if len(opts.Labels) > 0 {
			// Scenario 4: No branch/commit, but labels provided - use fake branch/commit
			timestamp := time.Now().Format("20060102-150405")
			branchName = fmt.Sprintf("fake-branch-%s", timestamp)
			body["branch"] = map[string]string{
				"name":   branchName,
				"commit": generateFakeCommitHash(timestamp),
			}
		}

		// Add labels if provided
		if len(opts.Labels) > 0 {
			body["labels"] = opts.Labels
			// Always include excludedLabels when using labels
			if len(opts.ExcludedLabels) > 0 {
				body["excludedLabels"] = opts.ExcludedLabels
			} else {
				body["excludedLabels"] = []string{}
			}
		}

		// URL is required when using branch
		if body["branch"] != nil && opts.URL != "" {
			body["url"] = opts.URL
		}
	}

	if opts.CustomName != "" {
		body["customName"] = opts.CustomName
	}

	// Make the request
	bodyBytes, err := c.makeRequest(requestOptions{
		method:      "POST",
		url:         fmt.Sprintf("%s/apps/%s/retest", c.cfg.TestRigor.APIURL, c.cfg.TestRigor.AppID),
		body:        body,
		contentType: "application/json",
		debugMode:   c.debugMode,
	})
	if err != nil {
		return nil, err
	}

	// Parse response
	var result map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	// Check for error response
	if taskID, ok := result["taskId"].(string); ok {
		return &TestRunResult{
			TaskID:     taskID,
			BranchName: branchName,
		}, nil
	}

	// If we get here, it's an error response
	return nil, parseErrorResponse(bodyBytes)
}

// GetTestStatus gets the current status of a test run
func (c *TestRigorClient) GetTestStatus(branchName string, labels []string) (*TestStatus, error) {
	// Build URL with query parameters
	url := fmt.Sprintf("%s/apps/%s/status", c.cfg.TestRigor.APIURL, c.cfg.TestRigor.AppID)

	// Add query parameters if provided
	params := make([]string, 0)
	if branchName != "" {
		params = append(params, fmt.Sprintf("branchName=%s", branchName))
	}
	if len(labels) > 0 {
		params = append(params, fmt.Sprintf("labels=%s", strings.Join(labels, ",")))
	}
	if len(params) > 0 {
		url += "?" + strings.Join(params, "&")
	}

	// Make the request
	bodyBytes, err := c.makeRequest(requestOptions{
		method:      "GET",
		url:         url,
		contentType: "application/json",
		debugMode:   c.debugMode,
	})
	if err != nil {
		return nil, err
	}

	// Parse response
	var result map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	// Get status from response
	status, _ := result["status"].(string)
	detailsURL, _ := result["detailsUrl"].(string)

	// Extract task ID from details URL if present
	var taskID string
	if detailsURL != "" {
		parts := strings.Split(detailsURL, "/")
		if len(parts) > 0 {
			taskID = parts[len(parts)-1]
		}
	}

	// Parse errors if present
	var errors []TestError
	if errList, ok := result["errors"].([]interface{}); ok {
		for _, err := range errList {
			if errMap, ok := err.(map[string]interface{}); ok {
				testErr := TestError{
					Category:    getString(errMap, "category"),
					Error:       getString(errMap, "error"),
					Occurrences: getInt(errMap, "occurrences"),
					Severity:    getString(errMap, "severity"),
					DetailsURL:  getString(errMap, "detailsUrl"),
				}
				errors = append(errors, testErr)
			}
		}
	}

	// Parse overall results if present
	var results TestResults
	if resultsMap, ok := result["overallResults"].(map[string]interface{}); ok {
		results = TestResults{
			Total:      getInt(resultsMap, "Total"),
			InQueue:    getInt(resultsMap, "In queue"),
			InProgress: getInt(resultsMap, "In progress"),
			Failed:     getInt(resultsMap, "Failed"),
			Passed:     getInt(resultsMap, "Passed"),
			Canceled:   getInt(resultsMap, "Canceled"),
			NotStarted: getInt(resultsMap, "Not started"),
			Crash:      getInt(resultsMap, "Crash"),
		}
	}

	return &TestStatus{
		Status:     status,
		DetailsURL: detailsURL,
		TaskID:     taskID,
		Errors:     errors,
		Results:    results,
	}, nil
}

// WaitForTestCompletion waits for a test run to complete
func (c *TestRigorClient) WaitForTestCompletion(branchName string, labels []string, pollInterval int, debugMode bool) error {
	lastStatus := ""
	lastResults := TestResults{}
	lastUpdate := time.Now()

	for {
		status, err := c.GetTestStatus(branchName, labels)
		if err != nil {
			return fmt.Errorf("error checking status: %v", err)
		}

		// Print status changes and periodic updates
		shouldPrint := false
		reason := ""

		// Print on status change
		if status.Status != lastStatus {
			shouldPrint = true
			reason = "status changed"
			lastStatus = status.Status
		}

		// Print on results change
		if status.Results != lastResults {
			shouldPrint = true
			reason = "results updated"
			lastResults = status.Results
		}

		// Print periodic updates (every 30 seconds)
		if time.Since(lastUpdate) >= 30*time.Second {
			shouldPrint = true
			reason = "periodic update"
			lastUpdate = time.Now()
		}

		if shouldPrint {
			fmt.Printf("\n[%s] Current status: %s\n", reason, status.Status)
			fmt.Printf("  Passed: %d, Failed: %d, In Progress: %d, In Queue: %d\n",
				status.Results.Passed,
				status.Results.Failed,
				status.Results.InProgress,
				status.Results.InQueue,
			)
		}

		// Check if the test run is complete
		if isComplete(status.Status) {
			fmt.Printf("\nTest run completed with status: %s\n", status.Status)
			if status.DetailsURL != "" {
				fmt.Printf("Details URL: %s\n", status.DetailsURL)
			}

			// Print final results
			fmt.Printf("\nFinal Results:\n")
			fmt.Printf("  Total: %d\n", status.Results.Total)
			fmt.Printf("  Passed: %d\n", status.Results.Passed)
			fmt.Printf("  Failed: %d\n", status.Results.Failed)
			fmt.Printf("  In Progress: %d\n", status.Results.InProgress)
			fmt.Printf("  In Queue: %d\n", status.Results.InQueue)
			fmt.Printf("  Not Started: %d\n", status.Results.NotStarted)
			fmt.Printf("  Canceled: %d\n", status.Results.Canceled)
			fmt.Printf("  Crash: %d\n", status.Results.Crash)

			// Print errors if any
			if len(status.Errors) > 0 {
				fmt.Printf("\nErrors:\n")
				for _, err := range status.Errors {
					fmt.Printf("  Category: %s\n", err.Category)
					fmt.Printf("  Error: %s\n", err.Error)
					fmt.Printf("  Severity: %s\n", err.Severity)
					fmt.Printf("  Occurrences: %d\n", err.Occurrences)
					if err.DetailsURL != "" {
						fmt.Printf("  Details URL: %s\n", err.DetailsURL)
					}
					fmt.Println()
				}
			}

			// Return error if test run failed
			if status.Status == "Failed" {
				return fmt.Errorf("test run failed")
			}
			return nil
		}

		time.Sleep(time.Duration(pollInterval) * time.Second)
	}
}

// isComplete checks if a test status indicates completion
func isComplete(status string) bool {
	completeStates := map[string]bool{
		"completed": true,
		"failed":    true,
		"error":     true,
		"cancelled": true,
		"canceled":  true,
	}
	return completeStates[strings.ToLower(status)]
}

// Helper functions for parsing JSON values
func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

func getInt(m map[string]interface{}, key string) int {
	if val, ok := m[key].(float64); ok {
		return int(val)
	}
	if val, ok := m[key].(int); ok {
		return val
	}
	return 0
}

// GetJUnitReport gets the JUnit report for a test run
func (c *TestRigorClient) GetJUnitReport(taskID string, debugMode bool) error {
	// Build URL using api2.testrigor.com for JUnit report endpoint
	url := fmt.Sprintf("https://api2.testrigor.com/api/v1/apps/%s/runs/%s/junit_report", c.cfg.TestRigor.AppID, taskID)

	// Make the request
	bodyBytes, err := c.makeRequest(requestOptions{
		method:      "GET",
		url:         url,
		contentType: "application/xml",
		debugMode:   debugMode,
	})
	if err != nil {
		return err
	}

	// Save to file
	reportPath := "test-report.xml"
	if err := os.WriteFile(reportPath, bodyBytes, 0644); err != nil {
		return fmt.Errorf("error saving report: %v", err)
	}

	// Get absolute path for display
	absPath, err := filepath.Abs(reportPath)
	if err != nil {
		absPath = reportPath // Fallback to relative path if we can't get absolute path
	}

	fmt.Printf("\nJUnit report downloaded successfully:\n")
	fmt.Printf("  Path: %s\n", reportPath)
	fmt.Printf("  Full path: %s\n", absPath)
	fmt.Printf("  Size: %d bytes\n", len(bodyBytes))

	return nil
}

// WaitForJUnitReport waits for the JUnit report to be ready and downloads it
func (c *TestRigorClient) WaitForJUnitReport(taskID string, pollInterval int, debugMode bool) error {
	for {
		err := c.GetJUnitReport(taskID, debugMode)
		if err == nil {
			// Report successfully downloaded
			return nil
		}

		// Check if we should keep waiting
		if strings.Contains(err.Error(), "still being generated") {
			if debugMode {
				fmt.Printf("Waiting for JUnit report to be generated...\n")
			}
			time.Sleep(time.Duration(pollInterval) * time.Second)
			continue
		}

		// Any other error should be returned
		return err
	}
}
