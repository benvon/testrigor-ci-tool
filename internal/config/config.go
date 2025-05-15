package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

// Config holds all configuration for our program
type Config struct {
	TestRigor TestRigorConfig
}

// TestRigorConfig holds all TestRigor specific configuration
type TestRigorConfig struct {
	AuthToken          string
	AppID              string
	APIURL             string
	ErrorOnTestFailure bool
	APIKey             string
}

// LoadConfig loads the configuration from file, environment variables, and command line flags
func LoadConfig() (*Config, error) {
	// Set defaults
	viper.SetDefault("testrigor.apiurl", "https://api.testrigor.com/api/v1")
	viper.SetDefault("testrigor.errorontestfailure", false)

	// Bind environment variables
	viper.BindEnv("testrigor.authtoken", "TESTRIGOR_AUTH_TOKEN")
	viper.BindEnv("testrigor.appid", "TESTRIGOR_APP_ID")
	viper.BindEnv("testrigor.apiurl", "TESTRIGOR_API_URL")
	viper.BindEnv("testrigor.errorontestfailure", "TR_CI_ERROR_ON_TEST_FAILURE")

	// Read config
	config := &Config{
		TestRigor: TestRigorConfig{
			AuthToken:          viper.GetString("testrigor.authtoken"),
			AppID:              viper.GetString("testrigor.appid"),
			APIURL:             viper.GetString("testrigor.apiurl"),
			ErrorOnTestFailure: viper.GetBool("testrigor.errorontestfailure"),
			APIKey:             viper.GetString("testrigor.apikey"),
		},
	}

	// Validate required fields
	if config.TestRigor.AuthToken == "" {
		return nil, fmt.Errorf("auth token is required. Set TESTRIGOR_AUTH_TOKEN environment variable or auth_token in config file")
	}
	if config.TestRigor.AppID == "" {
		return nil, fmt.Errorf("app ID is required. Set TESTRIGOR_APP_ID environment variable or app_id in config file")
	}

	return config, nil
}

// GetConfigPath returns the path to the config file
func GetConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".testrigor.yaml"
	}
	return fmt.Sprintf("%s/.testrigor.yaml", home)
}
