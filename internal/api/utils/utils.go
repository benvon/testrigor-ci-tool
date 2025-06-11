package utils

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/benvon/testrigor-ci-tool/internal/api/types"
)

// GenerateFakeCommitHash generates a fake Git commit hash that looks real but is obviously fake
func GenerateFakeCommitHash(timestamp string) string {
	// Remove hyphens from timestamp
	cleanTimestamp := strings.ReplaceAll(timestamp, "-", "")
	// Start with "fake" in hex (66616b65) followed by timestamp and padded with zeros
	// This ensures it's 40 characters and starts with "fake" in hex
	base := fmt.Sprintf("66616b65%s", cleanTimestamp)
	// Pad with zeros to reach 40 characters
	return fmt.Sprintf("%s%s", base, strings.Repeat("0", 40-len(base)))
}

// ParseErrorResponse parses an error response from the API
func ParseErrorResponse(bodyBytes []byte) error {
	var result map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return fmt.Errorf("error parsing error response: %v", err)
	}

	if message, ok := result["message"].(string); ok {
		return fmt.Errorf("API error: %s", message)
	}

	return fmt.Errorf("unknown API error: %s", string(bodyBytes))
}

// GetString safely gets a string value from a map
func GetString(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

// GetInt safely gets an integer value from a map
func GetInt(m map[string]interface{}, key string) int {
	if val, ok := m[key].(float64); ok {
		return int(val)
	}
	if val, ok := m[key].(int); ok {
		return val
	}
	return 0
}

// CheckTimeout verifies if the maximum wait time has been exceeded
func CheckTimeout(startTime time.Time, maxWaitTime time.Duration) error {
	if time.Since(startTime) > maxWaitTime {
		return fmt.Errorf("test run timed out after %v", maxWaitTime)
	}
	return nil
}

// HandleStatusCheckError processes status check errors and returns whether to continue
func HandleStatusCheckError(err error, consecutiveErrors *int, maxConsecutiveErrors int, debugMode bool) (bool, error) {
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
	if strings.Contains(err.Error(), "test failed") || strings.Contains(err.Error(), "test crashed:") {
		return false, err
	}
	return false, err
}

// CheckTestCompletion verifies if all tests have completed execution
func CheckTestCompletion(status *types.TestStatus, debugMode bool) bool {
	if status.Results.Total > 0 &&
		status.Results.InQueue == 0 &&
		status.Results.InProgress == 0 &&
		status.Results.NotStarted == 0 {
		if debugMode {
			fmt.Printf("\nAll tests have finished execution. Final status: %s\n", status.Status)
		}
		return true
	}
	return false
}
