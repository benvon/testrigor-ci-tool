// Package client provides TestRigor API client primitives.
package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"strconv"

	"github.com/benvon/testrigor-ci-tool/internal/api/types"
	"github.com/benvon/testrigor-ci-tool/internal/config"
)

// TestRigorClient is a primitive client for TestRigor API operations.
type TestRigorClient struct {
	httpClient *Client
	config     *config.Config
}

// NewTestRigorClient creates a new TestRigor API client.
func NewTestRigorClient(cfg *config.Config, httpClient HTTPClient) *TestRigorClient {
	return &TestRigorClient{
		httpClient: New(httpClient),
		config:     cfg,
	}
}

// StartTestRun starts a new test run. This is a primitive API operation.
func (c *TestRigorClient) StartTestRun(ctx context.Context, opts types.TestRunOptions, debugMode bool) (*types.TestRunResult, error) {
	body := c.buildStartTestRunBody(opts)
	branchName := c.extractBranchName(opts, body)

	headers := map[string]string{
		"Accept":     "application/json",
		"auth-token": c.config.TestRigor.AuthToken,
	}

	resp, err := c.httpClient.Execute(ctx, Request{
		Method:      "POST",
		URL:         fmt.Sprintf("%s/apps/%s/retest", c.config.TestRigor.APIURL, c.config.TestRigor.AppID),
		Body:        body,
		Headers:     headers,
		ContentType: "application/json",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start test run: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, c.parseAPIError(resp.StatusCode, resp.Body)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	taskID, ok := result["taskId"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid response: missing taskId")
	}

	return &types.TestRunResult{
		TaskID:     taskID,
		BranchName: branchName,
	}, nil
}

// GetTestStatus retrieves the current test status. This is a primitive API operation.
func (c *TestRigorClient) GetTestStatus(ctx context.Context, branchName string, labels []string, debugMode bool) (*types.TestStatus, error) {
	requestURL := c.buildStatusURL(branchName, labels)

	headers := map[string]string{
		"Accept":       "application/json",
		"Content-Type": "application/json",
		"auth-token":   c.config.TestRigor.AuthToken,
	}

	resp, err := c.httpClient.Execute(ctx, Request{
		Method:  "GET",
		URL:     requestURL,
		Headers: headers,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get test status: %w", err)
	}

	return c.parseTestStatus(resp.StatusCode, resp.Body, debugMode)
}

// CancelTestRun cancels a running test. This is a primitive API operation.
func (c *TestRigorClient) CancelTestRun(ctx context.Context, runID string) error {
	headers := map[string]string{
		"Accept":     "application/json",
		"auth-token": c.config.TestRigor.AuthToken,
	}

	resp, err := c.httpClient.Execute(ctx, Request{
		Method:      "PUT",
		URL:         fmt.Sprintf("%s/apps/%s/runs/%s/cancel", c.config.TestRigor.APIURL, c.config.TestRigor.AppID, runID),
		Headers:     headers,
		ContentType: "application/json",
	})
	if err != nil {
		return fmt.Errorf("failed to cancel test run: %w", err)
	}

	if resp.StatusCode != 200 {
		return c.parseAPIError(resp.StatusCode, resp.Body)
	}

	return nil
}

// GetJUnitReport downloads the JUnit report. This is a primitive API operation.
func (c *TestRigorClient) GetJUnitReport(ctx context.Context, taskID string) ([]byte, error) {
	headers := map[string]string{
		"auth-token": c.config.TestRigor.AuthToken,
	}

	resp, err := c.httpClient.Execute(ctx, Request{
		Method:      "GET",
		URL:         fmt.Sprintf("https://api2.testrigor.com/api/v1/apps/%s/runs/%s/junit_report", c.config.TestRigor.AppID, taskID),
		Headers:     headers,
		ContentType: "application/xml",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get JUnit report: %w", err)
	}

	if resp.StatusCode == 404 {
		// Check if it's a "report not ready" error
		var errorResp map[string]interface{}
		if json.Unmarshal(resp.Body, &errorResp) == nil {
			if msg, ok := errorResp["message"].(string); ok && strings.Contains(msg, "Report still being generated") {
				return nil, fmt.Errorf("report still being generated")
			}
		}
		return nil, fmt.Errorf("report not found")
	}

	if resp.StatusCode != 200 {
		return nil, c.parseAPIError(resp.StatusCode, resp.Body)
	}

	return resp.Body, nil
}

// buildStartTestRunBody constructs the request body for starting a test run.
func (c *TestRigorClient) buildStartTestRunBody(opts types.TestRunOptions) map[string]interface{} {
	body := map[string]interface{}{
		"forceCancelPreviousTesting": opts.ForceCancelPreviousTesting,
		"skipXrayCloud":              !opts.MakeXrayReports,
	}

	if len(opts.TestCaseUUIDs) > 0 {
		body["testCaseUuids"] = opts.TestCaseUUIDs
		if opts.URL != "" {
			body["url"] = opts.URL
		}
		return body
	}

	branchInfo := c.buildBranchInfo(opts)
	if branchInfo != nil {
		body["branch"] = branchInfo
	}

	if len(opts.Labels) > 0 {
		body["labels"] = opts.Labels
		body["excludedLabels"] = opts.ExcludedLabels
	}

	if opts.URL != "" && branchInfo != nil {
		body["url"] = opts.URL
	}

	if opts.CustomName != "" {
		body["customName"] = opts.CustomName
	}

	return body
}

// buildBranchInfo constructs branch information for the request.
func (c *TestRigorClient) buildBranchInfo(opts types.TestRunOptions) map[string]string {
	if opts.CommitHash != "" && opts.BranchName == "" {
		return nil
	}

	if opts.BranchName == "" && len(opts.Labels) == 0 {
		return nil
	}

	branchName := opts.BranchName
	if branchName == "" {
		branchName = c.generateBranchName(opts.Labels)
	}

	branchInfo := map[string]string{
		"name": branchName,
	}

	if opts.CommitHash != "" {
		branchInfo["commit"] = opts.CommitHash
	} else {
		branchInfo["commit"] = c.generateFakeCommitHash()
	}

	return branchInfo
}

// extractBranchName extracts the branch name from options or request body.
func (c *TestRigorClient) extractBranchName(opts types.TestRunOptions, body map[string]interface{}) string {
	if opts.BranchName != "" {
		return opts.BranchName
	}

	if branch, ok := body["branch"].(map[string]string); ok {
		if name, ok := branch["name"]; ok {
			return name
		}
	}

	return ""
}

// buildStatusURL constructs the URL for status requests.
func (c *TestRigorClient) buildStatusURL(branchName string, labels []string) string {
	baseURL := fmt.Sprintf("%s/apps/%s/status", c.config.TestRigor.APIURL, c.config.TestRigor.AppID)

	params := url.Values{}
	if branchName != "" {
		params.Set("branchName", branchName)
	}
	if len(labels) > 0 {
		params.Set("labels", strings.Join(labels, ","))
	}

	if len(params) > 0 {
		return baseURL + "?" + params.Encode()
	}
	return baseURL
}

// parseTestStatus parses the test status response.
func (c *TestRigorClient) parseTestStatus(statusCode int, body []byte, debugMode bool) (*types.TestStatus, error) {
	status := &types.TestStatus{HTTPStatusCode: statusCode}

	switch statusCode {
	case 200:
		status.Status = "completed"
	case 227:
		status.Status = "new"
	case 228:
		status.Status = "in_progress"
	case 230:
		status.Status = "failed"
	case 404:
		status.Status = "not_found"
		return nil, fmt.Errorf("test not found or not ready")
	case 400, 401, 403, 500, 502, 503, 504:
		status.Status = "error"
		return nil, c.parseAPIError(statusCode, body)
	}

	// For 200, 227, 228, 230, parse the body for extended details
	c.parseStatusBody(body, status, debugMode)
	return status, nil
}

// parseStatusBody parses the JSON body into a TestStatus struct.
func (c *TestRigorClient) parseStatusBody(body []byte, status *types.TestStatus, debugMode bool) {
	var data map[string]interface{}
	if json.Unmarshal(body, &data) != nil {
		return
	}

	if statusStr, ok := data["status"].(string); ok {
		status.Status = statusStr
	}

	if detailsURL, ok := data["detailsUrl"].(string); ok {
		status.DetailsURL = detailsURL
	}

	if taskID, ok := data["taskId"].(string); ok {
		status.TaskID = taskID
	}

	// Parse results
	if results, ok := data["overallResults"].(map[string]interface{}); ok {
		// Debug: print the raw results map if any field is zero and status is not new
		if debugMode && status.Status != "new" {
			zeroFields := []string{}
			fields := []string{"Total", "total", "In queue", "inQueue", "queued", "In progress", "inProgress", "running", "Passed", "passed", "Failed", "failed", "Not started", "notStarted", "Canceled", "canceled", "cancelled", "Crash", "crash"}
			for _, f := range fields {
				if c.getInt(results, f) == 0 {
					zeroFields = append(zeroFields, f)
				}
			}
			if len(zeroFields) > 0 {
				fmt.Printf("[testrigor-ci-tool debug] API overallResults: %+v\n", results)
			}
		}
		status.Results = types.TestResults{
			Total:      c.getInt(results, "Total") + c.getInt(results, "total"),
			InQueue:    c.getInt(results, "In queue") + c.getInt(results, "inQueue") + c.getInt(results, "queued"),
			InProgress: c.getInt(results, "In progress") + c.getInt(results, "inProgress") + c.getInt(results, "running"),
			Passed:     c.getInt(results, "Passed") + c.getInt(results, "passed"),
			Failed:     c.getInt(results, "Failed") + c.getInt(results, "failed"),
			NotStarted: c.getInt(results, "Not started") + c.getInt(results, "notStarted"),
			Canceled:   c.getInt(results, "Canceled") + c.getInt(results, "canceled") + c.getInt(results, "cancelled"),
			Crash:      c.getInt(results, "Crash") + c.getInt(results, "crash"),
		}
	}

	// Parse errors
	if errors, ok := data["errors"].([]interface{}); ok {
		for _, errItem := range errors {
			if errMap, ok := errItem.(map[string]interface{}); ok {
				testError := types.TestError{
					Category:    c.getString(errMap, "category"),
					Error:       c.getString(errMap, "error"),
					Severity:    c.getString(errMap, "severity"),
					Occurrences: c.getInt(errMap, "occurrences"),
					DetailsURL:  c.getString(errMap, "detailsUrl"),
				}
				status.Errors = append(status.Errors, testError)
			}
		}
	}
}

// parseAPIError parses API error responses.
func (c *TestRigorClient) parseAPIError(statusCode int, body []byte) error {
	var errorResp map[string]interface{}
	if json.Unmarshal(body, &errorResp) == nil {
		if msg, ok := errorResp["message"].(string); ok {
			return fmt.Errorf("API error (status %d): %s", statusCode, msg)
		}
	}
	return fmt.Errorf("API error (status %d): %s", statusCode, string(body))
}

// Helper functions for safe type conversion
func (c *TestRigorClient) getString(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

func (c *TestRigorClient) getInt(m map[string]interface{}, key string) int {
	if val, ok := m[key]; ok {
		switch v := val.(type) {
		case float64:
			return int(v)
		case int:
			return v
		case string:
			i, _ := strconv.Atoi(v)
			return i
		}
	}
	return 0
}

// generateBranchName generates a branch name from labels.
func (c *TestRigorClient) generateBranchName(labels []string) string {
	timestamp := fmt.Sprintf("%d", time.Now().Unix()) // Generate current Unix timestamp
	if len(labels) > 0 {
		labelPart := strings.Join(labels, "-")
		return fmt.Sprintf("%s-%s", labelPart, timestamp)
	}
	return fmt.Sprintf("fake-branch-%s", timestamp)
}

// generateFakeCommitHash generates a fake commit hash.
func (c *TestRigorClient) generateFakeCommitHash() string {
	// Simplified implementation - real implementation would use utils
	return "66616b6512345678901234567890123456789012"
}
