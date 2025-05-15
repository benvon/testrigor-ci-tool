package cmd

import (
	"fmt"
	"strings"

	"github.com/benvon/testrigor-ci-tool/internal/api"
	"github.com/benvon/testrigor-ci-tool/internal/config"
	"github.com/spf13/cobra"
)

var (
	runBranchName string
	runLabels     string
	runTimeout    int

	runCmd = &cobra.Command{
		Use:   "run",
		Short: "Run a test suite",
		Long:  `Run a test suite and wait for completion.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadConfig()
			if err != nil {
				return err
			}

			client := api.NewTestRigorClient(cfg)
			labels := []string{}
			if runLabels != "" {
				labels = strings.Split(runLabels, ",")
			}
			opts := api.TestRunOptions{
				BranchName: runBranchName,
				Labels:     labels,
			}
			result, err := client.StartTestRun(opts)
			if err != nil {
				return err
			}

			fmt.Printf("Started test run with ID: %s\n", result.TaskID)
			fmt.Println("Waiting for completion...")

			pollInterval := runTimeout // Use runTimeout as poll interval for simplicity
			err = client.WaitForTestCompletion(result.BranchName, labels, pollInterval, false)
			if err != nil {
				return err
			}

			fmt.Println("Test run completed.")
			return nil
		},
	}
)

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().StringVar(&runBranchName, "branch", "", "Branch name to run tests for")
	runCmd.Flags().StringVar(&runLabels, "labels", "", "Comma-separated list of labels to filter by")
	runCmd.Flags().IntVar(&runTimeout, "timeout", 300, "Timeout in seconds to wait for test completion")
}
