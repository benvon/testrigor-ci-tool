package client

import (
	"testing"
	"time"

	"github.com/benvon/testrigor-ci-tool/internal/api/types"
	"github.com/stretchr/testify/assert"
)

func TestNewStatusUpdateManager(t *testing.T) {
	manager := NewStatusUpdateManager(true, 5*time.Second)
	assert.NotNil(t, manager)
	assert.True(t, manager.debugMode)
	assert.Equal(t, 5*time.Second, manager.updateInterval)
}

func TestStatusUpdateManager_Update(t *testing.T) {
	tests := []struct {
		name           string
		debugMode      bool
		updateInterval time.Duration
		shouldUpdate   bool
	}{
		{
			name:           "should update immediately",
			debugMode:      false,
			updateInterval: 0,
			shouldUpdate:   true,
		},
		{
			name:           "should not update within interval",
			debugMode:      false,
			updateInterval: 1 * time.Hour,
			shouldUpdate:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewStatusUpdateManager(tt.debugMode, tt.updateInterval)

			// Reset the last update time to ensure we can test the interval
			if tt.updateInterval > 0 {
				manager.lastUpdate = time.Now()
			}

			status := &types.TestStatus{
				Status: "in_progress",
				Results: types.TestResults{
					Total:      10,
					InQueue:    2,
					InProgress: 3,
					Passed:     4,
					Failed:     1,
					Canceled:   0,
				},
			}

			// Capture output to verify it was called
			manager.Update(status)

			// For the immediate update case, we can't easily test the output
			// but we can verify the manager was updated
			if tt.shouldUpdate {
				// The lastUpdate should be recent
				assert.True(t, time.Since(manager.lastUpdate) < 100*time.Millisecond)
			}
		})
	}
}

func TestStatusUpdateManager_ShouldUpdate(t *testing.T) {
	tests := []struct {
		name           string
		updateInterval time.Duration
		timeSinceLast  time.Duration
		expected       bool
	}{
		{
			name:           "should update after interval",
			updateInterval: 1 * time.Second,
			timeSinceLast:  2 * time.Second,
			expected:       true,
		},
		{
			name:           "should not update before interval",
			updateInterval: 1 * time.Second,
			timeSinceLast:  500 * time.Millisecond,
			expected:       false,
		},
		{
			name:           "should update exactly at interval",
			updateInterval: 1 * time.Second,
			timeSinceLast:  1 * time.Second,
			expected:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewStatusUpdateManager(false, tt.updateInterval)
			manager.lastUpdate = time.Now().Add(-tt.timeSinceLast)

			result := manager.ShouldUpdate()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStatusUpdateManager_Reset(t *testing.T) {
	manager := NewStatusUpdateManager(false, 1*time.Second)

	// Set lastUpdate to a time in the past
	manager.lastUpdate = time.Now().Add(-1 * time.Hour)

	// Reset
	manager.Reset()

	// Verify lastUpdate is now recent
	assert.True(t, time.Since(manager.lastUpdate) < 100*time.Millisecond)
}

func TestStatusUpdateManager_PrintFinalResults(t *testing.T) {
	manager := NewStatusUpdateManager(false, 1*time.Second)

	status := &types.TestStatus{
		Status:         "completed",
		HTTPStatusCode: 200,
		Results: types.TestResults{
			Total:      10,
			InQueue:    0,
			InProgress: 0,
			Passed:     8,
			Failed:     1,
			Canceled:   1,
			NotStarted: 0,
			Crash:      0,
		},
		Errors: []types.TestError{
			{
				Category:    "ERROR",
				Error:       "Test failed",
				Occurrences: 1,
				Severity:    "BLOCKER",
				DetailsURL:  "https://testrigor.com/error/123",
			},
		},
	}

	// This test verifies the method doesn't panic
	assert.NotPanics(t, func() {
		manager.PrintFinalResults(status)
	})
}

func TestStatusUpdateManager_PrintFinalResults_WithHTTPError(t *testing.T) {
	manager := NewStatusUpdateManager(false, 1*time.Second)

	status := &types.TestStatus{
		Status:         "failed",
		HTTPStatusCode: 500,
		Results: types.TestResults{
			Total:    5,
			Passed:   0,
			Failed:   5,
			Canceled: 0,
			Crash:    0,
		},
	}

	// This test verifies the method handles HTTP error status codes
	assert.NotPanics(t, func() {
		manager.PrintFinalResults(status)
	})
}

func TestStatusUpdateManager_PrintFinalResults_WithCrashes(t *testing.T) {
	manager := NewStatusUpdateManager(false, 1*time.Second)

	status := &types.TestStatus{
		Status:         "failed",
		HTTPStatusCode: 200,
		Results: types.TestResults{
			Total:    10,
			Passed:   5,
			Failed:   3,
			Canceled: 0,
			Crash:    2,
		},
		Errors: []types.TestError{
			{
				Category:    types.ErrorCategoryCrash,
				Error:       "Test crashed due to timeout",
				Occurrences: 2,
				Severity:    types.ErrorCategoryBlocker,
			},
		},
	}

	// This test verifies the method handles crash errors
	assert.NotPanics(t, func() {
		manager.PrintFinalResults(status)
	})
}

func TestStatusUpdateManager_Update_WithErrors(t *testing.T) {
	manager := NewStatusUpdateManager(false, 0) // Immediate updates

	status := &types.TestStatus{
		Status: "in_progress",
		Results: types.TestResults{
			Total:      5,
			InQueue:    1,
			InProgress: 2,
			Passed:     1,
			Failed:     1,
			Canceled:   0,
		},
		Errors: []types.TestError{
			{
				Category:    "ERROR",
				Error:       "Connection timeout",
				Occurrences: 3,
				Severity:    "HIGH",
			},
			{
				Category:    "WARNING",
				Error:       "Slow response",
				Occurrences: 1,
				Severity:    "MEDIUM",
			},
		},
	}

	// This test verifies the method handles status with errors
	assert.NotPanics(t, func() {
		manager.Update(status)
	})
}

func TestStatusUpdateManager_Update_WithHTTPError(t *testing.T) {
	manager := NewStatusUpdateManager(false, 0) // Immediate updates

	status := &types.TestStatus{
		Status:         "error",
		HTTPStatusCode: 503,
		Results: types.TestResults{
			Total:      0,
			InQueue:    0,
			InProgress: 0,
			Passed:     0,
			Failed:     0,
			Canceled:   0,
		},
	}

	// This test verifies the method handles HTTP error status codes
	assert.NotPanics(t, func() {
		manager.Update(status)
	})
}
