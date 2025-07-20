package client

import (
	"bytes"
	"io"
	"net/http"
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

func TestNewDefaultHTTPClient(t *testing.T) {
	client := NewDefaultHTTPClient()
	assert.NotNil(t, client)
	assert.NotNil(t, client.client)
	assert.Equal(t, 30*time.Second, client.client.Timeout)
}

func TestDefaultHTTPClient_Do(t *testing.T) {
	client := NewDefaultHTTPClient()

	// Create a simple request
	req, err := http.NewRequest("GET", "http://example.com", nil)
	assert.NoError(t, err)

	// This will fail in tests since we can't make real HTTP requests
	// but we can test that the method exists and doesn't panic
	assert.NotPanics(t, func() {
		_, _ = client.Do(req)
	})
}

func TestMakeRequest(t *testing.T) {
	cfg := &config.Config{
		TestRigor: config.TestRigorConfig{
			AuthToken: "test-token",
			AppID:     "test-app",
			APIURL:    "https://api.testrigor.com/api/v1",
		},
	}

	tests := []struct {
		name        string
		opts        RequestOptions
		mockResp    *http.Response
		mockErr     error
		expectError bool
		errorMsg    string
	}{
		{
			name: "successful request",
			opts: RequestOptions{
				Method:      "GET",
				URL:         "https://api.testrigor.com/test",
				ContentType: "application/json",
				DebugMode:   false,
			},
			mockResp: &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewReader([]byte(`{"status": "success"}`))),
			},
			mockErr:     nil,
			expectError: false,
		},
		{
			name: "request with body",
			opts: RequestOptions{
				Method:      "POST",
				URL:         "https://api.testrigor.com/test",
				Body:        map[string]string{"key": "value"},
				ContentType: "application/json",
				DebugMode:   false,
			},
			mockResp: &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewReader([]byte(`{"status": "success"}`))),
			},
			mockErr:     nil,
			expectError: false,
		},
		{
			name: "HTTP error response",
			opts: RequestOptions{
				Method:      "GET",
				URL:         "https://api.testrigor.com/test",
				ContentType: "application/json",
				DebugMode:   false,
			},
			mockResp: &http.Response{
				StatusCode: 404,
				Body:       io.NopCloser(bytes.NewReader([]byte(`{"error": "Not found"}`))),
			},
			mockErr:     nil,
			expectError: true,
			errorMsg:    "API request failed with status 404",
		},
		{
			name: "network error",
			opts: RequestOptions{
				Method:      "GET",
				URL:         "https://api.testrigor.com/test",
				ContentType: "application/json",
				DebugMode:   false,
			},
			mockResp:    nil,
			mockErr:     assert.AnError,
			expectError: true,
			errorMsg:    "error making request",
		},
		{
			name: "invalid JSON body",
			opts: RequestOptions{
				Method:      "POST",
				URL:         "https://api.testrigor.com/test",
				Body:        make(chan int), // Invalid JSON type
				ContentType: "application/json",
				DebugMode:   false,
			},
			mockResp:    nil,
			mockErr:     nil,
			expectError: true,
			errorMsg:    "error marshaling request body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockHTTPClient{}

			// Only set up mock expectations if we expect the request to actually be made
			if !strings.Contains(tt.errorMsg, "marshaling") {
				if tt.mockResp != nil {
					mockClient.On("Do", mock.AnythingOfType("*http.Request")).Return(tt.mockResp, tt.mockErr)
				} else {
					mockClient.On("Do", mock.AnythingOfType("*http.Request")).Return((*http.Response)(nil), tt.mockErr)
				}
			}

			result, err := MakeRequest(mockClient, cfg, tt.opts)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}

			// Only assert expectations if we set them up
			if !strings.Contains(tt.errorMsg, "marshaling") {
				mockClient.AssertExpectations(t)
			}
		})
	}
}

func TestMakeRequest_Headers(t *testing.T) {
	cfg := &config.Config{
		TestRigor: config.TestRigorConfig{
			AuthToken: "test-token",
			AppID:     "test-app",
			APIURL:    "https://api.testrigor.com/api/v1",
		},
	}

	mockClient := &MockHTTPClient{}
	mockResp := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewReader([]byte(`{"status": "success"}`))),
	}
	mockClient.On("Do", mock.AnythingOfType("*http.Request")).Return(mockResp, nil)

	opts := RequestOptions{
		Method:      "GET",
		URL:         "https://api.testrigor.com/test",
		ContentType: "application/json",
		DebugMode:   false,
	}

	_, err := MakeRequest(mockClient, cfg, opts)
	assert.NoError(t, err)

	// Verify that the request was made with the correct headers
	call := mockClient.Calls[0]
	req := call.Arguments[0].(*http.Request)

	assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
	assert.Equal(t, "application/json", req.Header.Get("Accept"))
	assert.Equal(t, "test-token", req.Header.Get("auth-token"))
}

func TestRequestOptions_Validation(t *testing.T) {
	// Test that RequestOptions struct can be created with all fields
	opts := RequestOptions{
		Method:      "POST",
		URL:         "https://example.com",
		Body:        map[string]string{"key": "value"},
		ContentType: "application/json",
		DebugMode:   true,
	}

	assert.Equal(t, "POST", opts.Method)
	assert.Equal(t, "https://example.com", opts.URL)
	assert.Equal(t, map[string]string{"key": "value"}, opts.Body)
	assert.Equal(t, "application/json", opts.ContentType)
	assert.True(t, opts.DebugMode)
}
