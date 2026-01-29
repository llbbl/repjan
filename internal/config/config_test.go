// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_Defaults(t *testing.T) {
	// Clear any env vars that might affect the test
	os.Unsetenv("REPJAN_LOG_LEVEL")
	os.Unsetenv("REPJAN_LOG_FORMAT")
	os.Unsetenv("REPJAN_SYNC_INTERVAL")
	os.Unsetenv("REPJAN_DB_PATH")

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "info", cfg.LogLevel)
	assert.Equal(t, "text", cfg.LogFormat)
	assert.Equal(t, 5*time.Minute, cfg.SyncInterval)
	assert.Equal(t, "", cfg.DBPath)
}

func TestLoad_EnvVars(t *testing.T) {
	// Set env vars
	os.Setenv("REPJAN_LOG_LEVEL", "debug")
	os.Setenv("REPJAN_LOG_FORMAT", "json")
	os.Setenv("REPJAN_SYNC_INTERVAL", "30s")
	os.Setenv("REPJAN_DB_PATH", "/custom/path/test.db")
	defer func() {
		os.Unsetenv("REPJAN_LOG_LEVEL")
		os.Unsetenv("REPJAN_LOG_FORMAT")
		os.Unsetenv("REPJAN_SYNC_INTERVAL")
		os.Unsetenv("REPJAN_DB_PATH")
	}()

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "debug", cfg.LogLevel)
	assert.Equal(t, "json", cfg.LogFormat)
	assert.Equal(t, 30*time.Second, cfg.SyncInterval)
	assert.Equal(t, "/custom/path/test.db", cfg.DBPath)
}

func TestLoad_InvalidLogLevel(t *testing.T) {
	os.Setenv("REPJAN_LOG_LEVEL", "invalid")
	defer os.Unsetenv("REPJAN_LOG_LEVEL")

	cfg, err := Load()
	assert.Nil(t, cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid REPJAN_LOG_LEVEL")
}

func TestLoad_InvalidLogFormat(t *testing.T) {
	os.Setenv("REPJAN_LOG_FORMAT", "xml")
	defer os.Unsetenv("REPJAN_LOG_FORMAT")

	cfg, err := Load()
	assert.Nil(t, cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid REPJAN_LOG_FORMAT")
}

func TestLoad_InvalidDuration_UsesDefault(t *testing.T) {
	os.Setenv("REPJAN_SYNC_INTERVAL", "not-a-duration")
	defer os.Unsetenv("REPJAN_SYNC_INTERVAL")

	cfg, err := Load()
	require.NoError(t, err)

	// Invalid duration falls back to default
	assert.Equal(t, 5*time.Minute, cfg.SyncInterval)
}

func TestLoad_AllLogLevels(t *testing.T) {
	defer os.Unsetenv("REPJAN_LOG_LEVEL")

	for _, level := range validLogLevels {
		os.Setenv("REPJAN_LOG_LEVEL", level)
		cfg, err := Load()
		require.NoError(t, err, "log level %s should be valid", level)
		assert.Equal(t, level, cfg.LogLevel)
	}
}

func TestLoad_AllLogFormats(t *testing.T) {
	defer os.Unsetenv("REPJAN_LOG_FORMAT")

	for _, format := range validLogFormats {
		os.Setenv("REPJAN_LOG_FORMAT", format)
		cfg, err := Load()
		require.NoError(t, err, "log format %s should be valid", format)
		assert.Equal(t, format, cfg.LogFormat)
	}
}

func TestLoad_DurationVariations(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected time.Duration
	}{
		{
			name:     "seconds",
			input:    "30s",
			expected: 30 * time.Second,
		},
		{
			name:     "minutes",
			input:    "10m",
			expected: 10 * time.Minute,
		},
		{
			name:     "hours",
			input:    "2h",
			expected: 2 * time.Hour,
		},
		{
			name:     "combined duration",
			input:    "1h30m",
			expected: 1*time.Hour + 30*time.Minute,
		},
		{
			name:     "milliseconds",
			input:    "500ms",
			expected: 500 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("REPJAN_SYNC_INTERVAL", tt.input)
			defer os.Unsetenv("REPJAN_SYNC_INTERVAL")

			cfg, err := Load()
			require.NoError(t, err)
			assert.Equal(t, tt.expected, cfg.SyncInterval)
		})
	}
}

func TestLoad_EmptyEnvVars(t *testing.T) {
	// Set empty strings - should use defaults
	os.Setenv("REPJAN_LOG_LEVEL", "")
	os.Setenv("REPJAN_LOG_FORMAT", "")
	os.Setenv("REPJAN_SYNC_INTERVAL", "")
	os.Setenv("REPJAN_DB_PATH", "")
	defer func() {
		os.Unsetenv("REPJAN_LOG_LEVEL")
		os.Unsetenv("REPJAN_LOG_FORMAT")
		os.Unsetenv("REPJAN_SYNC_INTERVAL")
		os.Unsetenv("REPJAN_DB_PATH")
	}()

	cfg, err := Load()
	require.NoError(t, err)

	// Empty strings should fall back to defaults
	assert.Equal(t, "info", cfg.LogLevel)
	assert.Equal(t, "text", cfg.LogFormat)
	assert.Equal(t, 5*time.Minute, cfg.SyncInterval)
	assert.Equal(t, "", cfg.DBPath) // DBPath default is empty string
}

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name         string
		envValue     string
		defaultValue string
		expected     string
		setEnv       bool
	}{
		{
			name:         "returns env value when set",
			envValue:     "custom-value",
			defaultValue: "default",
			expected:     "custom-value",
			setEnv:       true,
		},
		{
			name:         "returns default when not set",
			envValue:     "",
			defaultValue: "default",
			expected:     "default",
			setEnv:       false,
		},
		{
			name:         "returns default when empty",
			envValue:     "",
			defaultValue: "default",
			expected:     "default",
			setEnv:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := "TEST_GET_ENV_KEY"
			if tt.setEnv {
				os.Setenv(key, tt.envValue)
			} else {
				os.Unsetenv(key)
			}
			defer os.Unsetenv(key)

			result := getEnv(key, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetDurationEnv(t *testing.T) {
	tests := []struct {
		name         string
		envValue     string
		defaultValue time.Duration
		expected     time.Duration
		setEnv       bool
	}{
		{
			name:         "returns parsed duration when valid",
			envValue:     "1h",
			defaultValue: 5 * time.Minute,
			expected:     1 * time.Hour,
			setEnv:       true,
		},
		{
			name:         "returns default when not set",
			envValue:     "",
			defaultValue: 5 * time.Minute,
			expected:     5 * time.Minute,
			setEnv:       false,
		},
		{
			name:         "returns default when empty",
			envValue:     "",
			defaultValue: 5 * time.Minute,
			expected:     5 * time.Minute,
			setEnv:       true,
		},
		{
			name:         "returns default when invalid",
			envValue:     "invalid",
			defaultValue: 5 * time.Minute,
			expected:     5 * time.Minute,
			setEnv:       true,
		},
		{
			name:         "returns default for numeric without unit",
			envValue:     "30",
			defaultValue: 5 * time.Minute,
			expected:     5 * time.Minute,
			setEnv:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := "TEST_GET_DURATION_KEY"
			if tt.setEnv {
				os.Setenv(key, tt.envValue)
			} else {
				os.Unsetenv(key)
			}
			defer os.Unsetenv(key)

			result := getDurationEnv(key, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLoad_DBPathWithSpaces(t *testing.T) {
	os.Setenv("REPJAN_DB_PATH", "/path/with spaces/test.db")
	defer os.Unsetenv("REPJAN_DB_PATH")

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "/path/with spaces/test.db", cfg.DBPath)
}

func TestLoad_DBPathWithSpecialChars(t *testing.T) {
	os.Setenv("REPJAN_DB_PATH", "/path/with-dashes_and_underscores/test.db")
	defer os.Unsetenv("REPJAN_DB_PATH")

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "/path/with-dashes_and_underscores/test.db", cfg.DBPath)
}
