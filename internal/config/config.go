package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

// Config holds all configuration for the TestRigor CI tool.
type Config struct {
	// TestRigor contains TestRigor-specific configuration
	TestRigor TestRigorConfig
}

// TestRigorConfig holds all TestRigor-specific configuration.
type TestRigorConfig struct {
	// AuthToken is the authentication token for the TestRigor API
	AuthToken string
	// AppID is the TestRigor application ID
	AppID string
	// APIURL is the base URL for the TestRigor API
	APIURL string
	// ErrorOnTestFailure determines whether to exit with error code when tests fail
	ErrorOnTestFailure bool
}

// LoadConfig loads the configuration from file, environment variables, and command line flags.
// It sets sensible defaults and validates required fields.
func LoadConfig() (*Config, error) {
	// Set defaults
	viper.SetDefault("testrigor.apiurl", "https://api.testrigor.com/api/v1")
	viper.SetDefault("testrigor.errorontestfailure", false)

	// Bind environment variables
	if err := viper.BindEnv("testrigor.authtoken", "TESTRIGOR_AUTH_TOKEN"); err != nil {
		return nil, fmt.Errorf("failed to bind auth token env var: %v", err)
	}
	if err := viper.BindEnv("testrigor.appid", "TESTRIGOR_APP_ID"); err != nil {
		return nil, fmt.Errorf("failed to bind app ID env var: %v", err)
	}
	if err := viper.BindEnv("testrigor.apiurl", "TESTRIGOR_API_URL"); err != nil {
		return nil, fmt.Errorf("failed to bind API URL env var: %v", err)
	}
	if err := viper.BindEnv("testrigor.errorontestfailure", "TR_CI_ERROR_ON_TEST_FAILURE"); err != nil {
		return nil, fmt.Errorf("failed to bind error on test failure env var: %v", err)
	}

	// Create config structure
	config := &Config{
		TestRigor: TestRigorConfig{
			AuthToken:          viper.GetString("testrigor.authtoken"),
			AppID:              viper.GetString("testrigor.appid"),
			APIURL:             viper.GetString("testrigor.apiurl"),
			ErrorOnTestFailure: viper.GetBool("testrigor.errorontestfailure"),
		},
	}

	// Validate required fields
	if err := config.validate(); err != nil {
		return nil, err
	}

	return config, nil
}

// validate validates the configuration and returns an error if invalid.
func (c *Config) validate() error {
	if c.TestRigor.AuthToken == "" {
		return fmt.Errorf("auth token is required. Set TESTRIGOR_AUTH_TOKEN environment variable or auth_token in config file")
	}
	if c.TestRigor.AppID == "" {
		return fmt.Errorf("app ID is required. Set TESTRIGOR_APP_ID environment variable or app_id in config file")
	}
	return nil
}

// GetConfigPath returns the path to the config file.
// It defaults to the user's home directory with the name .testrigor.yaml.
func GetConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".testrigor.yaml"
	}
	return fmt.Sprintf("%s/.testrigor.yaml", home)
}

// IsValid returns true if the configuration is valid.
func (c *Config) IsValid() bool {
	return c.TestRigor.AuthToken != "" && c.TestRigor.AppID != ""
}
