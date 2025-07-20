package api

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/benvon/testrigor-ci-tool/internal/api/types"
)

// StatusInterpreter handles all status interpretation logic for TestRigor API responses
type StatusInterpreter struct{}

// NewStatusInterpreter creates a new status interpreter
func NewStatusInterpreter() *StatusInterpreter {
	return &StatusInterpreter{}
}

// InterpretResponse interprets the HTTP response and returns appropriate result or error
func (si *StatusInterpreter) InterpretResponse(statusCode int, bodyBytes []byte) ([]byte, error) {
	var jsonCheck map[string]interface{}
	jsonErr := json.Unmarshal(bodyBytes, &jsonCheck)

	switch statusCode {
	case types.StatusOK:
		return bodyBytes, nil
	case types.StatusTestInProgress227, types.StatusTestInProgress228:
		// These are normal status codes indicating test is in progress
		return bodyBytes, fmt.Errorf("status %d", statusCode)
	case types.StatusTestFailed:
		return bodyBytes, fmt.Errorf("test failed")
	case types.StatusNotFound:
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
		return si.createAPIError(statusCode, bodyBytes, jsonCheck, jsonErr)
	case types.StatusBadRequest, types.StatusUnauthorized, types.StatusForbidden,
		types.StatusInternalServerError, types.StatusBadGateway, types.StatusServiceUnavailable,
		types.StatusGatewayTimeout:
		return si.createAPIError(statusCode, bodyBytes, jsonCheck, jsonErr)
	default:
		if jsonErr == nil {
			if msg, ok := jsonCheck["message"].(string); ok {
				return nil, fmt.Errorf("unexpected status code: %d, message: %s", statusCode, msg)
			}
		}
		return nil, fmt.Errorf("unexpected status code: %d", statusCode)
	}
}

// createAPIError creates a standardized API error message
func (si *StatusInterpreter) createAPIError(statusCode int, bodyBytes []byte, jsonCheck map[string]interface{}, jsonErr error) ([]byte, error) {
	if jsonErr != nil {
		return nil, fmt.Errorf("API error (status %d): %s", statusCode, string(bodyBytes))
	}
	msg := si.getString(jsonCheck, "message")
	details := si.extractErrorDetails(jsonCheck)
	if msg == "" && details == "" {
		return nil, fmt.Errorf("API error (status %d): %s", statusCode, string(bodyBytes))
	}
	return nil, fmt.Errorf("API error (status %d): %s, errors: %s", statusCode, msg, details)
}

// extractErrorDetails extracts error details from the JSON response
func (si *StatusInterpreter) extractErrorDetails(jsonCheck map[string]interface{}) string {
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

// ParseTestStatus parses a test status response from the API
func (si *StatusInterpreter) ParseTestStatus(statusCode int, bodyBytes []byte) (*types.TestStatus, error) {
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
					Category:    si.getString(errMap, "category"),
					Error:       si.getString(errMap, "error"),
					Occurrences: si.getInt(errMap, "occurrences"),
					Severity:    si.getString(errMap, "severity"),
					DetailsURL:  si.getString(errMap, "detailsUrl"),
				}
				errors = append(errors, testErr)
			} else if errStr, ok := err.(string); ok {
				// Handle string errors (like CRASH messages)
				testErr := types.TestError{
					Category:    types.ErrorCategoryCrash,
					Error:       errStr,
					Occurrences: 1,
					Severity:    types.ErrorCategoryBlocker,
				}
				errors = append(errors, testErr)
			}
		}
	}

	// Parse overall results if present
	var results types.TestResults
	if resultsMap, ok := result["overallResults"].(map[string]interface{}); ok {
		results = types.TestResults{
			Total:      si.getInt(resultsMap, "Total"),
			InQueue:    si.getInt(resultsMap, "In queue"),
			InProgress: si.getInt(resultsMap, "In progress"),
			Failed:     si.getInt(resultsMap, "Failed"),
			Passed:     si.getInt(resultsMap, "Passed"),
			Canceled:   si.getInt(resultsMap, "Canceled"),
			NotStarted: si.getInt(resultsMap, "Not started"),
			Crash:      si.getInt(resultsMap, "Crash"),
		}
	}

	testStatus := &types.TestStatus{
		Status:         status,
		DetailsURL:     detailsURL,
		TaskID:         taskID,
		Errors:         errors,
		Results:        results,
		HTTPStatusCode: statusCode,
	}

	// Check for test crashes in the errors
	for _, err := range errors {
		if strings.Contains(err.Error, "CRASH:") {
			return testStatus, fmt.Errorf("test crashed: %s", err.Error)
		}
	}

	return testStatus, nil
}

// IsValidStatusResponse determines if the status code represents a valid response
func (si *StatusInterpreter) IsValidStatusResponse(statusCode int) bool {
	validCodes := map[int]bool{
		types.StatusOK:                true,
		types.StatusTestInProgress227: true,
		types.StatusTestInProgress228: true,
		types.StatusTestFailed:        true,
	}
	return validCodes[statusCode]
}

// Helper functions for parsing JSON values
func (si *StatusInterpreter) getString(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

func (si *StatusInterpreter) getInt(m map[string]interface{}, key string) int {
	if val, ok := m[key].(float64); ok {
		return int(val)
	}
	if val, ok := m[key].(int); ok {
		return val
	}
	return 0
}
