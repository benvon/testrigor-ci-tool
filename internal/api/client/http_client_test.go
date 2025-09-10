package client

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	apiURL              = "https://api.example.com/test"
	httpContentType     = "application/json"
	httpAuthHeaderValue = "Bearer token"
)

// MockHTTPClient is a mock implementation of the HTTPClient interface for testing.
type MockHTTPClient struct {
	mock.Mock
}

// Do implements the HTTPClient interface.
func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	args := m.Called(req)
	if resp := args.Get(0); resp != nil {
		return resp.(*http.Response), args.Error(1)
	}
	return nil, args.Error(1)
}

// mockReadCloser implements io.ReadCloser for testing.
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

func TestNewDefaultHTTPClient(t *testing.T) {
	client := NewDefaultHTTPClient()
	assert.NotNil(t, client)
	assert.NotNil(t, client.client)
}

func TestNew(t *testing.T) {
	tests := []struct {
		name       string
		httpClient HTTPClient
		expectNew  bool
	}{
		{
			name:       "with provided client",
			httpClient: &MockHTTPClient{},
			expectNew:  false,
		},
		{
			name:       "with nil client",
			httpClient: nil,
			expectNew:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := New(tt.httpClient)
			assert.NotNil(t, client)
			assert.NotNil(t, client.httpClient)
		})
	}
}

func TestClientExecute(t *testing.T) {
	tests := []struct {
		name           string
		request        Request
		mockResponse   *http.Response
		mockError      error
		expectedError  bool
		expectedStatus int
	}{
		{
			name: "successful GET request",
			request: Request{
				Method: "GET",
				URL:    apiURL,
				Headers: map[string]string{
					"Authorization": httpAuthHeaderValue,
				},
			},
			mockResponse: &http.Response{
				StatusCode: 200,
				Body:       &mockReadCloser{data: []byte(`{"success": true}`)},
				Header:     make(http.Header),
			},
			expectedError:  false,
			expectedStatus: 200,
		},
		{
			name: "successful POST request with body",
			request: Request{
				Method:      "POST",
				URL:         apiURL,
				Body:        map[string]string{"key": "value"},
				ContentType: httpContentType,
			},
			mockResponse: &http.Response{
				StatusCode: 201,
				Body:       &mockReadCloser{data: []byte(`{"created": true}`)},
				Header:     make(http.Header),
			},
			expectedError:  false,
			expectedStatus: 201,
		},
		{
			name: "HTTP client error",
			request: Request{
				Method: "GET",
				URL:    apiURL,
			},
			mockError:     errors.New("network error"),
			expectedError: true,
		},
		{
			name: "invalid JSON body",
			request: Request{
				Method: "POST",
				URL:    apiURL,
				Body:   make(chan int), // Invalid JSON type
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockHTTPClient)
			client := New(mockClient)

			// Only set up mock expectations if we expect the HTTP call to be made
			// (i.e., if the error isn't from JSON marshaling)
			if tt.name != "invalid JSON body" {
				if tt.mockError != nil {
					mockClient.On("Do", mock.AnythingOfType("*http.Request")).Return(nil, tt.mockError)
				} else {
					mockClient.On("Do", mock.AnythingOfType("*http.Request")).Return(tt.mockResponse, nil)
				}
			}

			ctx := context.Background()
			resp, err := client.Execute(ctx, tt.request)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Equal(t, tt.expectedStatus, resp.StatusCode)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestClientBuildHTTPRequest(t *testing.T) {
	tests := []struct {
		name           string
		request        Request
		expectedMethod string
		expectedURL    string
		expectedError  bool
		checkHeaders   map[string]string
	}{
		{
			name: "GET request with headers",
			request: Request{
				Method: "GET",
				URL:    apiURL,
				Headers: map[string]string{
					"Authorization": httpAuthHeaderValue,
					"Accept":        httpContentType,
				},
			},
			expectedMethod: "GET",
			expectedURL:    apiURL,
			checkHeaders: map[string]string{
				"Authorization": httpAuthHeaderValue,
				"Accept":        httpContentType,
			},
		},
		{
			name: "POST request with content type",
			request: Request{
				Method:      "POST",
				URL:         apiURL,
				ContentType: httpContentType,
				Body:        map[string]string{"test": "data"},
			},
			expectedMethod: "POST",
			expectedURL:    apiURL,
			checkHeaders: map[string]string{
				"Content-Type": httpContentType,
			},
		},
		{
			name: "invalid URL",
			request: Request{
				Method: "GET",
				URL:    "://invalid-url",
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := New(nil)
			ctx := context.Background()

			req, err := client.buildHTTPRequest(ctx, tt.request)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, req)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, req)
				assert.Equal(t, tt.expectedMethod, req.Method)
				assert.Equal(t, tt.expectedURL, req.URL.String())

				// Check headers
				for key, expectedValue := range tt.checkHeaders {
					assert.Equal(t, expectedValue, req.Header.Get(key))
				}

				// Check body for POST requests
				if tt.request.Body != nil {
					assert.NotNil(t, req.Body)
					bodyBytes, err := io.ReadAll(req.Body)
					assert.NoError(t, err)
					assert.NotEmpty(t, bodyBytes)
				}
			}
		})
	}
}

func TestDefaultHTTPClientDo(t *testing.T) {
	// Create a local test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Respond with a simple JSON response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "test response", "method": "` + r.Method + `"}`))
	}))
	defer server.Close()

	client := NewDefaultHTTPClient()

	// Create a simple GET request to our local test server
	req, err := http.NewRequest("GET", server.URL+"/test", nil)
	assert.NoError(t, err)

	// Test the HTTP client with our local server
	resp, err := client.Do(req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	defer resp.Body.Close()

	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	// Verify the response body
	body, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Contains(t, string(body), "test response")
	assert.Contains(t, string(body), "GET")
}

func TestRequestValidation(t *testing.T) {
	tests := []struct {
		name    string
		request Request
		valid   bool
	}{
		{
			name: "valid GET request",
			request: Request{
				Method: "GET",
				URL:    apiURL,
			},
			valid: true,
		},
		{
			name: "valid POST request with body",
			request: Request{
				Method:      "POST",
				URL:         apiURL,
				Body:        map[string]string{"key": "value"},
				ContentType: httpContentType,
			},
			valid: true,
		},
		{
			name: "empty method",
			request: Request{
				Method: "",
				URL:    apiURL,
			},
			valid: true, // Go's http.NewRequest allows empty method
		},
		{
			name: "empty URL",
			request: Request{
				Method: "GET",
				URL:    "",
			},
			valid: true, // Go's http.NewRequest allows empty URL
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := New(nil)
			ctx := context.Background()

			_, err := client.buildHTTPRequest(ctx, tt.request)

			if tt.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}
