package cmd

import (
	"fmt"
	"strings"

	"github.com/benvon/testrigor-ci-tool/internal/api"
	"github.com/benvon/testrigor-ci-tool/internal/api/types"
	"github.com/benvon/testrigor-ci-tool/internal/config"
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
			// Load configuration
			cfg, err := config.LoadConfig()
			if err != nil {
				return err
			}

			// Get flag values
			debugMode, _ := cmd.Flags().GetBool("debug")
			labels, _ := cmd.Flags().GetStringSlice("labels")
			excludedLabels, _ := cmd.Flags().GetStringSlice("excluded-labels")
			branchName, _ := cmd.Flags().GetString("branch")
			commitHash, _ := cmd.Flags().GetString("commit")
			url, _ := cmd.Flags().GetString("url")
			testCase, _ := cmd.Flags().GetString("test-case")
			customName, _ := cmd.Flags().GetString("name")
			pollInterval, _ := cmd.Flags().GetInt("poll-interval")
			forceCancel := cmd.Flag("force-cancel").Changed
			fetchReport := cmd.Flag("fetch-report").Changed
			makeXrayReports := cmd.Flag("make-xray-reports").Changed
			timeout, _ := cmd.Flags().GetInt("timeout")

			// Create API client
			client := api.NewTestRigorClient(cfg)
			client.SetDebugMode(debugMode)

			// Prepare test run options
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

			// Print run parameters
			fmt.Println("Starting test run with parameters:")
			fmt.Printf("  Branch: %s\n", branchName)
			if commitHash != "" {
				fmt.Printf("  Commit: %s\n", commitHash)
			}
			if url != "" {
				fmt.Printf("  URL: %s\n", url)
			}
			if len(labels) > 0 {
				fmt.Printf("  Labels: %s\n", strings.Join(labels, ", "))
			}
			if len(excludedLabels) > 0 {
				fmt.Printf("  Excluded Labels: %s\n", strings.Join(excludedLabels, ", "))
			}
			if customName != "" {
				fmt.Printf("  Custom Name: %s\n", customName)
			}
			if testCase != "" {
				fmt.Printf("  Test Case: %s\n", testCase)
			}
			fmt.Printf("  Force Cancel Previous: %v\n", forceCancel)
			fmt.Println()

			// Start test run
			result, err := client.StartTestRun(opts)
			if err != nil {
				return fmt.Errorf("error starting test run: %v", err)
			}

			fmt.Printf("Test run started with task ID: %s\n", result.TaskID)
			fmt.Printf("Using branch name: %s for tracking\n", result.BranchName)
			fmt.Println()

			// Wait for completion
			err = client.WaitForTestCompletion(result.BranchName, labels, pollInterval, debugMode, timeout)
			if err != nil {
				// If it's a test failure and we're not configured to error on test failure, just log it
				if strings.Contains(err.Error(), "test run failed") && !cfg.TestRigor.ErrorOnTestFailure {
					fmt.Println(err.Error())
					err = nil
				} else {
					return err
				}
			}

			// If fetch-report is enabled, wait for and download the JUnit report
			if fetchReport {
				if debugMode {
					fmt.Printf("\nFetching JUnit report...\n")
				}
				if err := client.WaitForJUnitReport(result.TaskID, pollInterval, debugMode); err != nil {
					return fmt.Errorf("error fetching JUnit report: %v", err)
				}
			}

			// Return the original error if the test run failed and we're configured to error on test failure
			if err != nil && cfg.TestRigor.ErrorOnTestFailure {
				return err
			}
			return nil
		},
	}
)

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
