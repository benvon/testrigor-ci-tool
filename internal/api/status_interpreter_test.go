package api

import (
	"encoding/json"
	"testing"

	"github.com/benvon/testrigor-ci-tool/internal/api/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStatusInterpreter(t *testing.T) {
	si := NewStatusInterpreter()
	assert.NotNil(t, si)
}

func TestStatusInterpreter_InterpretResponse(t *testing.T) {
	si := NewStatusInterpreter()

	tests := []struct {
		name       string
		statusCode int
		body       interface{}
		expectErr  bool
		errMsg     string
	}{
		{
			name:       "success response",
			statusCode: types.StatusOK,
			body:       map[string]string{"status": "success"},
			expectErr:  false,
		},
		{
			name:       "test in progress 227",
			statusCode: types.StatusTestInProgress227,
			body:       map[string]string{"status": "in_progress"},
			expectErr:  true,
			errMsg:     "status 227",
		},
		{
			name:       "test in progress 228",
			statusCode: types.StatusTestInProgress228,
			body:       map[string]string{"status": "in_progress"},
			expectErr:  true,
			errMsg:     "status 228",
		},
		{
			name:       "test failed",
			statusCode: types.StatusTestFailed,
			body:       map[string]string{"status": "failed"},
			expectErr:  true,
			errMsg:     "test failed",
		},
		{
			name:       "not found with crash error",
			statusCode: types.StatusNotFound,
			body: map[string]interface{}{
				"errors": []string{"CRASH: Test crashed due to timeout"},
			},
			expectErr: true,
			errMsg:    "test crashed: CRASH: Test crashed due to timeout",
		},
		{
			name:       "not found without crash error",
			statusCode: types.StatusNotFound,
			body: map[string]interface{}{
				"message": "Resource not found",
				"errors":  []string{"Invalid resource"},
			},
			expectErr: true,
			errMsg:    "API error (status 404): Resource not found, errors: Invalid resource",
		},
		{
			name:       "bad request",
			statusCode: types.StatusBadRequest,
			body: map[string]interface{}{
				"message": "Bad request",
				"errors":  []string{"Invalid parameter"},
			},
			expectErr: true,
			errMsg:    "API error (status 400): Bad request, errors: Invalid parameter",
		},
		{
			name:       "unauthorized",
			statusCode: types.StatusUnauthorized,
			body: map[string]interface{}{
				"message": "Unauthorized",
			},
			expectErr: true,
			errMsg:    "API error (status 401): Unauthorized, errors: ",
		},
		{
			name:       "unexpected status code",
			statusCode: 999,
			body: map[string]interface{}{
				"message": "Unknown error",
			},
			expectErr: true,
			errMsg:    "unexpected status code: 999, message: Unknown error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bodyBytes, err := json.Marshal(tt.body)
			require.NoError(t, err)

			result, err := si.InterpretResponse(tt.statusCode, bodyBytes)

			if tt.expectErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, bodyBytes, result)
			}
		})
	}
}

func TestStatusInterpreter_ParseTestStatus(t *testing.T) {
	si := NewStatusInterpreter()

	tests := []struct {
		name       string
		statusCode int
		body       map[string]interface{}
		expected   *types.TestStatus
		expectErr  bool
	}{
		{
			name:       "valid test status",
			statusCode: types.StatusOK,
			body: map[string]interface{}{
				"status":     "completed",
				"detailsUrl": "https://testrigor.com/runs/123",
				"errors": []map[string]interface{}{
					{
						"category":    "ERROR",
						"error":       "Test failed",
						"occurrences": float64(1),
						"severity":    "BLOCKER",
						"detailsUrl":  "https://testrigor.com/errors/456",
					},
				},
				"overallResults": map[string]interface{}{
					"Total":       float64(10),
					"In queue":    float64(0),
					"In progress": float64(0),
					"Failed":      float64(2),
					"Passed":      float64(8),
					"Canceled":    float64(0),
					"Not started": float64(0),
					"Crash":       float64(0),
				},
			},
			expected: &types.TestStatus{
				Status:         "completed",
				DetailsURL:     "https://testrigor.com/runs/123",
				TaskID:         "123",
				HTTPStatusCode: types.StatusOK,
				Errors: []types.TestError{
					{
						Category:    "ERROR",
						Error:       "Test failed",
						Occurrences: 1,
						Severity:    "BLOCKER",
						DetailsURL:  "https://testrigor.com/errors/456",
					},
				},
				Results: types.TestResults{
					Total:      10,
					InQueue:    0,
					InProgress: 0,
					Failed:     2,
					Passed:     8,
					Canceled:   0,
					NotStarted: 0,
					Crash:      0,
				},
			},
			expectErr: false,
		},
		{
			name:       "test status with string errors",
			statusCode: types.StatusOK,
			body: map[string]interface{}{
				"status": "failed",
				"errors": []string{"CRASH: Test crashed", "Test timeout"},
			},
			expected: &types.TestStatus{
				Status:         "failed",
				HTTPStatusCode: types.StatusOK,
				Errors: []types.TestError{
					{
						Category:    types.ErrorCategoryCrash,
						Error:       "CRASH: Test crashed",
						Occurrences: 1,
						Severity:    types.ErrorCategoryBlocker,
					},
					{
						Category:    types.ErrorCategoryCrash,
						Error:       "Test timeout",
						Occurrences: 1,
						Severity:    types.ErrorCategoryBlocker,
					},
				},
			},
			expectErr: true, // Should return error due to crash
		},
		{
			name:       "test status with crash error",
			statusCode: types.StatusOK,
			body: map[string]interface{}{
				"status": "failed",
				"errors": []map[string]interface{}{
					{
						"category": "CRASH",
						"error":    "CRASH: Test crashed due to timeout",
					},
				},
			},
			expected: &types.TestStatus{
				Status:         "failed",
				HTTPStatusCode: types.StatusOK,
				Errors: []types.TestError{
					{
						Category: "CRASH",
						Error:    "CRASH: Test crashed due to timeout",
					},
				},
			},
			expectErr: true, // Should return error due to crash
		},
		{
			name:       "invalid JSON",
			statusCode: types.StatusOK,
			body:       nil,
			expectErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var bodyBytes []byte
			var err error

			if tt.body != nil {
				bodyBytes, err = json.Marshal(tt.body)
				require.NoError(t, err)
			} else {
				bodyBytes = []byte("invalid json")
			}

			result, err := si.ParseTestStatus(tt.statusCode, bodyBytes)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected.Status, result.Status)
				assert.Equal(t, tt.expected.DetailsURL, result.DetailsURL)
				assert.Equal(t, tt.expected.TaskID, result.TaskID)
				assert.Equal(t, tt.expected.HTTPStatusCode, result.HTTPStatusCode)
				assert.Equal(t, tt.expected.Results, result.Results)
				assert.Equal(t, len(tt.expected.Errors), len(result.Errors))
			}
		})
	}
}

func TestStatusInterpreter_IsValidStatusResponse(t *testing.T) {
	si := NewStatusInterpreter()

	tests := []struct {
		name       string
		statusCode int
		expected   bool
	}{
		{"status OK", types.StatusOK, true},
		{"status 227", types.StatusTestInProgress227, true},
		{"status 228", types.StatusTestInProgress228, true},
		{"status 230", types.StatusTestFailed, true},
		{"status 400", types.StatusBadRequest, false},
		{"status 404", types.StatusNotFound, false},
		{"status 500", types.StatusInternalServerError, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := si.IsValidStatusResponse(tt.statusCode)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStatusInterpreter_HelperFunctions(t *testing.T) {
	si := NewStatusInterpreter()

	// Test getString
	t.Run("getString", func(t *testing.T) {
		m := map[string]interface{}{
			"string": "value",
			"int":    123,
			"nil":    nil,
		}

		assert.Equal(t, "value", si.getString(m, "string"))
		assert.Equal(t, "", si.getString(m, "int"))
		assert.Equal(t, "", si.getString(m, "nil"))
		assert.Equal(t, "", si.getString(m, "missing"))
	})

	// Test getInt
	t.Run("getInt", func(t *testing.T) {
		m := map[string]interface{}{
			"float64": float64(123),
			"int":     456,
			"string":  "789",
			"nil":     nil,
		}

		assert.Equal(t, 123, si.getInt(m, "float64"))
		assert.Equal(t, 456, si.getInt(m, "int"))
		assert.Equal(t, 0, si.getInt(m, "string"))
		assert.Equal(t, 0, si.getInt(m, "nil"))
		assert.Equal(t, 0, si.getInt(m, "missing"))
	})
}

func TestStatusInterpreter_ExtractErrorDetails(t *testing.T) {
	si := NewStatusInterpreter()

	tests := []struct {
		name     string
		jsonData map[string]interface{}
		expected string
	}{
		{
			name: "single error",
			jsonData: map[string]interface{}{
				"errors": []interface{}{"Error 1"},
			},
			expected: "Error 1",
		},
		{
			name: "multiple errors",
			jsonData: map[string]interface{}{
				"errors": []interface{}{"Error 1", "Error 2", "Error 3"},
			},
			expected: "Error 1; Error 2; Error 3",
		},
		{
			name: "no errors",
			jsonData: map[string]interface{}{
				"errors": []interface{}{},
			},
			expected: "",
		},
		{
			name: "no errors field",
			jsonData: map[string]interface{}{
				"message": "No errors",
			},
			expected: "",
		},
		{
			name: "mixed error types",
			jsonData: map[string]interface{}{
				"errors": []interface{}{"String error", 123, nil},
			},
			expected: "String error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := si.extractErrorDetails(tt.jsonData)
			assert.Equal(t, tt.expected, result)
		})
	}
}
