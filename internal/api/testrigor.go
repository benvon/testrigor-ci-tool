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

	"github.com/benvon/testrigor-ci-tool/internal/api/client"
	"github.com/benvon/testrigor-ci-tool/internal/api/types"
	"github.com/benvon/testrigor-ci-tool/internal/api/utils"
	"github.com/benvon/testrigor-ci-tool/internal/config"
)

// TestRigorClient handles interactions with the TestRigor API
type TestRigorClient struct {
	cfg        *config.Config
	debugMode  bool
	httpClient client.HTTPClient
	startTime  time.Time
}

// NewTestRigorClient creates a new TestRigor API client
func NewTestRigorClient(cfg *config.Config, httpClient ...client.HTTPClient) *TestRigorClient {
	var c client.HTTPClient
	if len(httpClient) > 0 && httpClient[0] != nil {
		c = httpClient[0]
	} else {
		c = client.NewDefaultHTTPClient()
	}
	return &TestRigorClient{cfg: cfg, httpClient: c}
}

// SetDebugMode enables or disables debug output
func (c *TestRigorClient) SetDebugMode(debug bool) {
	c.debugMode = debug
}

// HTTPClient interface for making HTTP requests
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
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

// printDebug prints debug information about the request/response
func (c *TestRigorClient) printDebug(req *http.Request, resp *http.Response, body interface{}, bodyBytes []byte) {
	if req != nil {
		fmt.Printf("\nSending %s request to: %s\n", req.Method, req.URL)
		fmt.Printf("Request headers:\n%s", formatHeaders(req.Header))
		if body != nil {
			jsonBody, _ := json.MarshalIndent(body, "", "  ")
			fmt.Printf("Request body:\n%s\n", string(jsonBody))
		}
	}

	if resp != nil {
		fmt.Printf("Response status: %s\n", resp.Status)
		fmt.Printf("Response headers:\n%s", formatHeaders(resp.Header))
	}

	if len(bodyBytes) > 0 {
		var prettyJSON bytes.Buffer
		if err := json.Indent(&prettyJSON, bodyBytes, "", "  "); err != nil {
			fmt.Printf("Response body (raw):\n%s\n", string(bodyBytes))
		} else {
			fmt.Printf("Response body:\n%s\n", prettyJSON.String())
		}
	}
}

// handleResponse processes the HTTP response and returns appropriate result or error
func (c *TestRigorClient) handleResponse(resp *http.Response, bodyBytes []byte) ([]byte, error) {
	var jsonCheck map[string]interface{}
	jsonErr := json.Unmarshal(bodyBytes, &jsonCheck)

	switch resp.StatusCode {
	case 200:
		return bodyBytes, nil
	case 227, 228:
		// These are normal status codes indicating test is in progress
		return bodyBytes, fmt.Errorf("status %d", resp.StatusCode)
	case 230:
		return bodyBytes, fmt.Errorf("test failed")
	case 404:
		// Check if this is a test crash error
		if jsonErr == nil {
			if errs, ok := jsonCheck["errors"].([]interface{}); ok && len(errs) > 0 {
				for _, err := range errs {
					if errStr, ok := err.(string); ok && strings.Contains(errStr, "CRASH:") {
						return bodyBytes, fmt.Errorf("test crashed: %s", errStr)
					}
				}
			}
		}
		// If not a test crash, handle as normal API error
		if jsonErr != nil {
			return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(bodyBytes))
		}
		msg := getString(jsonCheck, "message")
		details := c.extractErrorDetails(jsonCheck)
		if msg == "" && details == "" {
			return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(bodyBytes))
		}
		return nil, fmt.Errorf("API error (status %d): %s, errors: %s", resp.StatusCode, msg, details)
	case 400, 401, 403, 500, 502, 503, 504:
		if jsonErr != nil {
			return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(bodyBytes))
		}
		msg := getString(jsonCheck, "message")
		details := c.extractErrorDetails(jsonCheck)
		if msg == "" && details == "" {
			return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(bodyBytes))
		}
		return nil, fmt.Errorf("API error (status %d): %s, errors: %s", resp.StatusCode, msg, details)
	default:
		if jsonErr == nil {
			if msg, ok := jsonCheck["message"].(string); ok {
				return nil, fmt.Errorf("unexpected status code: %d, message: %s", resp.StatusCode, msg)
			}
		}
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
}

// extractErrorDetails extracts error details from the JSON response
func (c *TestRigorClient) extractErrorDetails(jsonCheck map[string]interface{}) string {
	if errs, ok := jsonCheck["errors"].([]interface{}); ok && len(errs) > 0 {
		detailStrs := make([]string, 0, len(errs))
		for _, e := range errs {
			if s, ok := e.(string); ok {
				detailStrs = append(detailStrs, s)
			}
		}
		return strings.Join(detailStrs, "; ")
	}
	return ""
}

// prepareRequest prepares the HTTP request with headers and body
func (c *TestRigorClient) prepareRequest(opts requestOptions) (*http.Request, error) {
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

	req.Header.Set("Content-Type", opts.contentType)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("auth-token", c.cfg.TestRigor.AuthToken)

	for key, value := range opts.headers {
		req.Header.Set(key, value)
	}

	return req, nil
}

// processResponse processes the HTTP response and returns the body
func (c *TestRigorClient) processResponse(resp *http.Response) ([]byte, error) {
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %v", err)
	}

	if len(bodyBytes) == 0 {
		return nil, fmt.Errorf("empty response body")
	}

	return bodyBytes, nil
}

// makeRequest handles making HTTP requests with common error handling and debug output
func (c *TestRigorClient) makeRequest(opts requestOptions) ([]byte, error) {
	req, err := c.prepareRequest(opts)
	if err != nil {
		return nil, err
	}

	if c.debugMode {
		c.printDebug(req, nil, opts.body, nil)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := c.processResponse(resp)
	if err != nil {
		return nil, err
	}

	if c.debugMode {
		c.printDebug(nil, resp, nil, bodyBytes)
	}

	return c.handleResponse(resp, bodyBytes)
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

// prepareBranchInfo prepares branch information for the test run
func (c *TestRigorClient) prepareBranchInfo(opts types.TestRunOptions) (string, map[string]interface{}) {
	if opts.CommitHash != "" && opts.BranchName == "" {
		return "", nil
	}

	if opts.BranchName == "" && len(opts.Labels) == 0 {
		return "", nil
	}

	timestamp := time.Now().Format("20060102-150405")
	branchName := opts.BranchName
	if branchName == "" {
		branchName = fmt.Sprintf("fake-branch-%s", timestamp)
	}

	branchInfo := map[string]string{
		"name": branchName,
	}

	if opts.CommitHash != "" {
		branchInfo["commit"] = opts.CommitHash
	} else {
		branchInfo["commit"] = utils.GenerateFakeCommitHash(timestamp)
	}

	return branchName, map[string]interface{}{"branch": branchInfo}
}

// prepareRequestBody prepares the request body for starting a test run
func (c *TestRigorClient) prepareRequestBody(opts types.TestRunOptions) (map[string]interface{}, string) {
	body := map[string]interface{}{
		"forceCancelPreviousTesting": opts.ForceCancelPreviousTesting,
		"skipXrayCloud":              !opts.MakeXrayReports,
	}

	var branchName string

	if len(opts.TestCaseUUIDs) > 0 {
		body["testCaseUuids"] = opts.TestCaseUUIDs
		if opts.URL != "" {
			body["url"] = opts.URL
		}
		return body, branchName
	}

	var branchInfo map[string]interface{}
	branchName, branchInfo = c.prepareBranchInfo(opts)
	for k, v := range branchInfo {
		body[k] = v
	}

	if len(opts.Labels) > 0 {
		body["labels"] = opts.Labels
		body["excludedLabels"] = opts.ExcludedLabels
	}

	if branchInfo != nil && opts.URL != "" {
		body["url"] = opts.URL
	}

	if opts.CustomName != "" {
		body["customName"] = opts.CustomName
	}

	return body, branchName
}

// StartTestRun starts a new test run with the given options
func (c *TestRigorClient) StartTestRun(opts types.TestRunOptions) (*types.TestRunResult, error) {
	body, branchName := c.prepareRequestBody(opts)

	// Set the start time when the test run begins
	c.startTime = time.Now()

	bodyBytes, err := client.MakeRequest(c.httpClient, c.cfg, client.RequestOptions{
		Method:      "POST",
		URL:         fmt.Sprintf("%s/apps/%s/retest", c.cfg.TestRigor.APIURL, c.cfg.TestRigor.AppID),
		Body:        body,
		ContentType: "application/json",
		DebugMode:   c.debugMode,
	})
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	if taskID, ok := result["taskId"].(string); ok {
		return &types.TestRunResult{
			TaskID:     taskID,
			BranchName: branchName,
		}, nil
	}

	return nil, utils.ParseErrorResponse(bodyBytes)
}

// GetTestStatus gets the current status of a test run
func (c *TestRigorClient) GetTestStatus(branchName string, labels []string) (*types.TestStatus, error) {
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
	bodyBytes, err := client.MakeRequest(c.httpClient, c.cfg, client.RequestOptions{
		Method:      "GET",
		URL:         url,
		ContentType: "application/json",
		DebugMode:   c.debugMode,
	})

	// For API errors (400, 401, 403, 404, 500, etc.), return the error directly
	if err != nil && !strings.Contains(err.Error(), "status 227") && !strings.Contains(err.Error(), "status 228") {
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
	var errors []types.TestError
	if errList, ok := result["errors"].([]interface{}); ok {
		for _, err := range errList {
			if errMap, ok := err.(map[string]interface{}); ok {
				testErr := types.TestError{
					Category:    utils.GetString(errMap, "category"),
					Error:       utils.GetString(errMap, "error"),
					Occurrences: utils.GetInt(errMap, "occurrences"),
					Severity:    utils.GetString(errMap, "severity"),
					DetailsURL:  utils.GetString(errMap, "detailsUrl"),
				}
				errors = append(errors, testErr)
			} else if errStr, ok := err.(string); ok {
				// Handle string errors (like CRASH messages)
				testErr := types.TestError{
					Category:    "CRASH",
					Error:       errStr,
					Occurrences: 1,
					Severity:    "BLOCKER",
				}
				errors = append(errors, testErr)
			}
		}
	}

	// Parse overall results if present
	var results types.TestResults
	if resultsMap, ok := result["overallResults"].(map[string]interface{}); ok {
		results = types.TestResults{
			Total:      utils.GetInt(resultsMap, "Total"),
			InQueue:    utils.GetInt(resultsMap, "In queue"),
			InProgress: utils.GetInt(resultsMap, "In progress"),
			Failed:     utils.GetInt(resultsMap, "Failed"),
			Passed:     utils.GetInt(resultsMap, "Passed"),
			Canceled:   utils.GetInt(resultsMap, "Canceled"),
			NotStarted: utils.GetInt(resultsMap, "Not started"),
			Crash:      utils.GetInt(resultsMap, "Crash"),
		}
	}

	testStatus := &types.TestStatus{
		Status:     status,
		DetailsURL: detailsURL,
		TaskID:     taskID,
		Errors:     errors,
		Results:    results,
	}

	// Check for test crashes in the errors
	for _, err := range errors {
		if strings.Contains(err.Error, "CRASH:") {
			return testStatus, fmt.Errorf("test crashed: %s", err.Error)
		}
	}

	// If we got a "test in progress" error, return both the status and the error
	if err != nil && (strings.Contains(err.Error(), "status 227") || strings.Contains(err.Error(), "status 228")) {
		return testStatus, err
	}

	return testStatus, nil
}

// printTestStatus prints the current test status and results
func (c *TestRigorClient) printTestStatus(status *types.TestStatus, reason string) {
	fmt.Printf("\n[%s] Current status: %s\n", reason, status.Status)
	fmt.Printf("  Passed: %d, Failed: %d, In Progress: %d, In Queue: %d\n",
		status.Results.Passed,
		status.Results.Failed,
		status.Results.InProgress,
		status.Results.InQueue,
	)
}

// printFinalResults prints the final test results and errors
func (c *TestRigorClient) printFinalResults(status *types.TestStatus) {
	duration := time.Since(c.startTime)
	fmt.Printf("\nTest run completed with status: %s\n", status.Status)
	fmt.Printf("Total duration: %s\n", duration.Round(time.Second))
	if status.DetailsURL != "" {
		fmt.Printf("Details URL: %s\n", status.DetailsURL)
	}

	fmt.Printf("\nFinal Results:\n")
	fmt.Printf("  Total: %d\n", status.Results.Total)
	fmt.Printf("  Passed: %d\n", status.Results.Passed)
	fmt.Printf("  Failed: %d\n", status.Results.Failed)
	fmt.Printf("  In Progress: %d\n", status.Results.InProgress)
	fmt.Printf("  In Queue: %d\n", status.Results.InQueue)
	fmt.Printf("  Not Started: %d\n", status.Results.NotStarted)
	fmt.Printf("  Canceled: %d\n", status.Results.Canceled)
	fmt.Printf("  Crash: %d\n", status.Results.Crash)

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
}

// shouldPrintStatus determines if status should be printed based on changes and time
func (c *TestRigorClient) shouldPrintStatus(status *types.TestStatus, lastStatus string, lastResults types.TestResults, lastUpdate time.Time) (bool, string) {
	if status.Status != lastStatus {
		return true, "status changed"
	}
	if status.Results != lastResults {
		return true, "results updated"
	}
	if time.Since(lastUpdate) >= 30*time.Second {
		return true, "periodic update"
	}
	return false, ""
}

// handleTestStatus processes the test status and returns appropriate error
func (c *TestRigorClient) handleTestStatus(status *types.TestStatus, err error) (bool, error) {
	if err != nil {
		switch err.Error() {
		case "test failed":
			if status != nil {
				c.printFinalResults(status)
			}
			return false, fmt.Errorf("test run failed")
		case "test in progress":
			return true, nil
		default:
			return false, fmt.Errorf("error checking status: %v", err)
		}
	}

	if utils.CheckTestCompletion(status, c.debugMode) {
		c.printFinalResults(status)
		return false, nil
	}

	return true, nil
}

// checkTimeout verifies if the maximum wait time has been exceeded
func (c *TestRigorClient) checkTimeout(startTime time.Time, maxWaitTime time.Duration) error {
	if time.Since(startTime) > maxWaitTime {
		return fmt.Errorf("test run timed out after %v", maxWaitTime)
	}
	return nil
}

// handleStatusCheckError processes status check errors and returns whether to continue
func (c *TestRigorClient) handleStatusCheckError(err error, consecutiveErrors *int, maxConsecutiveErrors int, debugMode bool) (bool, error) {
	if strings.Contains(err.Error(), "status 227") || strings.Contains(err.Error(), "status 228") {
		*consecutiveErrors++
		if *consecutiveErrors >= maxConsecutiveErrors {
			return false, fmt.Errorf("received %d consecutive errors while checking test status: %v", *consecutiveErrors, err)
		}
		if debugMode {
			fmt.Printf("Error checking status (attempt %d/%d): %v\n", *consecutiveErrors, maxConsecutiveErrors, err)
		}
		return true, nil
	}
	if strings.Contains(err.Error(), "test failed") {
		return false, fmt.Errorf("test run failed")
	}
	return false, err
}

// checkTestCompletion verifies if all tests have completed execution
func (c *TestRigorClient) checkTestCompletion(status *types.TestStatus, debugMode bool) bool {
	if status.Results.Total > 0 &&
		status.Results.InQueue == 0 &&
		status.Results.InProgress == 0 &&
		status.Results.NotStarted == 0 {
		if debugMode {
			fmt.Printf("\nAll tests have finished execution. Final status: %s\n", status.Status)
		}
		c.printFinalResults(status)
		return true
	}
	return false
}

// pollTestStatus handles a single polling iteration for test status
func (c *TestRigorClient) pollTestStatus(branchName string, labels []string, debugMode bool) (*types.TestStatus, error) {
	status, err := c.GetTestStatus(branchName, labels)
	if err != nil {
		return status, err
	}
	return status, nil
}

// updateStatusDisplay updates the status display if needed
func (c *TestRigorClient) updateStatusDisplay(status *types.TestStatus, lastStatus *string, lastResults *types.TestResults, lastUpdate *time.Time) {
	shouldPrint, reason := c.shouldPrintStatus(status, *lastStatus, *lastResults, *lastUpdate)
	if shouldPrint {
		c.printTestStatus(status, reason)
		*lastStatus = status.Status
		*lastResults = status.Results
		*lastUpdate = time.Now()
	}
}

// handlePollingError processes polling errors and returns whether to continue
func (c *TestRigorClient) handlePollingError(err error, consecutiveErrors *int, maxConsecutiveErrors int, debugMode bool, pollInterval int) (bool, error) {
	if err != nil {
		// Check if it's a test in progress status code
		if strings.Contains(err.Error(), "status 227") || strings.Contains(err.Error(), "status 228") {
			time.Sleep(time.Duration(pollInterval) * time.Second)
			return true, nil
		}

		shouldContinue, err := c.handleStatusCheckError(err, consecutiveErrors, maxConsecutiveErrors, debugMode)
		if !shouldContinue {
			return false, err
		}
		time.Sleep(time.Duration(pollInterval) * time.Second)
		return true, nil
	}
	return true, nil
}

// CancelTestRun cancels a running test
func (c *TestRigorClient) CancelTestRun(branchName string, labels []string) error {
	// Build URL with query parameters
	url := fmt.Sprintf("%s/apps/%s/cancel", c.cfg.TestRigor.APIURL, c.cfg.TestRigor.AppID)

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
	_, err := client.MakeRequest(c.httpClient, c.cfg, client.RequestOptions{
		Method:      "POST",
		URL:         url,
		ContentType: "application/json",
		DebugMode:   c.debugMode,
	})

	return err
}

// WaitForTestCompletion waits for a test run to complete
func (c *TestRigorClient) WaitForTestCompletion(branchName string, labels []string, pollInterval int, debugMode bool, timeoutMinutes int) error {
	maxRetries := (timeoutMinutes * 60) / pollInterval // Convert timeout to number of retries
	retries := 0
	consecutiveErrors := 0
	maxConsecutiveErrors := 5
	statusManager := client.NewStatusUpdateManager(debugMode, time.Duration(pollInterval)*time.Second)

	// Ensure we cancel the test if we exit early
	defer func() {
		if err := c.CancelTestRun(branchName, labels); err != nil {
			if debugMode {
				fmt.Printf("Warning: Failed to cancel test run: %v\n", err)
			}
		}
	}()

	for {
		if retries >= maxRetries {
			return fmt.Errorf("timed out waiting for test completion after %d minutes", timeoutMinutes)
		}
		retries++
		status, err := c.GetTestStatus(branchName, labels)
		if err != nil {
			// Check if this is a test crash error
			if strings.Contains(err.Error(), "test crashed:") {
				if status != nil {
					statusManager.PrintFinalResults(status)
				}
				return err
			}
			shouldContinue, err := utils.HandleStatusCheckError(err, &consecutiveErrors, maxConsecutiveErrors, debugMode)
			if !shouldContinue {
				return err
			}
			time.Sleep(time.Duration(pollInterval) * time.Second)
			continue
		}
		consecutiveErrors = 0

		if status != nil {
			statusManager.Update(status)

			// Check for test crashes in the results
			if status.Results.Crash > 0 {
				statusManager.PrintFinalResults(status)
				return fmt.Errorf("test crashed: %d test(s) crashed", status.Results.Crash)
			}

			if utils.CheckTestCompletion(status, debugMode) {
				statusManager.PrintFinalResults(status)
				return nil
			}
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
	bodyBytes, err := client.MakeRequest(c.httpClient, c.cfg, client.RequestOptions{
		Method:      "GET",
		URL:         url,
		ContentType: "application/xml",
		DebugMode:   debugMode,
	})
	if err != nil {
		// Check if it's a "report not ready" error
		if strings.Contains(err.Error(), "Report still being generated") {
			return fmt.Errorf("Report still being generated")
		}
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
	maxRetries := 60 // e.g., wait up to 60 * pollInterval seconds
	retries := 0
	for {
		if retries >= maxRetries {
			return fmt.Errorf("timed out waiting for JUnit report after %d attempts", maxRetries)
		}
		retries++
		err := c.GetJUnitReport(taskID, debugMode)
		if err == nil {
			// Report successfully downloaded
			return nil
		}

		// Check if we should keep waiting
		if strings.Contains(err.Error(), "still being generated") || strings.Contains(err.Error(), "API request failed with status 404") {
			if debugMode {
				fmt.Printf("Waiting for JUnit report to be generated... (attempt %d/%d)\n", retries, maxRetries)
			}
			time.Sleep(time.Duration(pollInterval) * time.Second)
			continue
		}

		// Any other error should be returned
		return err
	}
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
