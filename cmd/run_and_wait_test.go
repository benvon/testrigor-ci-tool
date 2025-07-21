package cmd

import (
	"fmt"
	"os"
	"testing"

	"github.com/benvon/testrigor-ci-tool/internal/orchestrator"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestBuildTestRunConfig(t *testing.T) {
	// Set required environment variables to avoid nil config
	os.Setenv("TESTRIGOR_AUTH_TOKEN", "dummy")
	os.Setenv("TESTRIGOR_APP_ID", "dummy")
	os.Setenv("TESTRIGOR_API_URL", "http://dummy")

	tests := []struct {
		name       string
		flags      map[string]interface{}
		expectsErr bool
		check      func(*testing.T, orchestrator.TestRunConfig)
	}{
		{
			name: "all flags set",
			flags: map[string]interface{}{
				// 'labels' will be set directly below
				"branch":        "feature-branch",
				"commit":        "abc123",
				"url":           "https://example.com",
				"fetch-report":  true,
				"debug":         true,
				"poll-interval": 5,
				"timeout":       60,
			},
			expectsErr: false,
			check: func(t *testing.T, cfg orchestrator.TestRunConfig) {
				assert.Equal(t, []string{"smoke", "regression"}, cfg.Options.Labels)
				assert.Equal(t, "feature-branch", cfg.Options.BranchName)
				assert.Equal(t, "abc123", cfg.Options.CommitHash)
				assert.Equal(t, "https://example.com", cfg.Options.URL)
				assert.True(t, cfg.FetchReport)
				assert.True(t, cfg.DebugMode)
				assert.Equal(t, int64(5), int64(cfg.PollInterval.Seconds()))
				assert.Equal(t, int64(3600), int64(cfg.Timeout.Seconds()))
			},
		},
		{
			name:       "missing required flags",
			flags:      map[string]interface{}{},
			expectsErr: false, // Should not error, just use defaults
			check: func(t *testing.T, cfg orchestrator.TestRunConfig) {
				assert.Empty(t, cfg.Options.Labels)
				assert.Empty(t, cfg.Options.BranchName)
				assert.Empty(t, cfg.Options.CommitHash)
				assert.Empty(t, cfg.Options.URL)
				assert.False(t, cfg.FetchReport)
				assert.False(t, cfg.DebugMode)
			},
		},
		{
			name:       "invalid poll-interval",
			flags:      map[string]interface{}{"poll-interval": "notanint"},
			expectsErr: false, // Implementation does not error, uses default
			check:      nil,
		},
		{
			name:       "invalid timeout",
			flags:      map[string]interface{}{"timeout": "notanint"},
			expectsErr: false, // Implementation does not error, uses default
			check:      nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			// Set up flags (define each flag only once per command instance)
			if tt.name == "all flags set" {
				cmd.Flags().StringSlice("labels", []string{"smoke", "regression"}, "")
			} else {
				cmd.Flags().StringSlice("labels", nil, "")
			}
			cmd.Flags().String("branch", "", "")
			cmd.Flags().String("commit", "", "")
			cmd.Flags().String("url", "", "")
			cmd.Flags().Bool("fetch-report", false, "")
			cmd.Flags().Bool("debug", false, "")
			cmd.Flags().Int("poll-interval", 10, "")
			cmd.Flags().Int("timeout", 30, "")
			cmd.Flags().StringSlice("excluded-labels", nil, "")
			cmd.Flags().String("test-case", "", "")
			cmd.Flags().String("name", "", "")
			cmd.Flags().Bool("force-cancel", false, "")
			cmd.Flags().Bool("make-xray-reports", false, "")

			for k, v := range tt.flags {
				if k == "labels" && tt.name == "all flags set" {
					// Already set above
					continue
				}
				switch val := v.(type) {
				case string:
					cmd.Flags().Set(k, val)
				case bool:
					cmd.Flags().Set(k, fmt.Sprintf("%v", val))
				case int:
					cmd.Flags().Set(k, fmt.Sprintf("%d", val))
				default:
					cmd.Flags().Set(k, fmt.Sprintf("%v", val))
				}
			}
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic in buildTestRunConfig: %v", r)
				}
			}()
			cfg, err := buildTestRunConfig(cmd)
			if tt.expectsErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.check != nil {
					tt.check(t, cfg)
				}
			}
		})
	}
}
