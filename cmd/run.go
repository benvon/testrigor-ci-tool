package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/benvon/testrigor-ci-tool/internal/config"
	"github.com/spf13/cobra"
)

var (
	forceCancel bool
	branchName  string
	branchCommit string
	url         string
	labels      []string
	excludedLabels []string
	customName  string

	runCmd = &cobra.Command{
		Use:   "run",
		Short: "Start a test suite run",
		Long:  `Start a new test suite run with the specified parameters.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadConfig()
			if err != nil {
				return err
			}

			// Prepare request body
			body := map[string]interface{}{
				"forceCancelPreviousTesting": forceCancel,
			}

			if branchName != "" {
				body["branch"] = map[string]string{
					"name":   branchName,
					"commit": branchCommit,
				}
			}

			if url != "" {
				body["url"] = url
			}

			if len(labels) > 0 {
				body["labels"] = labels
			}

			if len(excludedLabels) > 0 {
				body["excludedLabels"] = excludedLabels
			}

			if customName != "" {
				body["customName"] = customName
			}

			jsonBody, err := json.Marshal(body)
			if err != nil {
				return fmt.Errorf("error marshaling request body: %v", err)
			}

			// Create request
			req, err := http.NewRequest(
				"POST",
				fmt.Sprintf("%s/apps/%s/retest", cfg.TestRigor.APIURL, cfg.TestRigor.AppID),
				bytes.NewBuffer(jsonBody),
			)
			if err != nil {
				return fmt.Errorf("error creating request: %v", err)
			}

			// Set headers
			req.Header.Set("Content-Type", "application/json")
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

			fmt.Println("Test suite run started successfully")
			return nil
		},
	}
)

func init() {
	runCmd.Flags().BoolVar(&forceCancel, "force-cancel", false, "Cancel any previous running tests")
	runCmd.Flags().StringVar(&branchName, "branch", "", "Branch name to test")
	runCmd.Flags().StringVar(&branchCommit, "commit", "", "Commit hash to test")
	runCmd.Flags().StringVar(&url, "url", "", "URL where the application is deployed")
	runCmd.Flags().StringSliceVar(&labels, "labels", []string{}, "Labels to include in the test run")
	runCmd.Flags().StringSliceVar(&excludedLabels, "exclude-labels", []string{}, "Labels to exclude from the test run")
	runCmd.Flags().StringVar(&customName, "name", "", "Custom name for the test run")
} 