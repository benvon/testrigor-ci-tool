package utils

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/benvon/testrigor-ci-tool/internal/api/types"
	"github.com/stretchr/testify/assert"
)

func TestGenerateFakeCommitHash(t *testing.T) {
	tests := []struct {
		name      string
		timestamp string
		expected  string
	}{
		{
			name:      "standard timestamp",
			timestamp: "20231201-143022",
			expected:  "66616b652023120114302200000000000000000000000000000000000000000000",
		},
		{
			name:      "timestamp with different format",
			timestamp: "2024-01-15-10-30-45",
			expected:  "66616b652024011510304500000000000000000000000000000000000000000000",
		},
		{
			name:      "short timestamp",
			timestamp: "20231201",
			expected:  "66616b652023120100000000000000000000000000000000000000000000000000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateFakeCommitHash(tt.timestamp)
			assert.Equal(t, 40, len(result))
			assert.True(t, result[:8] == "66616b65") // "fake" in hex
			assert.True(t, strings.HasSuffix(result, strings.Repeat("0", 40-len("66616b65"+strings.ReplaceAll(tt.timestamp, "-", "")))))
		})
	}
}

func TestParseErrorResponse(t *testing.T) {
	tests := []struct {
		name        string
		bodyBytes   []byte
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid error response",
			bodyBytes:   []byte(`{"message": "API error occurred"}`),
			expectError: true,
			errorMsg:    "API error: API error occurred",
		},
		{
			name:        "error response without message",
			bodyBytes:   []byte(`{"status": 400}`),
			expectError: true,
			errorMsg:    "unknown API error: {\"status\": 400}",
		},
		{
			name:        "invalid JSON",
			bodyBytes:   []byte(`invalid json`),
			expectError: true,
			errorMsg:    "error parsing error response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ParseErrorResponse(tt.bodyBytes)
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetString(t *testing.T) {
	m := map[string]interface{}{
		"string": "value",
		"int":    123,
		"float":  45.67,
		"nil":    nil,
	}

	tests := []struct {
		name     string
		key      string
		expected string
	}{
		{"string value", "string", "value"},
		{"int value", "int", ""},
		{"float value", "float", ""},
		{"nil value", "nil", ""},
		{"missing key", "missing", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetString(m, tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetInt(t *testing.T) {
	m := map[string]interface{}{
		"float64": float64(123),
		"int":     456,
		"string":  "789",
		"nil":     nil,
	}

	tests := []struct {
		name     string
		key      string
		expected int
	}{
		{"float64 value", "float64", 123},
		{"int value", "int", 456},
		{"string value", "string", 0},
		{"nil value", "nil", 0},
		{"missing key", "missing", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetInt(m, tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCheckTimeout(t *testing.T) {
	tests := []struct {
		name        string
		startTime   time.Time
		maxWaitTime time.Duration
		expectError bool
	}{
		{
			name:        "not timed out",
			startTime:   time.Now(),
			maxWaitTime: 1 * time.Hour,
			expectError: false,
		},
		{
			name:        "timed out",
			startTime:   time.Now().Add(-2 * time.Hour),
			maxWaitTime: 1 * time.Hour,
			expectError: true,
		},
		{
			name:        "exactly at timeout",
			startTime:   time.Now().Add(-1 * time.Hour),
			maxWaitTime: 1 * time.Hour,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckTimeout(tt.startTime, tt.maxWaitTime)
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "timed out")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestHandleStatusCheckError(t *testing.T) {
	tests := []struct {
		name                   string
		err                    error
		consecutiveErrors      int
		maxConsecutiveErrors   int
		debugMode              bool
		expectedContinue       bool
		expectedError          bool
		expectedConsecutiveErr int
	}{
		{
			name:                   "status 227 error",
			err:                    fmt.Errorf("status 227"),
			consecutiveErrors:      0,
			maxConsecutiveErrors:   5,
			debugMode:              false,
			expectedContinue:       true,
			expectedError:          false,
			expectedConsecutiveErr: 1,
		},
		{
			name:                   "status 228 error",
			err:                    fmt.Errorf("status 228"),
			consecutiveErrors:      0,
			maxConsecutiveErrors:   5,
			debugMode:              false,
			expectedContinue:       true,
			expectedError:          false,
			expectedConsecutiveErr: 1,
		},
		{
			name:                   "max consecutive errors reached",
			err:                    fmt.Errorf("status 227"),
			consecutiveErrors:      4,
			maxConsecutiveErrors:   5,
			debugMode:              false,
			expectedContinue:       false,
			expectedError:          true,
			expectedConsecutiveErr: 5,
		},
		{
			name:                   "test failed error",
			err:                    assert.AnError,
			consecutiveErrors:      0,
			maxConsecutiveErrors:   5,
			debugMode:              false,
			expectedContinue:       false,
			expectedError:          true,
			expectedConsecutiveErr: 0,
		},
		{
			name:                   "test crashed error",
			err:                    assert.AnError,
			consecutiveErrors:      0,
			maxConsecutiveErrors:   5,
			debugMode:              false,
			expectedContinue:       false,
			expectedError:          true,
			expectedConsecutiveErr: 0,
		},
		{
			name:                   "status 404 error",
			err:                    fmt.Errorf("API error (status 404): Test not found"),
			consecutiveErrors:      0,
			maxConsecutiveErrors:   5,
			debugMode:              false,
			expectedContinue:       true,
			expectedError:          false,
			expectedConsecutiveErr: 1,
		},
		{
			name:                   "status 404 error with different format",
			err:                    fmt.Errorf("status 404"),
			consecutiveErrors:      0,
			maxConsecutiveErrors:   5,
			debugMode:              false,
			expectedContinue:       true,
			expectedError:          false,
			expectedConsecutiveErr: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			consecutiveErrors := tt.consecutiveErrors
			continuePolling, err := HandleStatusCheckError(tt.err, &consecutiveErrors, tt.maxConsecutiveErrors, tt.debugMode)

			assert.Equal(t, tt.expectedContinue, continuePolling)
			assert.Equal(t, tt.expectedConsecutiveErr, consecutiveErrors)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCheckTestCompletion(t *testing.T) {
	tests := []struct {
		name           string
		status         *types.TestStatus
		debugMode      bool
		expectComplete bool
	}{
		{
			name: "all tests completed",
			status: &types.TestStatus{
				Results: types.TestResults{
					Total:      10,
					InQueue:    0,
					InProgress: 0,
					NotStarted: 0,
				},
			},
			debugMode:      false,
			expectComplete: true,
		},
		{
			name: "tests still in queue",
			status: &types.TestStatus{
				Results: types.TestResults{
					Total:      10,
					InQueue:    2,
					InProgress: 0,
					NotStarted: 0,
				},
			},
			debugMode:      false,
			expectComplete: false,
		},
		{
			name: "tests still in progress",
			status: &types.TestStatus{
				Results: types.TestResults{
					Total:      10,
					InQueue:    0,
					InProgress: 3,
					NotStarted: 0,
				},
			},
			debugMode:      false,
			expectComplete: false,
		},
		{
			name: "tests not started",
			status: &types.TestStatus{
				Results: types.TestResults{
					Total:      10,
					InQueue:    0,
					InProgress: 0,
					NotStarted: 5,
				},
			},
			debugMode:      false,
			expectComplete: false,
		},
		{
			name: "no tests total",
			status: &types.TestStatus{
				Results: types.TestResults{
					Total:      0,
					InQueue:    0,
					InProgress: 0,
					NotStarted: 0,
				},
			},
			debugMode:      false,
			expectComplete: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CheckTestCompletion(tt.status, tt.debugMode)
			assert.Equal(t, tt.expectComplete, result)
		})
	}
}

func TestIsTestInProgress(t *testing.T) {
	tests := []struct {
		name             string
		status           *types.TestStatus
		expectInProgress bool
	}{
		{
			name: "status in progress",
			status: &types.TestStatus{
				Status: types.StatusInProgress,
			},
			expectInProgress: true,
		},
		{
			name: "HTTP status 227",
			status: &types.TestStatus{
				Status:         "unknown",
				HTTPStatusCode: types.StatusTestInProgress227,
			},
			expectInProgress: true,
		},
		{
			name: "HTTP status 228",
			status: &types.TestStatus{
				Status:         "unknown",
				HTTPStatusCode: types.StatusTestInProgress228,
			},
			expectInProgress: true,
		},
		{
			name: "not in progress",
			status: &types.TestStatus{
				Status:         "completed",
				HTTPStatusCode: types.StatusOK,
			},
			expectInProgress: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsTestInProgress(tt.status)
			assert.Equal(t, tt.expectInProgress, result)
		})
	}
}

func TestHasTestCrashed(t *testing.T) {
	tests := []struct {
		name          string
		status        *types.TestStatus
		expectCrashed bool
	}{
		{
			name: "has crashes in results",
			status: &types.TestStatus{
				Results: types.TestResults{
					Crash: 2,
				},
			},
			expectCrashed: true,
		},
		{
			name: "has crash errors",
			status: &types.TestStatus{
				Results: types.TestResults{
					Crash: 0,
				},
				Errors: []types.TestError{
					{
						Category: types.ErrorCategoryCrash,
						Error:    "Test crashed",
					},
				},
			},
			expectCrashed: true,
		},
		{
			name: "no crashes",
			status: &types.TestStatus{
				Results: types.TestResults{
					Crash: 0,
				},
				Errors: []types.TestError{
					{
						Category: "ERROR",
						Error:    "Test failed",
					},
				},
			},
			expectCrashed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasTestCrashed(tt.status)
			assert.Equal(t, tt.expectCrashed, result)
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{"seconds", 45 * time.Second, "45s"},
		{"minutes", 5 * time.Minute, "5m"},
		{"hours", 2 * time.Hour, "2h"},
		{"mixed", 2*time.Hour + 30*time.Minute + 15*time.Second, "3h"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatDuration(tt.duration)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateTestRunOptions(t *testing.T) {
	tests := []struct {
		name        string
		opts        types.TestRunOptions
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid with test case UUIDs",
			opts: types.TestRunOptions{
				TestCaseUUIDs: []string{"uuid-1", "uuid-2"},
			},
			expectError: false,
		},
		{
			name: "valid with labels",
			opts: types.TestRunOptions{
				Labels: []string{"label1", "label2"},
			},
			expectError: false,
		},
		{
			name:        "no test case UUIDs or labels",
			opts:        types.TestRunOptions{},
			expectError: true,
			errorMsg:    "either TestCaseUUIDs or Labels must be provided",
		},
		{
			name: "both test case UUIDs and labels",
			opts: types.TestRunOptions{
				TestCaseUUIDs: []string{"uuid-1"},
				Labels:        []string{"label1"},
			},
			expectError: true,
			errorMsg:    "cannot specify both TestCaseUUIDs and Labels simultaneously",
		},
		{
			name: "invalid commit hash length",
			opts: types.TestRunOptions{
				Labels:     []string{"label1"},
				CommitHash: "short",
			},
			expectError: true,
			errorMsg:    "commit hash must be 40 characters long",
		},
		{
			name: "valid commit hash length",
			opts: types.TestRunOptions{
				Labels:     []string{"label1"},
				CommitHash: "1234567890123456789012345678901234567890",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTestRunOptions(tt.opts)
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
