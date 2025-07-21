package cmd

import (
	"testing"

	"github.com/benvon/testrigor-ci-tool/internal/api/types"
)

func TestPrintTestStatus(t *testing.T) {
	status := &types.TestStatus{
		Status: types.StatusCompleted,
		Results: types.TestResults{
			Total: 10, Passed: 8, Failed: 2, Crash: 0,
		},
	}
	// Just check that it doesn't panic
	printTestStatus(status, "branch", []string{"smoke"})

	// With errors
	status.Errors = []types.TestError{{Error: "foo", Severity: "HIGH", Occurrences: 1}}
	printTestStatus(status, "branch", []string{"smoke"})

	// In progress
	status.Status = types.StatusInProgress
	printTestStatus(status, "branch", []string{"smoke"})
}
