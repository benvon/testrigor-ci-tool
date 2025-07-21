package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	authTokenEnvVar           = "TESTRIGOR_AUTH_TOKEN"
	appIDEnvVar               = "TESTRIGOR_APP_ID"
	apiURLEnvVar              = "TESTRIGOR_API_URL"
	errorOnTestFailureEnvVar  = "TR_CI_ERROR_ON_TEST_FAILURE"
	authTokenDefault          = "test-token"
	appIDDefault              = "test-app"
	apiURLDefault             = "https://api.testrigor.com/api/v1"
	errorOnTestFailureDefault = false
	errorTestTokenIsRequired  = "auth token is required"
	errorTestAppIDIsRequired  = "app ID is required"
)

func TestLoadConfigWithValidEnvironment(t *testing.T) {
	// Set up environment variables
	_ = os.Setenv(authTokenEnvVar, authTokenDefault)
	_ = os.Setenv(appIDEnvVar, appIDDefault)
	_ = os.Setenv(apiURLEnvVar, apiURLDefault)
	_ = os.Setenv(errorOnTestFailureEnvVar, "true")

	defer func() {
		_ = os.Unsetenv(authTokenEnvVar)
		_ = os.Unsetenv(appIDEnvVar)
		_ = os.Unsetenv(apiURLEnvVar)
		_ = os.Unsetenv(errorOnTestFailureEnvVar)
	}()

	config, err := LoadConfig()
	assert.NoError(t, err)
	assert.NotNil(t, config)

	assert.Equal(t, authTokenDefault, config.TestRigor.AuthToken)
	assert.Equal(t, appIDDefault, config.TestRigor.AppID)
	assert.Equal(t, apiURLDefault, config.TestRigor.APIURL)
	assert.True(t, config.TestRigor.ErrorOnTestFailure)
}

func TestLoadConfigWithDefaults(t *testing.T) {
	// Set only required environment variables
	_ = os.Setenv(authTokenEnvVar, authTokenDefault)
	_ = os.Setenv(appIDEnvVar, appIDDefault)

	defer func() {
		_ = os.Unsetenv(authTokenEnvVar)
		_ = os.Unsetenv(appIDEnvVar)
	}()

	config, err := LoadConfig()
	assert.NoError(t, err)
	assert.NotNil(t, config)

	assert.Equal(t, authTokenDefault, config.TestRigor.AuthToken)
	assert.Equal(t, appIDDefault, config.TestRigor.AppID)
	assert.Equal(t, apiURLDefault, config.TestRigor.APIURL) // Default value
	assert.False(t, config.TestRigor.ErrorOnTestFailure)    // Default value
}

func TestLoadConfigMissingAuthToken(t *testing.T) {
	// Set only AppID
	_ = os.Setenv(appIDEnvVar, appIDDefault)

	defer func() {
		_ = os.Unsetenv(appIDEnvVar)
	}()

	config, err := LoadConfig()
	assert.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), errorTestTokenIsRequired)
}

func TestLoadConfigMissingAppID(t *testing.T) {
	// Set only AuthToken
	_ = os.Setenv(authTokenEnvVar, authTokenDefault)

	defer func() {
		_ = os.Unsetenv(authTokenEnvVar)
	}()

	config, err := LoadConfig()
	assert.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), errorTestAppIDIsRequired)
}

func TestLoadConfigMissingBoth(t *testing.T) {
	// Clear all environment variables
	_ = os.Unsetenv(authTokenEnvVar)
	_ = os.Unsetenv(appIDEnvVar)

	config, err := LoadConfig()
	assert.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), errorTestTokenIsRequired)
}

func TestConfigValidate(t *testing.T) {
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
					AuthToken: authTokenDefault,
					AppID:     appIDDefault,
				},
			},
			expectError: false,
		},
		{
			name: "missing auth token",
			config: &Config{
				TestRigor: TestRigorConfig{
					AuthToken: "",
					AppID:     appIDDefault,
				},
			},
			expectError: true,
			errorMsg:    errorTestTokenIsRequired,
		},
		{
			name: "missing app ID",
			config: &Config{
				TestRigor: TestRigorConfig{
					AuthToken: authTokenDefault,
					AppID:     "",
				},
			},
			expectError: true,
			errorMsg:    errorTestAppIDIsRequired,
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
			errorMsg:    errorTestTokenIsRequired,
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

func TestConfigIsValid(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		expected bool
	}{
		{
			name: "valid config",
			config: &Config{
				TestRigor: TestRigorConfig{
					AuthToken: authTokenDefault,
					AppID:     appIDDefault,
				},
			},
			expected: true,
		},
		{
			name: "missing auth token",
			config: &Config{
				TestRigor: TestRigorConfig{
					AuthToken: "",
					AppID:     appIDDefault,
				},
			},
			expected: false,
		},
		{
			name: "missing app ID",
			config: &Config{
				TestRigor: TestRigorConfig{
					AuthToken: authTokenDefault,
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

func TestTestRigorConfigFields(t *testing.T) {
	config := TestRigorConfig{
		AuthToken:          authTokenDefault,
		AppID:              appIDDefault,
		APIURL:             apiURLDefault,
		ErrorOnTestFailure: true,
	}

	assert.Equal(t, authTokenDefault, config.AuthToken)
	assert.Equal(t, appIDDefault, config.AppID)
	assert.Equal(t, apiURLDefault, config.APIURL)
	assert.True(t, config.ErrorOnTestFailure)
}

func TestConfigFields(t *testing.T) {
	config := &Config{
		TestRigor: TestRigorConfig{
			AuthToken:          authTokenDefault,
			AppID:              appIDDefault,
			APIURL:             apiURLDefault,
			ErrorOnTestFailure: false,
		},
	}

	assert.NotNil(t, config.TestRigor)
	assert.Equal(t, authTokenDefault, config.TestRigor.AuthToken)
	assert.Equal(t, appIDDefault, config.TestRigor.AppID)
	assert.Equal(t, apiURLDefault, config.TestRigor.APIURL)
	assert.False(t, config.TestRigor.ErrorOnTestFailure)
}
