package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/benvon/testrigor-ci-tool/internal/api/client"
	"github.com/benvon/testrigor-ci-tool/internal/api/types"
	"github.com/benvon/testrigor-ci-tool/internal/config"
	"github.com/benvon/testrigor-ci-tool/internal/orchestrator"
	"github.com/spf13/cobra"
)

var (
	runAndWaitCmd = &cobra.Command{
		Use:   "run-and-wait",
		Short: "Start a test run and wait for completion",
		Long: `Start a test run and wait for it to complete.
The branch name parameter can be used to track test runs. It can be any meaningful identifier,
such as:
- ci-{timestamp} for CI runs
- pr-{number} for pull request runs
- manual-{description} for manual test runs

If no branch name is provided, one will be automatically generated.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			// Load configuration
			cfg, err := config.LoadConfig()
			if err != nil {
				return fmt.Errorf("failed to load configuration: %w", err)
			}

			// Extract command flags
			runConfig, err := buildTestRunConfig(cmd)
			if err != nil {
				return fmt.Errorf("failed to build run configuration: %w", err)
			}

			// Create test runner orchestrator
			httpClient := client.NewDefaultHTTPClient()
			testRunner := orchestrator.NewTestRunner(cfg, httpClient, nil)

			// Execute the test run
			result, err := testRunner.ExecuteTestRun(ctx, runConfig)
			if err != nil {
				// Check if this is a test failure vs system error
				if result != nil && !result.Success && !cfg.TestRigor.ErrorOnTestFailure {
					// Test failed but we're not configured to error on test failure
					fmt.Printf("Test run completed with failures, but continuing due to configuration.\n")
					return nil
				}
				return err
			}

			// Check final result against configuration
			if !result.Success && cfg.TestRigor.ErrorOnTestFailure {
				return fmt.Errorf("test run failed: %d failed, %d crashed",
					result.Status.Results.Failed, result.Status.Results.Crash)
			}

			return nil
		},
	}
)

// buildTestRunConfig extracts command line flags and builds the test run configuration.
func buildTestRunConfig(cmd *cobra.Command) (orchestrator.TestRunConfig, error) {
	// Extract all flags
	debugMode, _ := cmd.Flags().GetBool("debug")
	labels, _ := cmd.Flags().GetStringSlice("labels")
	excludedLabels, _ := cmd.Flags().GetStringSlice("excluded-labels")
	branchName, _ := cmd.Flags().GetString("branch")
	commitHash, _ := cmd.Flags().GetString("commit")
	url, _ := cmd.Flags().GetString("url")
	testCase, _ := cmd.Flags().GetString("test-case")
	customName, _ := cmd.Flags().GetString("name")
	pollInterval, _ := cmd.Flags().GetInt("poll-interval")
	timeoutMinutes, _ := cmd.Flags().GetInt("timeout")
	forceCancel := cmd.Flag("force-cancel").Changed
	fetchReport := cmd.Flag("fetch-report").Changed
	makeXrayReports := cmd.Flag("make-xray-reports").Changed

	// Build test run options
	opts := types.TestRunOptions{
		ForceCancelPreviousTesting: forceCancel,
		BranchName:                 branchName,
		CommitHash:                 commitHash,
		URL:                        url,
		Labels:                     labels,
		ExcludedLabels:             excludedLabels,
		CustomName:                 customName,
		MakeXrayReports:            makeXrayReports,
	}

	// Add test case UUID if provided
	if testCase != "" {
		opts.TestCaseUUIDs = []string{testCase}
	}

	// Build complete run configuration
	runConfig := orchestrator.TestRunConfig{
		Options:      opts,
		PollInterval: time.Duration(pollInterval) * time.Second,
		Timeout:      time.Duration(timeoutMinutes) * time.Minute,
		FetchReport:  fetchReport,
		DebugMode:    debugMode,
	}

	return runConfig, nil
}

func init() {
	runAndWaitCmd.Flags().StringSlice("labels", []string{}, "Labels to filter tests")
	runAndWaitCmd.Flags().StringSlice("excluded-labels", []string{}, "Labels to exclude from test run")
	runAndWaitCmd.Flags().String("branch", "", "Branch name for tracking the test run (e.g., ci-123, pr-456, manual-smoke)")
	runAndWaitCmd.Flags().String("commit", "", "Commit hash for test run")
	runAndWaitCmd.Flags().String("url", "", "URL for test run")
	runAndWaitCmd.Flags().String("test-case", "", "Test case UUID to run")
	runAndWaitCmd.Flags().String("name", "", "Custom name for test run")
	runAndWaitCmd.Flags().Int("poll-interval", 10, "Polling interval in seconds")
	runAndWaitCmd.Flags().Int("timeout", 30, "Maximum time to wait for test completion in minutes (default: 30 minutes)")
	runAndWaitCmd.Flags().Bool("debug", false, "Enable debug output")
	runAndWaitCmd.Flags().Bool("force-cancel", false, "Force cancel previous testing")
	runAndWaitCmd.Flags().Bool("fetch-report", false, "Download JUnit report after test completion")
	runAndWaitCmd.Flags().Bool("make-xray-reports", false, "Enable Xray Cloud reporting (disabled by default)")
}
