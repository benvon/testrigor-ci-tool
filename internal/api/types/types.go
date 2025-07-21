package types

import (
	"strings"
)

// TestRigor status codes and states
const (
	// HTTP Status Codes
	StatusOK                  = 200
	StatusTestInProgress227   = 227
	StatusTestInProgress228   = 228
	StatusTestFailed          = 230
	StatusBadRequest          = 400
	StatusUnauthorized        = 401
	StatusForbidden           = 403
	StatusNotFound            = 404
	StatusInternalServerError = 500
	StatusBadGateway          = 502
	StatusServiceUnavailable  = 503
	StatusGatewayTimeout      = 504

	// Test Status States
	StatusCompleted  = "completed"
	StatusFailed     = "failed"
	StatusError      = "error"
	StatusCancelled  = "cancelled"
	StatusCanceled   = "canceled"
	StatusInProgress = "in_progress"
	StatusInQueue    = "in_queue"
	StatusNotStarted = "not_started"

	// Error Categories
	ErrorCategoryCrash   = "CRASH"
	ErrorCategoryBlocker = "BLOCKER"
)

// TestRunOptions represents the options for starting a test run
type TestRunOptions struct {
	// TestCaseUUIDs specifies the UUIDs of specific test cases to run
	TestCaseUUIDs []string
	// Labels specifies the labels to filter tests by
	Labels []string
	// ExcludedLabels specifies the labels to exclude from the test run
	ExcludedLabels []string
	// URL specifies the base URL for the test run
	URL string
	// BranchName specifies the Git branch name for the test run
	BranchName string
	// CommitHash specifies the Git commit hash for the test run
	CommitHash string
	// CustomName specifies a custom name for the test run
	CustomName string
	// ForceCancelPreviousTesting forces cancellation of any previous test runs
	ForceCancelPreviousTesting bool
	// MakeXrayReports enables Xray report generation
	MakeXrayReports bool
}

// TestRunResult represents the result of starting a test run
type TestRunResult struct {
	// TaskID is the unique identifier for the test run task
	TaskID string
	// BranchName is the branch name associated with the test run
	BranchName string
}

// TestError represents an error that occurred during a test run
type TestError struct {
	// Category is the error category (e.g., "CRASH", "BLOCKER")
	Category string
	// Error is the error message
	Error string
	// Occurrences is the number of times this error occurred
	Occurrences int
	// Severity is the error severity level
	Severity string
	// DetailsURL is the URL to view detailed error information
	DetailsURL string
}

// TestResults represents the overall results of a test run
type TestResults struct {
	// Total is the total number of tests
	Total int
	// InQueue is the number of tests waiting in queue
	InQueue int
	// InProgress is the number of tests currently running
	InProgress int
	// Failed is the number of tests that failed
	Failed int
	// Passed is the number of tests that passed
	Passed int
	// Canceled is the number of tests that were canceled
	Canceled int
	// NotStarted is the number of tests that haven't started
	NotStarted int
	// Crash is the number of tests that crashed
	Crash int
}

// TestStatus represents the current status of a test run
type TestStatus struct {
	// Status is the overall status of the test run
	Status string
	// DetailsURL is the URL to view detailed test information
	DetailsURL string
	// TaskID is the unique identifier for the test run task
	TaskID string
	// Errors contains any errors that occurred during the test run
	Errors []TestError
	// Results contains the overall test results
	Results TestResults
	// HTTPStatusCode is the HTTP status code from the API response
	HTTPStatusCode int
}

// IsComplete returns true if the test status indicates completion
func (ts *TestStatus) IsComplete() bool {
	switch strings.ToLower(ts.Status) {
	case "completed", "failed", "error", "cancelled", "canceled":
		return true
	}
	return false
}

// IsInProgress returns true if the test is currently running
func (ts *TestStatus) IsInProgress() bool {
	return ts.Status == StatusInProgress ||
		ts.HTTPStatusCode == StatusTestInProgress227 ||
		ts.HTTPStatusCode == StatusTestInProgress228
}

// HasCrashes returns true if any tests have crashed
func (ts *TestStatus) HasCrashes() bool {
	return ts.Results.Crash > 0
}

// HasErrors returns true if there are any errors
func (ts *TestStatus) HasErrors() bool {
	return len(ts.Errors) > 0
}

// GetCrashErrors returns all crash-related errors
func (ts *TestStatus) GetCrashErrors() []TestError {
	var crashErrors []TestError
	for _, err := range ts.Errors {
		if err.Category == ErrorCategoryCrash ||
			(err.Error != "" && (err.Error == "test crashed" ||
				err.Error == "test failed")) {
			crashErrors = append(crashErrors, err)
		}
	}
	return crashErrors
}
