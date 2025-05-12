package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/benvon/testrigor-ci-tool/internal/config"
	"github.com/spf13/cobra"
)

var (
	statusBranchName string
	statusLabels     string

	statusCmd = &cobra.Command{
		Use:   "status",
		Short: "Check the status of a test suite run",
		Long:  `Check the current status of a test suite run.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadConfig()
			if err != nil {
				return err
			}

			// Build URL with query parameters
			url := fmt.Sprintf("%s/apps/%s/status", cfg.TestRigor.APIURL, cfg.TestRigor.AppID)
			if statusBranchName != "" {
				url += fmt.Sprintf("?branchName=%s", statusBranchName)
			}
			if statusLabels != "" {
				if statusBranchName != "" {
					url += "&"
				} else {
					url += "?"
				}
				url += fmt.Sprintf("labels=%s", statusLabels)
			}

			// Create request
			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				return fmt.Errorf("error creating request: %v", err)
			}

			// Set headers
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Accept", "application/json")
			req.Header.Set("auth-token", cfg.TestRigor.AuthToken)

			// Send request
			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				return fmt.Errorf("error sending request: %v", err)
			}
			defer resp.Body.Close()

			// Read response body
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("error reading response body: %v", err)
			}

			// Parse response
			var result map[string]interface{}
			if err := json.Unmarshal(body, &result); err != nil {
				return fmt.Errorf("error parsing response: %v", err)
			}

			// Print status
			fmt.Printf("Status: %v\n", result["status"])
			if details, ok := result["detailsUrl"].(string); ok {
				fmt.Printf("Details URL: %s\n", details)
			}

			return nil
		},
	}
)

func init() {
	statusCmd.Flags().StringVar(&statusBranchName, "branch", "", "Branch name to check status for")
	statusCmd.Flags().StringVar(&statusLabels, "labels", "", "Comma-separated list of labels to check status for")
}
