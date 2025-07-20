package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig_WithValidEnvironment(t *testing.T) {
	// Set up environment variables
	os.Setenv("TESTRIGOR_AUTH_TOKEN", "test-token")
	os.Setenv("TESTRIGOR_APP_ID", "test-app")
	os.Setenv("TESTRIGOR_API_URL", "https://custom.api.testrigor.com/api/v1")
	os.Setenv("TR_CI_ERROR_ON_TEST_FAILURE", "true")

	defer func() {
		os.Unsetenv("TESTRIGOR_AUTH_TOKEN")
		os.Unsetenv("TESTRIGOR_APP_ID")
		os.Unsetenv("TESTRIGOR_API_URL")
		os.Unsetenv("TR_CI_ERROR_ON_TEST_FAILURE")
	}()

	config, err := LoadConfig()
	assert.NoError(t, err)
	assert.NotNil(t, config)

	assert.Equal(t, "test-token", config.TestRigor.AuthToken)
	assert.Equal(t, "test-app", config.TestRigor.AppID)
	assert.Equal(t, "https://custom.api.testrigor.com/api/v1", config.TestRigor.APIURL)
	assert.True(t, config.TestRigor.ErrorOnTestFailure)
}

func TestLoadConfig_WithDefaults(t *testing.T) {
	// Set only required environment variables
	os.Setenv("TESTRIGOR_AUTH_TOKEN", "test-token")
	os.Setenv("TESTRIGOR_APP_ID", "test-app")

	defer func() {
		os.Unsetenv("TESTRIGOR_AUTH_TOKEN")
		os.Unsetenv("TESTRIGOR_APP_ID")
	}()

	config, err := LoadConfig()
	assert.NoError(t, err)
	assert.NotNil(t, config)

	assert.Equal(t, "test-token", config.TestRigor.AuthToken)
	assert.Equal(t, "test-app", config.TestRigor.AppID)
	assert.Equal(t, "https://api.testrigor.com/api/v1", config.TestRigor.APIURL) // Default value
	assert.False(t, config.TestRigor.ErrorOnTestFailure)                         // Default value
}

func TestLoadConfig_MissingAuthToken(t *testing.T) {
	// Set only AppID
	os.Setenv("TESTRIGOR_APP_ID", "test-app")

	defer func() {
		os.Unsetenv("TESTRIGOR_APP_ID")
	}()

	config, err := LoadConfig()
	assert.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "auth token is required")
}

func TestLoadConfig_MissingAppID(t *testing.T) {
	// Set only AuthToken
	os.Setenv("TESTRIGOR_AUTH_TOKEN", "test-token")

	defer func() {
		os.Unsetenv("TESTRIGOR_AUTH_TOKEN")
	}()

	config, err := LoadConfig()
	assert.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "app ID is required")
}

func TestLoadConfig_MissingBoth(t *testing.T) {
	// Clear all environment variables
	os.Unsetenv("TESTRIGOR_AUTH_TOKEN")
	os.Unsetenv("TESTRIGOR_APP_ID")

	config, err := LoadConfig()
	assert.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "auth token is required")
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config",
			config: &Config{
				TestRigor: TestRigorConfig{
					AuthToken: "test-token",
					AppID:     "test-app",
				},
			},
			expectError: false,
		},
		{
			name: "missing auth token",
			config: &Config{
				TestRigor: TestRigorConfig{
					AuthToken: "",
					AppID:     "test-app",
				},
			},
			expectError: true,
			errorMsg:    "auth token is required",
		},
		{
			name: "missing app ID",
			config: &Config{
				TestRigor: TestRigorConfig{
					AuthToken: "test-token",
					AppID:     "",
				},
			},
			expectError: true,
			errorMsg:    "app ID is required",
		},
		{
			name: "missing both",
			config: &Config{
				TestRigor: TestRigorConfig{
					AuthToken: "",
					AppID:     "",
				},
			},
			expectError: true,
			errorMsg:    "auth token is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.validate()
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfig_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		expected bool
	}{
		{
			name: "valid config",
			config: &Config{
				TestRigor: TestRigorConfig{
					AuthToken: "test-token",
					AppID:     "test-app",
				},
			},
			expected: true,
		},
		{
			name: "missing auth token",
			config: &Config{
				TestRigor: TestRigorConfig{
					AuthToken: "",
					AppID:     "test-app",
				},
			},
			expected: false,
		},
		{
			name: "missing app ID",
			config: &Config{
				TestRigor: TestRigorConfig{
					AuthToken: "test-token",
					AppID:     "",
				},
			},
			expected: false,
		},
		{
			name: "missing both",
			config: &Config{
				TestRigor: TestRigorConfig{
					AuthToken: "",
					AppID:     "",
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.IsValid()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetConfigPath(t *testing.T) {
	path := GetConfigPath()
	assert.NotEmpty(t, path)
	assert.Contains(t, path, ".testrigor.yaml")
}

func TestTestRigorConfig_Fields(t *testing.T) {
	config := TestRigorConfig{
		AuthToken:          "test-token",
		AppID:              "test-app",
		APIURL:             "https://api.testrigor.com/api/v1",
		ErrorOnTestFailure: true,
	}

	assert.Equal(t, "test-token", config.AuthToken)
	assert.Equal(t, "test-app", config.AppID)
	assert.Equal(t, "https://api.testrigor.com/api/v1", config.APIURL)
	assert.True(t, config.ErrorOnTestFailure)
}

func TestConfig_Fields(t *testing.T) {
	config := &Config{
		TestRigor: TestRigorConfig{
			AuthToken:          "test-token",
			AppID:              "test-app",
			APIURL:             "https://api.testrigor.com/api/v1",
			ErrorOnTestFailure: false,
		},
	}

	assert.NotNil(t, config.TestRigor)
	assert.Equal(t, "test-token", config.TestRigor.AuthToken)
	assert.Equal(t, "test-app", config.TestRigor.AppID)
	assert.Equal(t, "https://api.testrigor.com/api/v1", config.TestRigor.APIURL)
	assert.False(t, config.TestRigor.ErrorOnTestFailure)
}
