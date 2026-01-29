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
