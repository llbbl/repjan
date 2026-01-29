// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

// Package config provides environment-based configuration for repjan.
package config

import (
	"fmt"
	"os"
	"slices"
	"time"

	"github.com/joho/godotenv"
)

// Config holds application configuration loaded from environment variables.
type Config struct {
	LogLevel     string        // debug, info, warn, error (default: info)
	LogFormat    string        // text, json (default: text)
	SyncInterval time.Duration // default: 5m
	DBPath       string        // default: ~/.repjan/repjan.db (empty means use default)
}

// validLogLevels contains the allowed log level values.
var validLogLevels = []string{"debug", "info", "warn", "error"}

// validLogFormats contains the allowed log format values.
var validLogFormats = []string{"text", "json"}

// Load reads configuration from environment variables, with .env file as optional override.
// The .env file is loaded if present but errors are ignored if it doesn't exist.
func Load() (*Config, error) {
	// Try to load .env file (ignore if not found)
	_ = godotenv.Load()

	// Read from env vars with defaults
	cfg := &Config{
		LogLevel:     getEnv("REPJAN_LOG_LEVEL", "info"),
		LogFormat:    getEnv("REPJAN_LOG_FORMAT", "text"),
		SyncInterval: getDurationEnv("REPJAN_SYNC_INTERVAL", 5*time.Minute),
		DBPath:       getEnv("REPJAN_DB_PATH", ""),
	}

	// Validate log level
	if !slices.Contains(validLogLevels, cfg.LogLevel) {
		return nil, fmt.Errorf("invalid REPJAN_LOG_LEVEL %q: must be one of %v", cfg.LogLevel, validLogLevels)
	}

	// Validate log format
	if !slices.Contains(validLogFormats, cfg.LogFormat) {
		return nil, fmt.Errorf("invalid REPJAN_LOG_FORMAT %q: must be one of %v", cfg.LogFormat, validLogFormats)
	}

	return cfg, nil
}

// getEnv retrieves an environment variable or returns a default value.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getDurationEnv retrieves a duration environment variable or returns a default value.
// If the value cannot be parsed as a duration, the default is returned.
func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return defaultValue
	}
	return duration
}
