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
	cfg               *config.Config
	debugMode         bool
	httpClient        client.HTTPClient
	startTime         time.Time
	statusInterpreter *StatusInterpreter
}

// NewTestRigorClient creates a new TestRigor API client
func NewTestRigorClient(cfg *config.Config, httpClient ...client.HTTPClient) *TestRigorClient {
	var c client.HTTPClient
	if len(httpClient) > 0 && httpClient[0] != nil {
		c = httpClient[0]
	} else {
		c = client.NewDefaultHTTPClient()
	}
	return &TestRigorClient{
		cfg:               cfg,
		httpClient:        c,
		statusInterpreter: NewStatusInterpreter(),
	}
}

// SetDebugMode enables or disables debug output
func (c *TestRigorClient) SetDebugMode(debug bool) {
	c.debugMode = debug
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
		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			fmt.Printf("HTTP Status Code: %d\n", resp.StatusCode)
		}
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

	// Use centralized status interpreter
	return c.statusInterpreter.InterpretResponse(resp.StatusCode, bodyBytes)
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
		// Use labels to generate branch name if available, otherwise use fake-branch
		if len(opts.Labels) > 0 {
			// Join labels with hyphens and add timestamp for uniqueness
			labelPart := strings.Join(opts.Labels, "-")
			branchName = fmt.Sprintf("%s-%s", labelPart, timestamp)
		} else {
			branchName = fmt.Sprintf("fake-branch-%s", timestamp)
		}
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
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("auth-token", c.cfg.TestRigor.AuthToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %v", err)
	}

	// Use centralized status interpreter to parse the response
	return c.statusInterpreter.ParseTestStatus(resp.StatusCode, bodyBytes)
}

// printTestStatus prints the current test status and results
func (c *TestRigorClient) printTestStatus(status *types.TestStatus, reason string) {
	fmt.Printf("\n[%s] Current status: %s\n", reason, status.Status)
	if status.HTTPStatusCode != 0 && (status.HTTPStatusCode < 200 || status.HTTPStatusCode > 299) {
		fmt.Printf("HTTP Status Code: %d\n", status.HTTPStatusCode)
	}
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

			// Check if test has completed (either passed or failed)
			if status.Status == "Failed" || status.Results.Failed > 0 || status.HTTPStatusCode == 230 {
				statusManager.PrintFinalResults(status)
				return nil // Return nil to indicate normal completion, even if tests failed
			}

			if utils.CheckTestCompletion(status, debugMode) {
				statusManager.PrintFinalResults(status)
				return nil
			}
		}

		time.Sleep(time.Duration(pollInterval) * time.Second)
	}
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
	return c.WaitForJUnitReportWithRetries(taskID, pollInterval, debugMode, 60)
}

// WaitForJUnitReportWithRetries waits for the JUnit report to be ready and downloads it with configurable retries
func (c *TestRigorClient) WaitForJUnitReportWithRetries(taskID string, pollInterval int, debugMode bool, maxRetries int) error {
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

// formatHeaders formats HTTP headers for debug output
func formatHeaders(headers http.Header) string {
	var result strings.Builder
	for key, values := range headers {
		result.WriteString(fmt.Sprintf("%s: %s\n", key, strings.Join(values, ", ")))
	}
	return result.String()
}
