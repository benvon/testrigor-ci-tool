// Package client provides HTTP client primitives for making API requests.
// This package contains only simple, single-purpose functions for HTTP operations.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

// privateIPBlocks contains CIDR ranges for private and reserved IPs that must not
// be reachable to prevent SSRF (Server-Side Request Forgery) attacks.
var privateIPBlocks []*net.IPNet

func init() {
	for _, cidr := range []string{
		"0.0.0.0/8",       // Current network
		"10.0.0.0/8",      // Private
		"100.64.0.0/10",   // Shared address space (CGNAT)
		"127.0.0.0/8",     // Loopback
		"169.254.0.0/16",  // Link-local (cloud metadata, etc.)
		"172.16.0.0/12",   // Private
		"192.168.0.0/16",  // Private
		"224.0.0.0/4",     // Multicast
		"240.0.0.0/4",     // Reserved
		"::1/128",         // IPv6 loopback
		"fe80::/10",       // IPv6 link-local
		"fc00::/7",        // IPv6 unique local
	} {
		_, block, _ := net.ParseCIDR(cidr)
		privateIPBlocks = append(privateIPBlocks, block)
	}
}

// isPrivateOrReservedIP returns true if the IP is in a private or reserved range.
func isPrivateOrReservedIP(ip net.IP) bool {
	for _, block := range privateIPBlocks {
		if block.Contains(ip) {
			return true
		}
	}
	return false
}

// safeDialContext resolves the host and blocks connections to private/reserved IPs
// to prevent SSRF when the request URL comes from config (e.g., TESTRIGOR_API_URL).
func safeDialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}

	addrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}

	for _, ipAddr := range addrs {
		if isPrivateOrReservedIP(ipAddr.IP) {
			return nil, fmt.Errorf("connection to private/reserved IP %s is not allowed (SSRF protection)", ipAddr.IP)
		}
	}

	dialer := &net.Dialer{Timeout: 30 * time.Second}
	var lastErr error
	for _, ipAddr := range addrs {
		conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(ipAddr.IP.String(), port))
		if err == nil {
			return conn, nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("no addresses to connect to for %s", addr)
}

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
// The client uses a transport that blocks connections to private/reserved IPs to prevent SSRF.
func NewDefaultHTTPClient() *DefaultHTTPClient {
	return &DefaultHTTPClient{
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				DialContext: safeDialContext,
			},
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
