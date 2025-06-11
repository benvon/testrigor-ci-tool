package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/benvon/testrigor-ci-tool/internal/api/client"
	"github.com/benvon/testrigor-ci-tool/internal/api/types"
	"github.com/benvon/testrigor-ci-tool/internal/api/utils"
	"github.com/benvon/testrigor-ci-tool/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockHTTPClient is a mock implementation of the HTTPClient interface
type MockHTTPClient struct {
	mock.Mock
	statuses []*types.TestStatus
	errors   []error
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	args := m.Called(req)

	// Handle function-based response
	if fn, ok := args.Get(0).(func(*http.Request) (*http.Response, error)); ok {
		return fn(req)
	}

	// Handle direct response
	resp := args.Get(0)
	if resp == nil {
		return nil, args.Error(1)
	}
	return resp.(*http.Response), args.Error(1)
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
		opts          types.TestRunOptions
		mockResponse  interface{}
		expectedError bool
		checkResult   func(*testing.T, *types.TestRunResult)
	}{
		{
			name: "successful test run with test case UUIDs",
			opts: types.TestRunOptions{
				TestCaseUUIDs: []string{"test-uuid-1"},
				URL:           "https://example.com",
			},
			mockResponse: map[string]interface{}{
				"taskId":     "task-123",
				"branchName": "test-branch",
			},
			expectedError: false,
			checkResult: func(t *testing.T, result *types.TestRunResult) {
				assert.Equal(t, "task-123", result.TaskID)
				assert.Equal(t, "", result.BranchName)
			},
		},
		{
			name: "error response",
			opts: types.TestRunOptions{
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
	hash := utils.GenerateFakeCommitHash(timestamp)

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

	assert.Equal(t, "value", utils.GetString(m, "string"))
	assert.Equal(t, "", utils.GetString(m, "int"))
	assert.Equal(t, "", utils.GetString(m, "nil"))
	assert.Equal(t, "", utils.GetString(m, "nonexistent"))
}

func TestGetInt(t *testing.T) {
	data := map[string]interface{}{
		"int":    123,
		"float":  123.45,
		"string": "123",
		"nil":    nil,
	}

	// Test integer value
	assert.Equal(t, 123, utils.GetInt(data, "int"))

	// Test float value
	assert.Equal(t, 123, utils.GetInt(data, "float"))

	// Test string value
	assert.Equal(t, 0, utils.GetInt(data, "string"))

	// Test nil value
	assert.Equal(t, 0, utils.GetInt(data, "nil"))

	// Test non-existent key
	assert.Equal(t, 0, utils.GetInt(data, "nonexistent"))
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
			name:           "API error with message",
			responseBody:   `{"status": 404, "message": "Test run not found"}`,
			responseStatus: 404,
			expectedError:  "API error (status 404): Test run not found",
			shouldError:    true,
		},
		{
			name:           "API error with message and details",
			responseBody:   `{"status": 400, "message": "Invalid request", "errors": ["Invalid test case UUID"]}`,
			responseStatus: 400,
			expectedError:  "API error (status 400): Invalid request, errors: Invalid test case UUID",
			shouldError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock HTTP client
			mockClient := new(MockHTTPClient)

			// Prepare mock response
			mockResponse := &http.Response{
				StatusCode: tt.responseStatus,
				Body:       &mockReadCloser{data: []byte(tt.responseBody)},
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
			_, err := client.makeRequest(requestOptions{
				method:      "GET",
				url:         "https://api.testrigor.com/api/v1/test",
				contentType: "application/json",
				debugMode:   false,
			})

			if tt.shouldError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}

			// Verify that all expectations were met
			mockClient.AssertExpectations(t)
		})
	}
}

func TestPrintTestStatus(t *testing.T) {
	status := &types.TestStatus{
		Status: "In Progress",
		Results: types.TestResults{
			Total:      10,
			Passed:     5,
			Failed:     2,
			InProgress: 3,
			InQueue:    0,
		},
	}

	// Create a buffer to capture output
	var buf bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Create client and call printTestStatus
	client := NewTestRigorClient(&config.Config{})
	client.printTestStatus(status, "test reason")

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout
	io.Copy(&buf, r)

	output := buf.String()
	assert.Contains(t, output, "Current status: In Progress")
	assert.Contains(t, output, "Passed: 5")
	assert.Contains(t, output, "Failed: 2")
	assert.Contains(t, output, "In Progress: 3")
	assert.Contains(t, output, "In Queue: 0")
}

func TestPrintFinalResults(t *testing.T) {
	status := &types.TestStatus{
		Status:     "Completed",
		DetailsURL: "https://testrigor.com/details/123",
		TaskID:     "task-123",
		Results: types.TestResults{
			Total:      10,
			Passed:     7,
			Failed:     2,
			InProgress: 0,
			InQueue:    0,
			NotStarted: 0,
			Canceled:   1,
			Crash:      0,
		},
		Errors: []types.TestError{
			{
				Category:    "Test Error",
				Error:       "Test failed",
				Severity:    "High",
				Occurrences: 2,
				DetailsURL:  "https://testrigor.com/error/123",
			},
		},
	}

	// Create a buffer to capture output
	var buf bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Create client and call printFinalResults
	client := NewTestRigorClient(&config.Config{})
	client.printFinalResults(status)

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout
	io.Copy(&buf, r)

	output := buf.String()
	assert.Contains(t, output, "Test run completed with status: Completed")
	assert.Contains(t, output, "Details URL: https://testrigor.com/details/123")
	assert.Contains(t, output, "Total: 10")
	assert.Contains(t, output, "Passed: 7")
	assert.Contains(t, output, "Failed: 2")
	assert.Contains(t, output, "Canceled: 1")
	assert.Contains(t, output, "Category: Test Error")
	assert.Contains(t, output, "Error: Test failed")
	assert.Contains(t, output, "Severity: High")
	assert.Contains(t, output, "Occurrences: 2")
	assert.Contains(t, output, "Details URL: https://testrigor.com/error/123")
}

func TestShouldPrintStatus(t *testing.T) {
	tests := []struct {
		name           string
		status         *types.TestStatus
		lastStatus     string
		lastResults    types.TestResults
		lastUpdate     time.Time
		expected       bool
		expectedReason string
	}{
		{
			name: "status changed",
			status: &types.TestStatus{
				Status: "In Progress",
				Results: types.TestResults{
					Total:  10,
					Passed: 5,
				},
			},
			lastStatus: "New",
			lastResults: types.TestResults{
				Total:  10,
				Passed: 5,
			},
			lastUpdate:     time.Now().Add(-time.Minute),
			expected:       true,
			expectedReason: "status changed",
		},
		{
			name: "results updated",
			status: &types.TestStatus{
				Status: "In Progress",
				Results: types.TestResults{
					Total:  10,
					Passed: 6,
				},
			},
			lastStatus: "In Progress",
			lastResults: types.TestResults{
				Total:  10,
				Passed: 5,
			},
			lastUpdate:     time.Now().Add(-time.Minute),
			expected:       true,
			expectedReason: "results updated",
		},
		{
			name: "periodic update",
			status: &types.TestStatus{
				Status: "In Progress",
				Results: types.TestResults{
					Total:  10,
					Passed: 5,
				},
			},
			lastStatus: "In Progress",
			lastResults: types.TestResults{
				Total:  10,
				Passed: 5,
			},
			lastUpdate:     time.Now().Add(-31 * time.Second),
			expected:       true,
			expectedReason: "periodic update",
		},
		{
			name: "no update needed",
			status: &types.TestStatus{
				Status: "In Progress",
				Results: types.TestResults{
					Total:  10,
					Passed: 5,
				},
			},
			lastStatus: "In Progress",
			lastResults: types.TestResults{
				Total:  10,
				Passed: 5,
			},
			lastUpdate:     time.Now().Add(-10 * time.Second),
			expected:       false,
			expectedReason: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewTestRigorClient(&config.Config{})
			shouldPrint, reason := client.shouldPrintStatus(tt.status, tt.lastStatus, tt.lastResults, tt.lastUpdate)
			assert.Equal(t, tt.expected, shouldPrint)
			assert.Equal(t, tt.expectedReason, reason)
		})
	}
}

func TestWaitForTestCompletion(t *testing.T) {
	tests := []struct {
		name           string
		statuses       []map[string]interface{}
		errors         []error
		timeoutMinutes int
		wantErr        bool
		errMsg         string
	}{
		{
			name: "successful completion",
			statuses: []map[string]interface{}{
				{
					"status": "in_progress",
					"overallResults": map[string]interface{}{
						"Total":       1,
						"In queue":    0,
						"In progress": 1,
						"Failed":      0,
						"Passed":      0,
						"Not started": 0,
						"Canceled":    0,
						"Crash":       0,
					},
				},
				{
					"status": "completed",
					"overallResults": map[string]interface{}{
						"Total":       1,
						"In queue":    0,
						"In progress": 0,
						"Failed":      0,
						"Passed":      1,
						"Not started": 0,
						"Canceled":    0,
						"Crash":       0,
					},
				},
			},
			errors:         []error{nil, nil},
			timeoutMinutes: 1,
			wantErr:        false,
		},
		{
			name: "timeout",
			statuses: []map[string]interface{}{
				{
					"status": "in_progress",
					"overallResults": map[string]interface{}{
						"Total":       1,
						"In queue":    0,
						"In progress": 1,
						"Failed":      0,
						"Passed":      0,
						"Not started": 0,
						"Canceled":    0,
						"Crash":       0,
					},
				},
			},
			errors:         []error{nil},
			timeoutMinutes: 1,
			wantErr:        true,
			errMsg:         "timed out waiting for test completion after 1 minutes",
		},
		{
			name: "test failure",
			statuses: []map[string]interface{}{
				{
					"status": "failed",
					"overallResults": map[string]interface{}{
						"Total":       1,
						"In queue":    0,
						"In progress": 0,
						"Failed":      1,
						"Passed":      0,
						"Not started": 0,
						"Canceled":    0,
						"Crash":       0,
					},
				},
			},
			errors:         []error{fmt.Errorf("test failed")},
			timeoutMinutes: 1,
			wantErr:        true,
			errMsg:         "error making request: test failed",
		},
		{
			name: "test crash in results",
			statuses: []map[string]interface{}{
				{
					"status": "failed",
					"overallResults": map[string]interface{}{
						"Total":       1,
						"In queue":    0,
						"In progress": 0,
						"Failed":      0,
						"Passed":      0,
						"Not started": 0,
						"Canceled":    0,
						"Crash":       1,
					},
				},
			},
			errors:         []error{nil},
			timeoutMinutes: 1,
			wantErr:        true,
			errMsg:         "test crashed: 1 test(s) crashed",
		},
		{
			name: "test crash in error message",
			statuses: []map[string]interface{}{
				{
					"status": "failed",
					"overallResults": map[string]interface{}{
						"Total":       1,
						"In queue":    0,
						"In progress": 0,
						"Failed":      0,
						"Passed":      0,
						"Not started": 0,
						"Canceled":    0,
						"Crash":       0,
					},
					"errors": []interface{}{
						"CRASH: API call to 'https://development-eu01-kontoor.demandware.net/s/-/dw/data/v20_4/inventory_lists/wrangler/product_inventory_records/112362911:S' returned HTTP code '404' but expected code is '200'",
					},
				},
			},
			errors:         []error{nil},
			timeoutMinutes: 1,
			wantErr:        true,
			errMsg:         "test crashed: CRASH: API call to 'https://development-eu01-kontoor.demandware.net/s/-/dw/data/v20_4/inventory_lists/wrangler/product_inventory_records/112362911:S' returned HTTP code '404' but expected code is '200'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock HTTP client
			mockClient := new(MockHTTPClient)

			// Set up mock responses for GetTestStatus
			callCount := 0
			mockClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
				return req.Method == "GET" && strings.Contains(req.URL.Path, "/status")
			})).Return(func(req *http.Request) (*http.Response, error) {
				callCount++
				if callCount > len(tt.statuses) {
					// For timeout case, keep returning the last status
					if tt.wantErr && tt.errMsg == "timed out waiting for test completion after 1 minutes" {
						responseBody, _ := json.Marshal(tt.statuses[len(tt.statuses)-1])
						return &http.Response{
							StatusCode: 200,
							Body:       &mockReadCloser{data: responseBody},
						}, nil
					}
					// For successful completion, keep returning the last status
					if !tt.wantErr {
						responseBody, _ := json.Marshal(tt.statuses[len(tt.statuses)-1])
						return &http.Response{
							StatusCode: 200,
							Body:       &mockReadCloser{data: responseBody},
						}, nil
					}
					return nil, fmt.Errorf("unexpected call count: %d", callCount)
				}

				responseBody, _ := json.Marshal(tt.statuses[callCount-1])
				statusCode := 200
				if tt.errors[callCount-1] != nil {
					if strings.Contains(tt.errors[callCount-1].Error(), "test failed") {
						statusCode = 230
					} else {
						statusCode = 227
					}
				}
				return &http.Response{
					StatusCode: statusCode,
					Body:       &mockReadCloser{data: responseBody},
				}, tt.errors[callCount-1]
			})

			// Set up mock for cancel request
			mockClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
				return req.Method == "POST" && strings.Contains(req.URL.Path, "/cancel")
			})).Return(&http.Response{
				StatusCode: 200,
				Body:       &mockReadCloser{data: []byte(`{"status": "success"}`)},
			}, nil).Maybe()

			// Create test client with mock HTTP client
			client := NewTestRigorClient(&config.Config{
				TestRigor: config.TestRigorConfig{
					APIURL:    "https://api.testrigor.com",
					AppID:     "test-app",
					AuthToken: "test-token",
				},
			}, mockClient)

			// Use a much shorter poll interval for tests (1ms)
			err := client.WaitForTestCompletion("test-branch", []string{"test-label"}, 1, true, tt.timeoutMinutes)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error message containing %q, got %q", tt.errMsg, err.Error())
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			// Verify that all expectations were met
			mockClient.AssertExpectations(t)
		})
	}
}

func TestCancelTestRun(t *testing.T) {
	tests := []struct {
		name    string
		error   error
		wantErr bool
		errMsg  string
	}{
		{
			name:    "successful cancellation",
			error:   nil,
			wantErr: false,
		},
		{
			name:    "API error",
			error:   fmt.Errorf("API error: 500 Internal Server Error"),
			wantErr: true,
			errMsg:  "API error: 500 Internal Server Error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock HTTP client
			mockClient := new(MockHTTPClient)

			// Set up mock response for cancel request
			mockClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
				return req.Method == "POST" && strings.Contains(req.URL.Path, "/cancel")
			})).Return(&http.Response{
				StatusCode: 200,
				Body:       &mockReadCloser{data: []byte(`{"status": "success"}`)},
			}, tt.error).Once()

			// Create test client with mock HTTP client
			client := NewTestRigorClient(&config.Config{
				TestRigor: config.TestRigorConfig{
					APIURL:    "https://api.testrigor.com",
					AppID:     "test-app",
					AuthToken: "test-token",
				},
			}, mockClient)

			err := client.CancelTestRun("test-branch", []string{"test-label"})
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error message containing %q, got %q", tt.errMsg, err.Error())
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			// Verify that all expectations were met
			mockClient.AssertExpectations(t)
		})
	}
}

func TestDefaultHTTPClient_Do(t *testing.T) {
	client := client.NewDefaultHTTPClient()
	assert.NotNil(t, client)

	// Test with a non-existent URL to verify timeout
	req, _ := http.NewRequest("GET", "http://nonexistent.example.com", nil)
	_, err := client.Do(req)
	assert.Error(t, err)
}

func TestPrintDebug(t *testing.T) {
	// Create mock HTTP client
	mockClient := new(MockHTTPClient)

	// Prepare mock response
	responseBody := `{"status": "success"}`
	mockResponse := &http.Response{
		StatusCode: 200,
		Status:     "200 OK",
		Body:       &mockReadCloser{data: []byte(responseBody)},
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

	// Enable debug mode
	client.SetDebugMode(true)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Execute test
	_, err := client.makeRequest(requestOptions{
		method:      "GET",
		url:         "https://api.testrigor.com/api/v1/test",
		contentType: "application/json",
		debugMode:   true,
	})

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read captured output
	var buf bytes.Buffer
	io.Copy(&buf, r)

	// Verify output
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "Sending GET request to: https://api.testrigor.com/api/v1/test")
	assert.Contains(t, buf.String(), "Response status: 200 OK")
	assert.Contains(t, buf.String(), "Response body:")

	// Verify that all expectations were met
	mockClient.AssertExpectations(t)
}

func TestFormatHeaders(t *testing.T) {
	headers := http.Header{
		"Content-Type": []string{"application/json"},
		"Accept":       []string{"application/json", "text/plain"},
	}

	formatted := formatHeaders(headers)
	assert.Contains(t, formatted, "Content-Type: application/json")
	assert.Contains(t, formatted, "Accept: application/json, text/plain")
}

func TestPrepareBranchInfo(t *testing.T) {
	tests := []struct {
		name     string
		opts     types.TestRunOptions
		expected map[string]interface{}
	}{
		{
			name: "with branch name and commit hash",
			opts: types.TestRunOptions{
				BranchName: "test-branch",
				CommitHash: "abc123",
			},
			expected: map[string]interface{}{
				"branch": map[string]string{
					"name":   "test-branch",
					"commit": "abc123",
				},
			},
		},
		{
			name: "with branch name only",
			opts: types.TestRunOptions{
				BranchName: "test-branch",
			},
			expected: map[string]interface{}{
				"branch": map[string]string{
					"name":   "test-branch",
					"commit": "66616b65", // Should start with this
				},
			},
		},
		{
			name: "with commit hash only",
			opts: types.TestRunOptions{
				CommitHash: "abc123",
			},
			expected: nil,
		},
		{
			name:     "with no branch info",
			opts:     types.TestRunOptions{},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewTestRigorClient(&config.Config{})
			branchName, branchInfo := client.prepareBranchInfo(tt.opts)

			if tt.expected == nil {
				assert.Empty(t, branchName)
				assert.Nil(t, branchInfo)
			} else {
				assert.Equal(t, tt.opts.BranchName, branchName)
				assert.NotNil(t, branchInfo)
				expectedBranch := tt.expected["branch"].(map[string]string)
				actualBranch := branchInfo["branch"].(map[string]string)
				assert.Equal(t, expectedBranch["name"], actualBranch["name"])
				if tt.opts.CommitHash != "" {
					assert.Equal(t, expectedBranch["commit"], actualBranch["commit"])
				} else {
					assert.True(t, strings.HasPrefix(actualBranch["commit"], "66616b65"))
				}
			}
		})
	}
}

func TestGetJUnitReport(t *testing.T) {
	tests := []struct {
		name          string
		taskID        string
		mockResponse  string
		responseCode  int
		expectedError string
	}{
		{
			name:   "successful report download",
			taskID: "task-123",
			mockResponse: `<?xml version="1.0" encoding="UTF-8"?>
<testsuites>
  <testsuite name="Test Suite">
    <testcase name="Test Case 1" time="1.0"/>
  </testsuite>
</testsuites>`,
			responseCode:  200,
			expectedError: "",
		},
		{
			name:          "report not ready",
			taskID:        "task-123",
			mockResponse:  `{"status": 404, "message": "Report still being generated"}`,
			responseCode:  404,
			expectedError: "Report still being generated",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock HTTP client
			mockClient := new(MockHTTPClient)

			// Prepare mock response
			mockResponse := &http.Response{
				StatusCode: tt.responseCode,
				Body:       &mockReadCloser{data: []byte(tt.mockResponse)},
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
			err := client.GetJUnitReport(tt.taskID, false)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				// Verify file was created
				_, err := os.Stat("test-report.xml")
				assert.NoError(t, err)
				// Clean up
				os.Remove("test-report.xml")
			}

			// Verify that all expectations were met
			mockClient.AssertExpectations(t)
		})
	}
}

func TestWaitForJUnitReport(t *testing.T) {
	tests := []struct {
		name          string
		taskID        string
		mockResponses []string
		expectedError string
	}{
		{
			name:   "successful report download",
			taskID: "task-123",
			mockResponses: []string{
				`{"status": 404, "message": "Report still being generated"}`,
				`<?xml version="1.0" encoding="UTF-8"?>
<testsuites>
  <testsuite name="Test Suite">
    <testcase name="Test Case 1" time="1.0"/>
  </testsuite>
</testsuites>`,
			},
			expectedError: "",
		},
		{
			name:   "report never ready",
			taskID: "task-123",
			mockResponses: []string{
				`{"status": 404, "message": "Report still being generated"}`,
				`{"status": 404, "message": "Report still being generated"}`,
				`{"status": 404, "message": "Report still being generated"}`,
			},
			expectedError: "timed out waiting for JUnit report",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock HTTP client
			mockClient := new(MockHTTPClient)

			// Set up mock responses
			callCount := 0
			mockClient.On("Do", mock.Anything).Return(func(req *http.Request) (*http.Response, error) {
				callCount++
				if callCount > len(tt.mockResponses) {
					// For timeout case, keep returning "still being generated"
					if tt.expectedError == "timed out waiting for JUnit report" {
						return &http.Response{
							StatusCode: 404,
							Body:       &mockReadCloser{data: []byte(`{"status": 404, "message": "Report still being generated"}`)},
						}, nil
					}
					return nil, fmt.Errorf("unexpected call count: %d", callCount)
				}
				response := tt.mockResponses[callCount-1]
				statusCode := 200
				if strings.Contains(response, "still being generated") {
					statusCode = 404
				}
				return &http.Response{
					StatusCode: statusCode,
					Body:       &mockReadCloser{data: []byte(response)},
				}, nil
			}, nil).Maybe()

			// Create test client with mock HTTP client
			client := NewTestRigorClient(&config.Config{
				TestRigor: config.TestRigorConfig{
					AuthToken: "test-token",
					AppID:     "test-app",
					APIURL:    "https://api.testrigor.com/api/v1",
				},
			}, mockClient)

			// Execute test with a short poll interval
			err := client.WaitForJUnitReport(tt.taskID, 1, false)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				// Verify file was created
				_, err := os.Stat("test-report.xml")
				assert.NoError(t, err)
				// Clean up
				os.Remove("test-report.xml")
			}

			// Verify that all expectations were met
			mockClient.AssertExpectations(t)
		})
	}
}
