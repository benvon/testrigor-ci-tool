package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/benvon/testrigor-ci-tool/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockHTTPClient is a mock implementation of the HTTPClient interface
type MockHTTPClient struct {
	mock.Mock
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	args := m.Called(req)
	return args.Get(0).(*http.Response), args.Error(1)
}

func TestNewTestRigorClient(t *testing.T) {
	cfg := &config.Config{
		TestRigor: config.TestRigorConfig{
			AuthToken: "test-token",
			AppID:     "test-app",
			APIURL:    "https://api.testrigor.com/api/v1",
		},
	}

	client := NewTestRigorClient(cfg)
	assert.NotNil(t, client)
	assert.Equal(t, cfg, client.cfg)
	assert.False(t, client.debugMode)
}

func TestSetDebugMode(t *testing.T) {
	client := NewTestRigorClient(&config.Config{})
	client.SetDebugMode(true)
	assert.True(t, client.debugMode)
	client.SetDebugMode(false)
	assert.False(t, client.debugMode)
}

func TestStartTestRun(t *testing.T) {
	tests := []struct {
		name          string
		opts          TestRunOptions
		mockResponse  interface{}
		expectedError bool
		checkResult   func(*testing.T, *TestRunResult)
	}{
		{
			name: "successful test run with test case UUIDs",
			opts: TestRunOptions{
				TestCaseUUIDs: []string{"test-uuid-1"},
				URL:           "https://example.com",
			},
			mockResponse: map[string]interface{}{
				"taskId":     "task-123",
				"branchName": "test-branch",
			},
			expectedError: false,
			checkResult: func(t *testing.T, result *TestRunResult) {
				assert.Equal(t, "task-123", result.TaskID)
				assert.Equal(t, "", result.BranchName)
			},
		},
		{
			name: "error response",
			opts: TestRunOptions{
				TestCaseUUIDs: []string{"test-uuid-1"},
			},
			mockResponse: map[string]interface{}{
				"status":  400,
				"message": "Invalid request",
				"errors":  []string{"Invalid test case UUID"},
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock HTTP client
			mockClient := new(MockHTTPClient)

			// Prepare mock response
			responseBody, _ := json.Marshal(tt.mockResponse)
			mockResponse := &http.Response{
				StatusCode: 200,
				Body:       &mockReadCloser{data: responseBody},
			}

			// Set up mock expectations
			mockClient.On("Do", mock.Anything).Return(mockResponse, nil)

			// Create test client with mock HTTP client
			client := NewTestRigorClient(&config.Config{
				TestRigor: config.TestRigorConfig{
					AuthToken: "test-token",
					AppID:     "test-app",
					APIURL:    "https://api.testrigor.com/api/v1",
				},
			}, mockClient)

			// Execute test
			result, err := client.StartTestRun(tt.opts)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				if tt.checkResult != nil {
					tt.checkResult(t, result)
				}
			}

			// Verify that all expectations were met
			mockClient.AssertExpectations(t)
		})
	}
}

// mockReadCloser is a simple implementation of io.ReadCloser for testing
type mockReadCloser struct {
	data []byte
	pos  int
}

func (m *mockReadCloser) Read(p []byte) (n int, err error) {
	if m.pos >= len(m.data) {
		return 0, io.EOF
	}
	n = copy(p, m.data[m.pos:])
	m.pos += n
	return n, nil
}

func (m *mockReadCloser) Close() error {
	m.pos = 0 // Reset position on close
	return nil
}

func TestGenerateFakeCommitHash(t *testing.T) {
	timestamp := "2024-05-12-141314"
	hash := generateFakeCommitHash(timestamp)

	assert.Len(t, hash, 40)
	assert.True(t, strings.HasPrefix(hash, "66616b65"))
	assert.Contains(t, hash, "20240512141314")
}

func TestIsComplete(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected bool
	}{
		{"completed", "Completed", true},
		{"failed", "Failed", true},
		{"canceled", "Canceled", true},
		{"in progress", "In progress", false},
		{"new", "New", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, isComplete(tt.status))
		})
	}
}

func TestGetString(t *testing.T) {
	m := map[string]interface{}{
		"string": "value",
		"int":    123,
		"nil":    nil,
	}

	assert.Equal(t, "value", getString(m, "string"))
	assert.Equal(t, "", getString(m, "int"))
	assert.Equal(t, "", getString(m, "nil"))
	assert.Equal(t, "", getString(m, "nonexistent"))
}

func TestGetInt(t *testing.T) {
	m := map[string]interface{}{
		"int":    123,
		"string": "value",
		"nil":    nil,
	}

	assert.Equal(t, 123, getInt(m, "int"))
	assert.Equal(t, 0, getInt(m, "string"))
	assert.Equal(t, 0, getInt(m, "nil"))
	assert.Equal(t, 0, getInt(m, "nonexistent"))
}

func TestMakeRequestErrorHandling(t *testing.T) {
	tests := []struct {
		name           string
		responseBody   string
		responseStatus int
		expectedError  string
		shouldError    bool
	}{
		{
			name:           "malformed JSON with 500 status",
			responseBody:   `{"status": "error", "message": "Invalid request", "errors": [{"invalid json"}`,
			responseStatus: 500,
			expectedError:  "API error (status 500):",
			shouldError:    true,
		},
		{
			name:           "malformed JSON with 200 status",
			responseBody:   `{"status": "success", "data": {"invalid json`,
			responseStatus: 200,
			expectedError:  "", // Should not error, return raw response
			shouldError:    false,
		},
		{
			name:           "malformed JSON with 227 status",
			responseBody:   `{"status": "new", "data": {"invalid json`,
			responseStatus: 227,
			expectedError:  "test in progress",
			shouldError:    true,
		},
		{
			name:           "malformed JSON with 228 status",
			responseBody:   `{"status": "in_progress", "data": {"invalid json`,
			responseStatus: 228,
			expectedError:  "test in progress",
			shouldError:    true,
		},
		{
			name:           "malformed JSON with 230 status",
			responseBody:   `{"status": "failed", "data": {"invalid json`,
			responseStatus: 230,
			expectedError:  "test failed",
			shouldError:    true,
		},
		{
			name:           "non-JSON response",
			responseBody:   "Internal Server Error",
			responseStatus: 500,
			expectedError:  "API error (status 500): Internal Server Error",
			shouldError:    true,
		},
		{
			name:           "empty response",
			responseBody:   "",
			responseStatus: 200,
			expectedError:  "empty response body",
			shouldError:    true,
		},
		{
			name:           "test in progress (227)",
			responseBody:   `{"status": "in_progress"}`,
			responseStatus: 227,
			expectedError:  "test in progress",
			shouldError:    true,
		},
		{
			name:           "test in progress (228)",
			responseBody:   `{"status": "in_progress"}`,
			responseStatus: 228,
			expectedError:  "test in progress",
			shouldError:    true,
		},
		{
			name:           "test failed (230)",
			responseBody:   `{"status": "failed"}`,
			responseStatus: 230,
			expectedError:  "test failed",
			shouldError:    true,
		},
		{
			name:           "API error 400",
			responseBody:   `{"message": "Bad Request", "errors": ["Invalid input"]}`,
			responseStatus: 400,
			expectedError:  "API error (status 400): Bad Request, errors: Invalid input",
			shouldError:    true,
		},
		{
			name:           "API error 401",
			responseBody:   `{"message": "Unauthorized"}`,
			responseStatus: 401,
			expectedError:  "API error (status 401): Unauthorized",
			shouldError:    true,
		},
		{
			name:           "API error 404",
			responseBody:   `{"message": "Not Found"}`,
			responseStatus: 404,
			expectedError:  "API error (status 404): Not Found",
			shouldError:    true,
		},
		{
			name:           "API error 500",
			responseBody:   `{"message": "Internal Server Error"}`,
			responseStatus: 500,
			expectedError:  "API error (status 500): Internal Server Error",
			shouldError:    true,
		},
		{
			name:           "successful response",
			responseBody:   `{"status": "success"}`,
			responseStatus: 200,
			expectedError:  "",
			shouldError:    false,
		},
		{
			name:           "unexpected status code",
			responseBody:   `{"status": "unknown"}`,
			responseStatus: 999,
			expectedError:  "unexpected status code: 999",
			shouldError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock HTTP client
			mockClient := new(MockHTTPClient)

			// Prepare mock response
			var responseBody io.ReadCloser
			if tt.responseBody != "" {
				responseBody = &mockReadCloser{data: []byte(tt.responseBody)}
			} else {
				responseBody = &mockReadCloser{data: []byte{}}
			}

			mockResponse := &http.Response{
				StatusCode: tt.responseStatus,
				Body:       responseBody,
			}

			// Set up mock expectations
			mockClient.On("Do", mock.Anything).Return(mockResponse, nil).Once()

			// Create test client with mock HTTP client
			client := NewTestRigorClient(&config.Config{
				TestRigor: config.TestRigorConfig{
					AuthToken: "test-token",
					AppID:     "test-app",
					APIURL:    "https://api.testrigor.com/api/v1",
				},
			}, mockClient)

			// Execute test
			_, err := client.makeRequest(requestOptions{
				method:      "GET",
				url:         "https://api.testrigor.com/api/v1/test",
				contentType: "application/json",
			})

			if tt.shouldError {
				assert.Error(t, err)
				if err != nil {
					assert.Contains(t, err.Error(), tt.expectedError)
				}
			} else {
				assert.NoError(t, err)
			}

			// Verify that all expectations were met
			mockClient.AssertExpectations(t)
		})
	}
}

func TestPrintTestStatus(t *testing.T) {
	client := NewTestRigorClient(&config.Config{})
	status := &TestStatus{
		Status: "In Progress",
		Results: TestResults{
			Passed:     5,
			Failed:     2,
			InProgress: 3,
			InQueue:    1,
		},
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	client.printTestStatus(status, "test reason")

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read captured output
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	assert.Contains(t, output, "[test reason] Current status: In Progress")
	assert.Contains(t, output, "Passed: 5, Failed: 2, In Progress: 3, In Queue: 1")
}

func TestPrintFinalResults(t *testing.T) {
	client := NewTestRigorClient(&config.Config{})
	status := &TestStatus{
		Status:     "Completed",
		DetailsURL: "https://testrigor.com/details/123",
		Results: TestResults{
			Total:      11,
			Passed:     5,
			Failed:     2,
			InProgress: 0,
			InQueue:    0,
			NotStarted: 1,
			Canceled:   2,
			Crash:      1,
		},
		Errors: []TestError{
			{
				Category:    "Test Error",
				Error:       "Test failed",
				Severity:    "High",
				Occurrences: 1,
				DetailsURL:  "https://testrigor.com/error/123",
			},
		},
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	client.printFinalResults(status)

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read captured output
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	assert.Contains(t, output, "Test run completed with status: Completed")
	assert.Contains(t, output, "Details URL: https://testrigor.com/details/123")
	assert.Contains(t, output, "Total: 11")
	assert.Contains(t, output, "Passed: 5")
	assert.Contains(t, output, "Failed: 2")
	assert.Contains(t, output, "Category: Test Error")
	assert.Contains(t, output, "Error: Test failed")
}

func TestShouldPrintStatus(t *testing.T) {
	client := NewTestRigorClient(&config.Config{})
	now := time.Now()

	tests := []struct {
		name        string
		status      *TestStatus
		lastStatus  string
		lastResults TestResults
		lastUpdate  time.Time
		expected    bool
		reason      string
	}{
		{
			name: "status changed",
			status: &TestStatus{
				Status:  "In Progress",
				Results: TestResults{Passed: 1},
			},
			lastStatus:  "New",
			lastResults: TestResults{},
			lastUpdate:  now,
			expected:    true,
			reason:      "status changed",
		},
		{
			name: "results updated",
			status: &TestStatus{
				Status:  "In Progress",
				Results: TestResults{Passed: 2},
			},
			lastStatus:  "In Progress",
			lastResults: TestResults{Passed: 1},
			lastUpdate:  now,
			expected:    true,
			reason:      "results updated",
		},
		{
			name: "periodic update",
			status: &TestStatus{
				Status:  "In Progress",
				Results: TestResults{Passed: 1},
			},
			lastStatus:  "In Progress",
			lastResults: TestResults{Passed: 1},
			lastUpdate:  now.Add(-31 * time.Second),
			expected:    true,
			reason:      "periodic update",
		},
		{
			name: "no update needed",
			status: &TestStatus{
				Status:  "In Progress",
				Results: TestResults{Passed: 1},
			},
			lastStatus:  "In Progress",
			lastResults: TestResults{Passed: 1},
			lastUpdate:  now,
			expected:    false,
			reason:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shouldPrint, reason := client.shouldPrintStatus(tt.status, tt.lastStatus, tt.lastResults, tt.lastUpdate)
			assert.Equal(t, tt.expected, shouldPrint)
			assert.Equal(t, tt.reason, reason)
		})
	}
}

func TestWaitForTestCompletion(t *testing.T) {
	tests := []struct {
		name      string
		responses []struct {
			status int
			body   string
		}
		expectedError string
		shouldError   bool
	}{
		{
			name: "successful completion",
			responses: []struct {
				status int
				body   string
			}{
				{status: 227, body: `{"status": "new", "overallResults": {"In queue": 1}}`},
				{status: 228, body: `{"status": "in_progress", "overallResults": {"In progress": 1}}`},
				{status: 200, body: `{"status": "completed", "overallResults": {"Passed": 1}}`},
			},
			shouldError: false,
		},
		{
			name: "test failure",
			responses: []struct {
				status int
				body   string
			}{
				{status: 227, body: `{"status": "new", "overallResults": {"In queue": 1}}`},
				{status: 228, body: `{"status": "in_progress", "overallResults": {"In progress": 1}}`},
				{status: 230, body: `{"status": "failed", "overallResults": {"Failed": 1}}`},
			},
			expectedError: "test run failed",
			shouldError:   true,
		},
		{
			name: "API error during polling",
			responses: []struct {
				status int
				body   string
			}{
				{status: 227, body: `{"status": "new", "overallResults": {"In queue": 1}}`},
				{status: 500, body: `{"message": "Internal Server Error"}`},
			},
			expectedError: "API error (status 500): Internal Server Error",
			shouldError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock HTTP client
			mockClient := new(MockHTTPClient)

			// Set up mock responses in sequence
			for _, resp := range tt.responses {
				mockResponse := &http.Response{
					StatusCode: resp.status,
					Body:       &mockReadCloser{data: []byte(resp.body)},
				}
				// Use Once() to ensure each response is used exactly once
				mockClient.On("Do", mock.Anything).Return(mockResponse, nil).Once()
			}

			// Create test client with mock HTTP client
			client := NewTestRigorClient(&config.Config{
				TestRigor: config.TestRigorConfig{
					AuthToken: "test-token",
					AppID:     "test-app",
					APIURL:    "https://api.testrigor.com/api/v1",
				},
			}, mockClient)

			// Execute test with a shorter poll interval for testing
			err := client.WaitForTestCompletion("test-branch", []string{"test-label"}, 1, true)

			if tt.shouldError {
				assert.Error(t, err)
				if err != nil {
					assert.Contains(t, err.Error(), tt.expectedError)
				}
			} else {
				assert.NoError(t, err)
			}

			// Verify that all expectations were met
			mockClient.AssertExpectations(t)
		})
	}
}
