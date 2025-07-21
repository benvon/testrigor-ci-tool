package api

import "time"

// HTTP Status Codes
const (
	HTTPStatusOK                  = 200
	HTTPStatusBadRequest          = 400
	HTTPStatusUnauthorized        = 401
	HTTPStatusNotFound            = 404
	HTTPStatusInternalServerError = 500
	HTTPStatusServiceUnavailable  = 503
)

// TestRigor specific status codes
const (
	TestRigorStatusInProgress227 = 227
	TestRigorStatusInProgress228 = 228
	TestRigorStatusFailed230     = 230
)

// Timeouts and intervals
const (
	DefaultHTTPTimeout    = 30 * time.Second
	DefaultPollInterval   = 10
	DefaultTimeoutMinutes = 30
	StatusUpdateInterval  = 5 * time.Second
	MaxConsecutiveErrors  = 5
	JUnitReportMaxRetries = 60
)

// File permissions
const (
	DefaultFilePermission = 0644
)

// Test status values
const (
	TestStatusNew        = "New"
	TestStatusInProgress = "In progress"
	TestStatusCompleted  = "completed"
	TestStatusFailed     = "Failed"
	TestStatusCanceled   = "Canceled"
)

// Error messages
const (
	ErrTestCrashed    = "test crashed:"
	ErrReportNotReady = "Report still being generated"
	ErrTestNotReady   = "status 404"
	ErrAPIError404    = "API error (status 404)"
)

// File paths
const (
	DefaultReportPath = "test-report.xml"
)
