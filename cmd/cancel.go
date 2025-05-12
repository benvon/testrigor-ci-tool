package cmd

import (
	"fmt"
	"net/http"

	"github.com/benvon/testrigor-ci-tool/internal/config"
	"github.com/spf13/cobra"
)

var (
	runID string

	cancelCmd = &cobra.Command{
		Use:   "cancel",
		Short: "Cancel a running test suite",
		Long:  `Cancel a currently running test suite by its run ID.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if runID == "" {
				return fmt.Errorf("run ID is required")
			}

			cfg, err := config.LoadConfig()
			if err != nil {
				return err
			}

			// Create request
			req, err := http.NewRequest(
				"PUT",
				fmt.Sprintf("%s/apps/%s/runs/%s/cancel", cfg.TestRigor.APIURL, cfg.TestRigor.AppID, runID),
				nil,
			)
			if err != nil {
				return fmt.Errorf("error creating request: %v", err)
			}

			// Set headers
			req.Header.Set("auth-token", cfg.TestRigor.AuthToken)

			// Send request
			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				return fmt.Errorf("error sending request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
			}

			fmt.Println("Test suite run cancelled successfully")
			return nil
		},
	}
)

func init() {
	cancelCmd.Flags().StringVar(&runID, "run-id", "", "ID of the run to cancel")
	cancelCmd.MarkFlagRequired("run-id")
} 