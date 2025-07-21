package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/benvon/testrigor-ci-tool/internal/config"
)

// HTTPClient defines the interface for making HTTP requests.
// This allows for easy mocking in tests.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// DefaultHTTPClient is the default implementation of HTTPClient.
type DefaultHTTPClient struct {
	client *http.Client
}

// NewDefaultHTTPClient creates a new default HTTP client with a 30-second timeout.
func NewDefaultHTTPClient() *DefaultHTTPClient {
	return &DefaultHTTPClient{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Do implements the HTTPClient interface by delegating to the underlying http.Client.
func (c *DefaultHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return c.client.Do(req)
}

// RequestOptions represents the options for making an HTTP request.
type RequestOptions struct {
	// Method is the HTTP method (GET, POST, etc.)
	Method string
	// URL is the target URL for the request
	URL string
	// Body is the request body (will be JSON marshaled)
	Body interface{}
	// ContentType is the Content-Type header value
	ContentType string
	// DebugMode enables debug output
	DebugMode bool
}

// MakeRequest makes an HTTP request with the given options and returns the response body.
// It handles JSON marshaling, authentication, and basic error handling.
func MakeRequest(client HTTPClient, cfg *config.Config, opts RequestOptions) ([]byte, error) {
	var bodyReader io.Reader
	if opts.Body != nil {
		bodyBytes, err := json.Marshal(opts.Body)
		if err != nil {
			return nil, fmt.Errorf("error marshaling request body: %v", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequest(opts.Method, opts.URL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	// Set standard headers
	req.Header.Set("Content-Type", opts.ContentType)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("auth-token", cfg.TestRigor.AuthToken)

	// Debug output
	if opts.DebugMode {
		fmt.Printf("Making %s request to %s\n", opts.Method, opts.URL)
		if opts.Body != nil {
			bodyBytes, _ := json.MarshalIndent(opts.Body, "", "  ")
			fmt.Printf("Request body: %s\n", string(bodyBytes))
		}
	}

	// Make the request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %v", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			// Log the error but don't fail the request
			if opts.DebugMode {
				fmt.Printf("Warning: failed to close response body: %v\n", closeErr)
			}
		}
	}()

	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %v", err)
	}

	// Check for HTTP errors
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Debug output
	if opts.DebugMode {
		fmt.Printf("Response status: %d\n", resp.StatusCode)
		fmt.Printf("Response body: %s\n", string(bodyBytes))
	}

	return bodyBytes, nil
}
