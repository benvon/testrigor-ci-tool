package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

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
