package client

import (
	"fmt"
	"time"

	"github.com/benvon/testrigor-ci-tool/internal/api/types"
)

// StatusUpdateManager handles status updates and display
type StatusUpdateManager struct {
	debugMode      bool
	lastUpdate     time.Time
	updateInterval time.Duration
}

// NewStatusUpdateManager creates a new status update manager
func NewStatusUpdateManager(debugMode bool, updateInterval time.Duration) *StatusUpdateManager {
	return &StatusUpdateManager{
		debugMode:      debugMode,
		lastUpdate:     time.Now(),
		updateInterval: updateInterval,
	}
}

// Update updates the status display if enough time has passed
func (m *StatusUpdateManager) Update(status *types.TestStatus) {
	now := time.Now()
	if now.Sub(m.lastUpdate) < m.updateInterval {
		return
	}

	m.lastUpdate = now
	m.printStatus(status)
}

// printStatus prints the current status
func (m *StatusUpdateManager) printStatus(status *types.TestStatus) {
	fmt.Printf("\nStatus: %s", status.Status)
	if status.HTTPStatusCode != 0 && (status.HTTPStatusCode < 200 || status.HTTPStatusCode > 299) {
		fmt.Printf(" (HTTP %d)", status.HTTPStatusCode)
	}
	fmt.Printf(" | Total: %d | Queue: %d | Running: %d | Passed: %d | Failed: %d | Canceled: %d\n",
		status.Results.Total,
		status.Results.InQueue,
		status.Results.InProgress,
		status.Results.Passed,
		status.Results.Failed,
		status.Results.Canceled,
	)

	if len(status.Errors) > 0 {
		fmt.Printf("Errors:\n")
		for _, err := range status.Errors {
			fmt.Printf("  - %s: %s (Severity: %s, Occurrences: %d)\n",
				err.Category, err.Error, err.Severity, err.Occurrences)
		}
	}
}

// PrintFinalResults prints the final test results
func (m *StatusUpdateManager) PrintFinalResults(status *types.TestStatus) {
	fmt.Printf("\nTest run completed with status: %s", status.Status)
	if status.HTTPStatusCode != 0 && (status.HTTPStatusCode < 200 || status.HTTPStatusCode > 299) {
		fmt.Printf(" (HTTP %d)", status.HTTPStatusCode)
	}
	fmt.Printf("\nFinal results: Total: %d | Passed: %d | Failed: %d | Canceled: %d | Crash: %d\n",
		status.Results.Total,
		status.Results.Passed,
		status.Results.Failed,
		status.Results.Canceled,
		status.Results.Crash,
	)

	if len(status.Errors) > 0 {
		fmt.Printf("\nErrors encountered:\n")
		for _, err := range status.Errors {
			fmt.Printf("  - %s: %s (Severity: %s, Occurrences: %d)\n",
				err.Category, err.Error, err.Severity, err.Occurrences)
		}
	}
}
