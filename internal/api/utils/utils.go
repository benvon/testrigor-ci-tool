package utils

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/benvon/testrigor-ci-tool/internal/api/types"
)

// GenerateFakeCommitHash generates a fake Git commit hash that looks real but is obviously fake.
// This is used when no commit hash is provided but branch information is needed.
func GenerateFakeCommitHash(timestamp string) string {
	// Remove hyphens from timestamp
	cleanTimestamp := strings.ReplaceAll(timestamp, "-", "")
	// Start with "fake" in hex (66616b65) followed by timestamp and padded with zeros
	// This ensures it's 40 characters and starts with "fake" in hex
	base := fmt.Sprintf("66616b65%s", cleanTimestamp)
	// Pad with zeros to reach 40 characters
	return fmt.Sprintf("%s%s", base, strings.Repeat("0", 40-len(base)))
}

// ParseErrorResponse parses an error response from the API and returns a formatted error message.
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

// GetString safely gets a string value from a map, returning empty string if not found or wrong type.
func GetString(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

// GetInt safely gets an integer value from a map, returning 0 if not found or wrong type.
func GetInt(m map[string]interface{}, key string) int {
	if val, ok := m[key].(float64); ok {
		return int(val)
	}
	if val, ok := m[key].(int); ok {
		return val
	}
	return 0
}

// CheckTimeout verifies if the maximum wait time has been exceeded.
func CheckTimeout(startTime time.Time, maxWaitTime time.Duration) error {
	if time.Since(startTime) > maxWaitTime {
		return fmt.Errorf("test run timed out after %v", maxWaitTime)
	}
	return nil
}

// HandleStatusCheckError processes status check errors and returns whether to continue polling.
// It handles specific error cases like test in progress status codes and test failures.
func HandleStatusCheckError(err error, consecutiveErrors *int, maxConsecutiveErrors int, debugMode bool) (bool, error) {
	// Check for test in progress status codes
	if strings.Contains(err.Error(), fmt.Sprintf("status %d", types.StatusTestInProgress227)) ||
		strings.Contains(err.Error(), fmt.Sprintf("status %d", types.StatusTestInProgress228)) {
		*consecutiveErrors++
		if *consecutiveErrors >= maxConsecutiveErrors {
			return false, fmt.Errorf("received %d consecutive errors while checking test status: %v", *consecutiveErrors, err)
		}
		if debugMode {
			fmt.Printf("Error checking status (attempt %d/%d): %v\n", *consecutiveErrors, maxConsecutiveErrors, err)
		}
		return true, nil
	}

	// Check for test failure or crash
	if strings.Contains(err.Error(), "test failed") || strings.Contains(err.Error(), "test crashed:") {
		return false, err
	}

	return false, err
}

// CheckTestCompletion verifies if all tests have completed execution.
// Returns true if all tests are finished (no tests in queue, in progress, or not started).
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

// IsTestInProgress checks if the test is currently in progress based on status or HTTP status code.
func IsTestInProgress(status *types.TestStatus) bool {
	return status.IsInProgress()
}

// HasTestCrashed checks if any tests have crashed based on the results or error messages.
func HasTestCrashed(status *types.TestStatus) bool {
	return status.HasCrashes() || len(status.GetCrashErrors()) > 0
}

// FormatDuration formats a duration in a human-readable format.
func FormatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.0fm", d.Minutes())
	}
	return fmt.Sprintf("%.0fh", d.Hours())
}

// ValidateTestRunOptions validates the test run options and returns an error if invalid.
func ValidateTestRunOptions(opts types.TestRunOptions) error {
	// Check that either test case UUIDs or labels are provided
	if len(opts.TestCaseUUIDs) == 0 && len(opts.Labels) == 0 {
		return fmt.Errorf("either TestCaseUUIDs or Labels must be provided")
	}

	// Check that both test case UUIDs and labels are not provided simultaneously
	if len(opts.TestCaseUUIDs) > 0 && len(opts.Labels) > 0 {
		return fmt.Errorf("cannot specify both TestCaseUUIDs and Labels simultaneously")
	}

	// Validate commit hash format if provided
	if opts.CommitHash != "" && len(opts.CommitHash) != 40 {
		return fmt.Errorf("commit hash must be 40 characters long")
	}

	return nil
}
