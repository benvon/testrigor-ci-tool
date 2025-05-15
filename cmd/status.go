package cmd

import (
	"fmt"

	"github.com/benvon/testrigor-ci-tool/internal/api"
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

			client := api.NewTestRigorClient(cfg)
			status, err := client.GetTestStatus(statusBranchName, []string{statusLabels})
			if err != nil {
				return err
			}

			fmt.Printf("Status: %s\n", status.Status)
			if status.DetailsURL != "" {
				fmt.Printf("Details URL: %s\n", status.DetailsURL)
			}

			return nil
		},
	}
)

func init() {
	rootCmd.AddCommand(statusCmd)
	statusCmd.Flags().StringVar(&statusBranchName, "branch", "", "Branch name to check status for")
	statusCmd.Flags().StringVar(&statusLabels, "labels", "", "Comma-separated list of labels to filter by")
}
