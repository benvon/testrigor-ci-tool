package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	// Save original env vars
	originalAuthToken := os.Getenv("TESTRIGOR_AUTH_TOKEN")
	originalAppID := os.Getenv("TESTRIGOR_APP_ID")
	originalAPIURL := os.Getenv("TESTRIGOR_API_URL")
	originalErrorOnFailure := os.Getenv("TR_CI_ERROR_ON_TEST_FAILURE")

	// Cleanup after test
	defer func() {
		os.Setenv("TESTRIGOR_AUTH_TOKEN", originalAuthToken)
		os.Setenv("TESTRIGOR_APP_ID", originalAppID)
		os.Setenv("TESTRIGOR_API_URL", originalAPIURL)
		os.Setenv("TR_CI_ERROR_ON_TEST_FAILURE", originalErrorOnFailure)
	}()

	tests := []struct {
		name          string
		envVars       map[string]string
		expectedError bool
		checkConfig   func(*testing.T, *Config)
	}{
		{
			name: "valid configuration",
			envVars: map[string]string{
				"TESTRIGOR_AUTH_TOKEN": "test-token",
				"TESTRIGOR_APP_ID":     "test-app",
			},
			expectedError: false,
			checkConfig: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "test-token", cfg.TestRigor.AuthToken)
				assert.Equal(t, "test-app", cfg.TestRigor.AppID)
				assert.Equal(t, "https://api.testrigor.com/api/v1", cfg.TestRigor.APIURL)
				assert.False(t, cfg.TestRigor.ErrorOnTestFailure)
			},
		},
		{
			name: "missing auth token",
			envVars: map[string]string{
				"TESTRIGOR_APP_ID": "test-app",
			},
			expectedError: true,
		},
		{
			name: "missing app ID",
			envVars: map[string]string{
				"TESTRIGOR_AUTH_TOKEN": "test-token",
			},
			expectedError: true,
		},
		{
			name: "custom API URL and error on failure",
			envVars: map[string]string{
				"TESTRIGOR_AUTH_TOKEN":        "test-token",
				"TESTRIGOR_APP_ID":            "test-app",
				"TESTRIGOR_API_URL":           "https://custom-api.testrigor.com/api/v1",
				"TR_CI_ERROR_ON_TEST_FAILURE": "true",
			},
			expectedError: false,
			checkConfig: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "test-token", cfg.TestRigor.AuthToken)
				assert.Equal(t, "test-app", cfg.TestRigor.AppID)
				assert.Equal(t, "https://custom-api.testrigor.com/api/v1", cfg.TestRigor.APIURL)
				assert.True(t, cfg.TestRigor.ErrorOnTestFailure)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment variables
			os.Unsetenv("TESTRIGOR_AUTH_TOKEN")
			os.Unsetenv("TESTRIGOR_APP_ID")
			os.Unsetenv("TESTRIGOR_API_URL")
			os.Unsetenv("TR_CI_ERROR_ON_TEST_FAILURE")

			// Set test environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			cfg, err := LoadConfig()
			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, cfg)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, cfg)
				if tt.checkConfig != nil {
					tt.checkConfig(t, cfg)
				}
			}
		})
	}
}

func TestGetConfigPath(t *testing.T) {
	path := GetConfigPath()
	assert.Contains(t, path, ".testrigor.yaml")

	// Test that it falls back to local path if home dir is not available
	originalHome := os.Getenv("HOME")
	os.Unsetenv("HOME")
	defer os.Setenv("HOME", originalHome)

	path = GetConfigPath()
	assert.Equal(t, ".testrigor.yaml", path)
}
