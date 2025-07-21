package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	Version string
	Commit  string
	Date    string

	rootCmd = &cobra.Command{
		Use:   "testrigor",
		Short: "A CLI tool for managing TestRigor test suite runs",
		Long: `A command line utility for managing TestRigor test suite runs.
It supports configuration through environment variables, command line flags, and a config file.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// If version flag is set, print version and exit
			if cmd.Flag("version").Changed {
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Version: %s\n", Version); err != nil {
					return err
				}
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Commit: %s\n", Commit); err != nil {
					return err
				}
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Build Date: %s\n", Date); err != nil {
					return err
				}
				return nil
			}
			return cmd.Help()
		},
	}
)

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.testrigor.yaml)")
	rootCmd.Flags().Bool("version", false, "Print version information and exit")

	// Add commands
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(cancelCmd)
	rootCmd.AddCommand(runAndWaitCmd)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".testrigor" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".testrigor")
	}

	// Read in environment variables that match
	viper.AutomaticEnv()

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
