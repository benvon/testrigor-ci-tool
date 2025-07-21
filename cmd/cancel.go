package cmd

import (
	"context"
	"fmt"

	"github.com/benvon/testrigor-ci-tool/internal/api/client"
	"github.com/benvon/testrigor-ci-tool/internal/config"
	"github.com/spf13/cobra"
)

const (
	runIDFlag = "run-id"
)

var (
	cancelCmd = &cobra.Command{
		Use:   "cancel",
		Short: "Cancel a running test",
		Long:  `Cancel a currently running test suite by its run ID.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			// Load configuration
			cfg, err := config.LoadConfig()
			if err != nil {
				return fmt.Errorf("failed to load configuration: %w", err)
			}

			// Extract flags
			runID, _ := cmd.Flags().GetString(runIDFlag)

			// Validate required parameters
			if runID == "" {
				return fmt.Errorf("run ID is required")
			}

			// Create API client
			httpClient := client.NewDefaultHTTPClient()
			apiClient := client.NewTestRigorClient(cfg, httpClient)

			// Cancel the test run
			fmt.Printf("Canceling test run with ID: %s\n", runID)

			err = apiClient.CancelTestRun(ctx, runID)
			if err != nil {
				return fmt.Errorf("failed to cancel test run: %w", err)
			}

			fmt.Printf("Test run %s has been canceled successfully.\n", runID)
			return nil
		},
	}
)

func init() {
	cancelCmd.Flags().String(runIDFlag, "", "ID of the run to cancel (required)")

	// Mark run-id as required
	if err := cancelCmd.MarkFlagRequired(runIDFlag); err != nil {
		panic(err)
	}
}
