package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/benvon/testrigor-ci-tool/internal/api/client"
	"github.com/benvon/testrigor-ci-tool/internal/api/types"
	"github.com/benvon/testrigor-ci-tool/internal/config"
	"github.com/spf13/cobra"
)

var (
	statusCmd = &cobra.Command{
		Use:   "status",
		Short: "Check the current status of a test suite run",
		Long: `Check the current status of a test suite run by branch name.
This command allows you to check the status without starting a new test run.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			// Load configuration
			cfg, err := config.LoadConfig()
			if err != nil {
				return fmt.Errorf("failed to load configuration: %w", err)
			}

			// Extract flags
			branchName, _ := cmd.Flags().GetString("branch")
			labelsStr, _ := cmd.Flags().GetString("labels")

			// Validate required parameters
			if branchName == "" {
				return fmt.Errorf("branch name is required")
			}

			// Parse labels
			var labels []string
			if labelsStr != "" {
				labels = strings.Split(labelsStr, ",")
				// Trim whitespace from labels
				for i, label := range labels {
					labels[i] = strings.TrimSpace(label)
				}
			}

			// Create API client
			httpClient := client.NewDefaultHTTPClient()
			apiClient := client.NewTestRigorClient(cfg, httpClient)

			// Get test status
			status, err := apiClient.GetTestStatus(ctx, branchName, labels)
			if err != nil {
				return fmt.Errorf("failed to get test status: %w", err)
			}

			// Print status information
			printTestStatus(status, branchName, labels)

			return nil
		},
	}
)

// printTestStatus prints the test status information in a formatted way.
func printTestStatus(status *types.TestStatus, branchName string, labels []string) {
	fmt.Printf("Test Status for Branch: %s\n", branchName)
	if len(labels) > 0 {
		fmt.Printf("Labels: %s\n", strings.Join(labels, ", "))
	}
	fmt.Println(strings.Repeat("-", 50))

	fmt.Printf("Status: %s\n", status.Status)

	if status.HTTPStatusCode != 0 {
		fmt.Printf("HTTP Status Code: %d\n", status.HTTPStatusCode)
	}

	if status.TaskID != "" {
		fmt.Printf("Task ID: %s\n", status.TaskID)
	}

	if status.DetailsURL != "" {
		fmt.Printf("Details URL: %s\n", status.DetailsURL)
	}

	// Print test results
	fmt.Printf("\nTest Results:\n")
	fmt.Printf("  Total: %d\n", status.Results.Total)
	fmt.Printf("  Passed: %d\n", status.Results.Passed)
	fmt.Printf("  Failed: %d\n", status.Results.Failed)
	fmt.Printf("  In Progress: %d\n", status.Results.InProgress)
	fmt.Printf("  In Queue: %d\n", status.Results.InQueue)
	fmt.Printf("  Not Started: %d\n", status.Results.NotStarted)
	fmt.Printf("  Canceled: %d\n", status.Results.Canceled)
	fmt.Printf("  Crash: %d\n", status.Results.Crash)

	// Print errors if any
	if len(status.Errors) > 0 {
		fmt.Printf("\nErrors:\n")
		for i, err := range status.Errors {
			fmt.Printf("  Error %d:\n", i+1)
			fmt.Printf("    Category: %s\n", err.Category)
			fmt.Printf("    Message: %s\n", err.Error)
			fmt.Printf("    Severity: %s\n", err.Severity)
			fmt.Printf("    Occurrences: %d\n", err.Occurrences)
			if err.DetailsURL != "" {
				fmt.Printf("    Details URL: %s\n", err.DetailsURL)
			}
			fmt.Println()
		}
	}

	// Print completion status
	if status.IsComplete() {
		fmt.Printf("\nTest run is complete.\n")
	} else if status.IsInProgress() {
		fmt.Printf("\nTest run is still in progress.\n")
	}
}

func init() {
	statusCmd.Flags().String("branch", "", "Branch name to check status for (required)")
	statusCmd.Flags().String("labels", "", "Comma-separated list of labels to filter by")

	// Mark branch as required
	if err := statusCmd.MarkFlagRequired("branch"); err != nil {
		panic(err)
	}
}
