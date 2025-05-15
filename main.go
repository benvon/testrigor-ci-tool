package main

import (
	"fmt"
	"os"

	"github.com/benvon/testrigor-ci-tool/cmd"
)

var (
	version string
	commit  string
	date    string
)

func main() {
	// Set version information in the root command
	cmd.Version = version
	cmd.Commit = commit
	cmd.Date = date

	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
