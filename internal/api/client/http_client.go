// Package client provides HTTP client primitives for making API requests.
// This package contains only simple, single-purpose functions for HTTP operations.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
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

// Request represents an HTTP request with all necessary parameters.
type Request struct {
	Method      string
	URL         string
	Body        interface{}
	Headers     map[string]string
	ContentType string
}

// Response represents an HTTP response with body and metadata.
type Response struct {
	StatusCode int
	Body       []byte
	Headers    http.Header
}

// Client is a primitive HTTP client that handles only HTTP operations.
type Client struct {
	httpClient HTTPClient
}

// New creates a new HTTP client with the provided HTTPClient implementation.
func New(httpClient HTTPClient) *Client {
	if httpClient == nil {
		httpClient = NewDefaultHTTPClient()
	}
	return &Client{httpClient: httpClient}
}

// Execute performs an HTTP request and returns the response.
// This is a primitive function that only handles HTTP mechanics.
func (c *Client) Execute(ctx context.Context, req Request) (*Response, error) {
	httpReq, err := c.buildHTTPRequest(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to build HTTP request: %w", err)
	}

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute HTTP request: %w", err)
	}
	defer func() {
		_ = httpResp.Body.Close()
	}()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return &Response{
		StatusCode: httpResp.StatusCode,
		Body:       body,
		Headers:    httpResp.Header,
	}, nil
}

// buildHTTPRequest constructs an HTTP request from the Request struct.
func (c *Client) buildHTTPRequest(ctx context.Context, req Request) (*http.Request, error) {
	var bodyReader io.Reader

	if req.Body != nil {
		bodyBytes, err := json.Marshal(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	httpReq, err := http.NewRequestWithContext(ctx, req.Method, req.URL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set content type
	if req.ContentType != "" {
		httpReq.Header.Set("Content-Type", req.ContentType)
	}

	// Set custom headers
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	return httpReq, nil
}
