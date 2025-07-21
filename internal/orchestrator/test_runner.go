// Package orchestrator provides high-level orchestration services that coordinate
// multiple primitive components to implement complex business workflows.
package orchestrator

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/benvon/testrigor-ci-tool/internal/api/client"
	"github.com/benvon/testrigor-ci-tool/internal/api/types"
	"github.com/benvon/testrigor-ci-tool/internal/config"
)

// TestRigorClient interface defines the operations needed for test execution.
type TestRigorClient interface {
	StartTestRun(ctx context.Context, opts types.TestRunOptions) (*types.TestRunResult, error)
	GetTestStatus(ctx context.Context, branchName string, labels []string) (*types.TestStatus, error)
	GetJUnitReport(ctx context.Context, taskID string) ([]byte, error)
}

// TestRunner orchestrates the complete test execution workflow.
// It coordinates primitives to start tests, monitor status, and handle reports.
type TestRunner struct {
	apiClient TestRigorClient
	config    *config.Config
	logger    Logger
}

// Logger interface for outputting information during test execution.
type Logger interface {
	Printf(format string, args ...interface{})
	Println(args ...interface{})
}

// DefaultLogger implements Logger using standard fmt functions.
type DefaultLogger struct{}

// Printf implements Logger interface.
func (d DefaultLogger) Printf(format string, args ...interface{}) {
	fmt.Printf(format, args...)
}

// Println implements Logger interface.
func (d DefaultLogger) Println(args ...interface{}) {
	fmt.Println(args...)
}

// TestRunConfig contains configuration for a test run execution.
type TestRunConfig struct {
	Options       types.TestRunOptions
	PollInterval  time.Duration
	Timeout       time.Duration
	FetchReport   bool
	DebugMode     bool
}

// TestRunResult contains the complete result of a test run execution.
type TestRunResult struct {
	TaskID       string
	BranchName   string
	Status       *types.TestStatus
	Duration     time.Duration
	ReportPath   string
	Success      bool
}

// NewTestRunner creates a new test runner orchestrator.
func NewTestRunner(cfg *config.Config, httpClient client.HTTPClient, logger Logger) *TestRunner {
	if logger == nil {
		logger = DefaultLogger{}
	}

	return &TestRunner{
		apiClient: client.NewTestRigorClient(cfg, httpClient),
		config:    cfg,
		logger:    logger,
	}
}

// ExecuteTestRun orchestrates the complete test execution workflow.
// This is the main orchestrator function that coordinates multiple primitives.
func (tr *TestRunner) ExecuteTestRun(ctx context.Context, runConfig TestRunConfig) (*TestRunResult, error) {
	startTime := time.Now()
	
	tr.logRunParameters(runConfig)

	// Step 1: Start the test run
	tr.logger.Println("Starting test run...")
	result, err := tr.apiClient.StartTestRun(ctx, runConfig.Options)
	if err != nil {
		return nil, fmt.Errorf("failed to start test run: %w", err)
	}

	tr.logger.Printf("Test run started with task ID: %s\n", result.TaskID)
	tr.logger.Printf("Using branch name: %s for tracking\n", result.BranchName)

	// Step 2: Monitor test execution
	tr.logger.Println("Monitoring test execution...")
	finalStatus, err := tr.monitorTestExecution(ctx, result.BranchName, runConfig)
	if err != nil {
		return nil, fmt.Errorf("error during test execution: %w", err)
	}

	duration := time.Since(startTime)
	
	// Step 3: Determine success
	success := tr.isTestRunSuccessful(finalStatus)

	// Step 4: Download report if requested
	var reportPath string
	if runConfig.FetchReport {
		tr.logger.Println("Downloading JUnit report...")
		reportPath, err = tr.downloadReport(ctx, result.TaskID, runConfig.DebugMode)
		if err != nil {
			tr.logger.Printf("Warning: Failed to download report: %v\n", err)
		}
	}

	// Step 5: Print final results
	tr.printFinalResults(finalStatus, duration)

	// Return comprehensive result
	return &TestRunResult{
		TaskID:     result.TaskID,
		BranchName: result.BranchName,
		Status:     finalStatus,
		Duration:   duration,
		ReportPath: reportPath,
		Success:    success,
	}, nil
}

// monitorTestExecution monitors the test execution until completion.
func (tr *TestRunner) monitorTestExecution(ctx context.Context, branchName string, runConfig TestRunConfig) (*types.TestStatus, error) {
	pollTicker := time.NewTicker(runConfig.PollInterval)
	defer pollTicker.Stop()

	statusTicker := time.NewTicker(30 * time.Second) // Status updates every 30s
	defer statusTicker.Stop()

	timeoutTimer := time.NewTimer(runConfig.Timeout)
	defer timeoutTimer.Stop()

	var lastStatus *types.TestStatus
	consecutiveErrors := 0
	maxConsecutiveErrors := 5

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timeoutTimer.C:
			return nil, fmt.Errorf("timeout waiting for test completion after %v", runConfig.Timeout)
		case <-pollTicker.C:
			status, err := tr.apiClient.GetTestStatus(ctx, branchName, runConfig.Options.Labels)
			if err != nil {
				consecutiveErrors++
				if consecutiveErrors >= maxConsecutiveErrors {
					return nil, fmt.Errorf("too many consecutive errors: %w", err)
				}
				if runConfig.DebugMode {
					tr.logger.Printf("Status check error (attempt %d): %v\n", consecutiveErrors, err)
				}
				continue
			}

			consecutiveErrors = 0
			lastStatus = status

			// Check for crashes first (before checking completion)
			if status.HasCrashes() {
				return status, fmt.Errorf("test crashed: %d test(s) crashed", status.Results.Crash)
			}

			// Check for completion
			if status.IsComplete() {
				return status, nil
			}

		case <-statusTicker.C:
			if lastStatus != nil {
				tr.printStatusUpdate(lastStatus)
			}
		}
	}
}

// downloadReport downloads the JUnit report with retry logic.
func (tr *TestRunner) downloadReport(ctx context.Context, taskID string, debugMode bool) (string, error) {
	maxRetries := 10
	retryInterval := 30 * time.Second

	for i := 0; i < maxRetries; i++ {
		reportData, err := tr.apiClient.GetJUnitReport(ctx, taskID)
		if err != nil {
			if err.Error() == "report still being generated" {
				if debugMode {
					tr.logger.Printf("Report not ready, retrying in %v (attempt %d/%d)\n", retryInterval, i+1, maxRetries)
				}
				time.Sleep(retryInterval)
				continue
			}
			return "", err
		}

		// Save report to file
		reportPath := "test-report.xml"
		if err := os.WriteFile(reportPath, reportData, 0644); err != nil {
			return "", fmt.Errorf("failed to save report: %w", err)
		}

		// Get absolute path for display
		absPath, err := filepath.Abs(reportPath)
		if err != nil {
			absPath = reportPath
		}

		tr.logger.Printf("JUnit report downloaded successfully:\n")
		tr.logger.Printf("  Path: %s\n", reportPath)
		tr.logger.Printf("  Full path: %s\n", absPath)
		tr.logger.Printf("  Size: %d bytes\n", len(reportData))

		return reportPath, nil
	}

	return "", fmt.Errorf("report not ready after %d attempts", maxRetries)
}

// isTestRunSuccessful determines if the test run was successful.
func (tr *TestRunner) isTestRunSuccessful(status *types.TestStatus) bool {
	if status == nil {
		return false
	}

	// Success if completed with no failures or crashes
	return status.Status == types.StatusCompleted && 
		   status.Results.Failed == 0 && 
		   status.Results.Crash == 0
}

// logRunParameters logs the test run parameters.
func (tr *TestRunner) logRunParameters(runConfig TestRunConfig) {
	tr.logger.Println("Starting test run with parameters:")
	tr.logger.Printf("  Branch: %s\n", runConfig.Options.BranchName)
	
	if runConfig.Options.CommitHash != "" {
		tr.logger.Printf("  Commit: %s\n", runConfig.Options.CommitHash)
	}
	
	if runConfig.Options.URL != "" {
		tr.logger.Printf("  URL: %s\n", runConfig.Options.URL)
	}
	
	if len(runConfig.Options.Labels) > 0 {
		tr.logger.Printf("  Labels: %v\n", runConfig.Options.Labels)
	}
	
	if len(runConfig.Options.ExcludedLabels) > 0 {
		tr.logger.Printf("  Excluded Labels: %v\n", runConfig.Options.ExcludedLabels)
	}
	
	if runConfig.Options.CustomName != "" {
		tr.logger.Printf("  Custom Name: %s\n", runConfig.Options.CustomName)
	}
	
	if len(runConfig.Options.TestCaseUUIDs) > 0 {
		tr.logger.Printf("  Test Cases: %v\n", runConfig.Options.TestCaseUUIDs)
	}
	
	tr.logger.Printf("  Force Cancel Previous: %v\n", runConfig.Options.ForceCancelPreviousTesting)
	tr.logger.Println()
}

// printStatusUpdate prints a status update.
func (tr *TestRunner) printStatusUpdate(status *types.TestStatus) {
	tr.logger.Printf("[%s] Test Status: %s\n", time.Now().Format("15:04:05"), status.Status)
	
	if status.HTTPStatusCode != 0 && (status.HTTPStatusCode < 200 || status.HTTPStatusCode > 299) {
		tr.logger.Printf("  HTTP Status Code: %d\n", status.HTTPStatusCode)
	}
	
	total := status.Results.Total
	completed := status.Results.Passed + status.Results.Failed + status.Results.Canceled + status.Results.Crash
	
	var progressPercent float64
	if total > 0 {
		progressPercent = float64(completed) / float64(total) * 100
	}
	
	tr.logger.Printf("  Progress: %d/%d tests completed | Queue: %d | Running: %d | Passed: %d | Failed: %d | Canceled: %d (%.1f%% complete)\n",
		completed, total,
		status.Results.InQueue,
		status.Results.InProgress,
		status.Results.Passed,
		status.Results.Failed,
		status.Results.Canceled,
		progressPercent,
	)
}

// printFinalResults prints the final test results.
func (tr *TestRunner) printFinalResults(status *types.TestStatus, duration time.Duration) {
	tr.logger.Printf("\nTest run completed with status: %s\n", status.Status)
	tr.logger.Printf("Total duration: %s\n", duration.Round(time.Second))
	
	if status.DetailsURL != "" {
		tr.logger.Printf("Details URL: %s\n", status.DetailsURL)
	}

	tr.logger.Printf("\nFinal Results:\n")
	tr.logger.Printf("  Total: %d\n", status.Results.Total)
	tr.logger.Printf("  Passed: %d\n", status.Results.Passed)
	tr.logger.Printf("  Failed: %d\n", status.Results.Failed)
	tr.logger.Printf("  In Progress: %d\n", status.Results.InProgress)
	tr.logger.Printf("  In Queue: %d\n", status.Results.InQueue)
	tr.logger.Printf("  Not Started: %d\n", status.Results.NotStarted)
	tr.logger.Printf("  Canceled: %d\n", status.Results.Canceled)
	tr.logger.Printf("  Crash: %d\n", status.Results.Crash)

	if len(status.Errors) > 0 {
		tr.logger.Printf("\nErrors:\n")
		for _, err := range status.Errors {
			tr.logger.Printf("  Category: %s\n", err.Category)
			tr.logger.Printf("  Error: %s\n", err.Error)
			tr.logger.Printf("  Severity: %s\n", err.Severity)
			tr.logger.Printf("  Occurrences: %d\n", err.Occurrences)
			if err.DetailsURL != "" {
				tr.logger.Printf("  Details URL: %s\n", err.DetailsURL)
			}
			tr.logger.Println()
		}
	}
}