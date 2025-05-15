package cmd

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// resetCommand resets the command state for testing
func resetCommand() {
	rootCmd = &cobra.Command{
		Use:   "testrigor",
		Short: "A CLI tool for managing TestRigor test suite runs",
		Long: `A command line utility for managing TestRigor test suite runs.
It supports configuration through environment variables, command line flags, and a config file.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// If version flag is set, print version and exit
			if cmd.Flag("version").Changed {
				fmt.Fprintf(cmd.OutOrStdout(), "Version: %s\n", Version)
				fmt.Fprintf(cmd.OutOrStdout(), "Commit: %s\n", Commit)
				fmt.Fprintf(cmd.OutOrStdout(), "Build Date: %s\n", Date)
				return nil
			}
			return cmd.Help()
		},
	}
	rootCmd.Flags().Bool("version", false, "Print version information and exit")

	// Add subcommands
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(cancelCmd)
	rootCmd.AddCommand(runAndWaitCmd)
}

func TestVersionFlag(t *testing.T) {
	// Reset command state
	resetCommand()

	// Set test version information
	Version = "1.2.3"
	Commit = "abc123"
	Date = "2024-03-20"

	// Create a buffer to capture output
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"--version"})

	err := rootCmd.Execute()
	assert.NoError(t, err)

	// Check output contains version information
	output := buf.String()
	assert.Contains(t, output, "Version: 1.2.3")
	assert.Contains(t, output, "Commit: abc123")
	assert.Contains(t, output, "Build Date: 2024-03-20")
}

func TestHelpOutput(t *testing.T) {
	// Reset command state
	resetCommand()

	// Create a buffer to capture output
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{})

	err := rootCmd.Execute()
	assert.NoError(t, err)

	// Check output contains help information
	output := buf.String()
	assert.Contains(t, output, "Usage:")
	assert.Contains(t, output, "Available Commands:")
}
