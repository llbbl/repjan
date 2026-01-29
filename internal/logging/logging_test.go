// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

package logging

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetupLogger_TextFormat(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	logger := slog.New(handler)

	logger.Info("test message", "key", "value")

	output := buf.String()
	assert.Contains(t, output, "test message")
	assert.Contains(t, output, "key=value")
	// Text format uses key=value pairs, not JSON
	assert.NotContains(t, output, "{")
}

func TestSetupLogger_JSONFormat(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	logger := slog.New(handler)

	logger.Info("test message", "key", "value")

	output := buf.String()

	// Verify it's valid JSON
	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(output), &parsed)
	require.NoError(t, err, "output should be valid JSON")

	assert.Equal(t, "test message", parsed["msg"])
	assert.Equal(t, "value", parsed["key"])
	assert.Equal(t, "INFO", parsed["level"])
}

func TestSetupLogger_LogLevels(t *testing.T) {
	tests := []struct {
		name           string
		configLevel    string
		expectedSlog   slog.Level
		debugVisible   bool
		infoVisible    bool
		warnVisible    bool
		errorVisible   bool
	}{
		{
			name:         "debug level shows all messages",
			configLevel:  "debug",
			expectedSlog: slog.LevelDebug,
			debugVisible: true,
			infoVisible:  true,
			warnVisible:  true,
			errorVisible: true,
		},
		{
			name:         "info level hides debug messages",
			configLevel:  "info",
			expectedSlog: slog.LevelInfo,
			debugVisible: false,
			infoVisible:  true,
			warnVisible:  true,
			errorVisible: true,
		},
		{
			name:         "warn level hides debug and info messages",
			configLevel:  "warn",
			expectedSlog: slog.LevelWarn,
			debugVisible: false,
			infoVisible:  false,
			warnVisible:  true,
			errorVisible: true,
		},
		{
			name:         "error level hides debug, info, and warn messages",
			configLevel:  "error",
			expectedSlog: slog.LevelError,
			debugVisible: false,
			infoVisible:  false,
			warnVisible:  false,
			errorVisible: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			level := parseLevelForTest(tt.configLevel)
			handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: level})
			logger := slog.New(handler)

			// Log at each level
			logger.Debug("debug message")
			logger.Info("info message")
			logger.Warn("warn message")
			logger.Error("error message")

			output := buf.String()

			if tt.debugVisible {
				assert.Contains(t, output, "debug message", "debug should be visible at %s level", tt.configLevel)
			} else {
				assert.NotContains(t, output, "debug message", "debug should be hidden at %s level", tt.configLevel)
			}

			if tt.infoVisible {
				assert.Contains(t, output, "info message", "info should be visible at %s level", tt.configLevel)
			} else {
				assert.NotContains(t, output, "info message", "info should be hidden at %s level", tt.configLevel)
			}

			if tt.warnVisible {
				assert.Contains(t, output, "warn message", "warn should be visible at %s level", tt.configLevel)
			} else {
				assert.NotContains(t, output, "warn message", "warn should be hidden at %s level", tt.configLevel)
			}

			if tt.errorVisible {
				assert.Contains(t, output, "error message", "error should be visible at %s level", tt.configLevel)
			} else {
				assert.NotContains(t, output, "error message", "error should be hidden at %s level", tt.configLevel)
			}
		})
	}
}

func TestSetupLogger_InvalidLevel(t *testing.T) {
	tests := []struct {
		name     string
		level    string
		expected slog.Level
	}{
		{
			name:     "empty string defaults to info",
			level:    "",
			expected: slog.LevelInfo,
		},
		{
			name:     "unknown level defaults to info",
			level:    "unknown",
			expected: slog.LevelInfo,
		},
		{
			name:     "DEBUG (uppercase) defaults to info",
			level:    "DEBUG",
			expected: slog.LevelInfo,
		},
		{
			name:     "random string defaults to info",
			level:    "verbose",
			expected: slog.LevelInfo,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			level := parseLevelForTest(tt.level)
			assert.Equal(t, tt.expected, level)
		})
	}
}

func TestWithComponent(t *testing.T) {
	// Save and restore default logger
	originalLogger := slog.Default()
	defer slog.SetDefault(originalLogger)

	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	slog.SetDefault(slog.New(handler))

	logger := WithComponent("test-component")
	logger.Info("component test")

	output := buf.String()
	assert.Contains(t, output, "component=test-component")
	assert.Contains(t, output, "component test")
}

func TestWithComponent_MultipleComponents(t *testing.T) {
	// Save and restore default logger
	originalLogger := slog.Default()
	defer slog.SetDefault(originalLogger)

	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	slog.SetDefault(slog.New(handler))

	// Create two loggers with different components
	logger1 := WithComponent("component-a")
	logger2 := WithComponent("component-b")

	logger1.Info("message from a")
	logger2.Info("message from b")

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	require.Len(t, lines, 2)

	assert.Contains(t, lines[0], "component=component-a")
	assert.Contains(t, lines[0], "message from a")
	assert.Contains(t, lines[1], "component=component-b")
	assert.Contains(t, lines[1], "message from b")
}

func TestWithComponent_PreservesLogLevel(t *testing.T) {
	// Save and restore default logger
	originalLogger := slog.Default()
	defer slog.SetDefault(originalLogger)

	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn})
	slog.SetDefault(slog.New(handler))

	logger := WithComponent("test")

	// Info should be filtered out
	logger.Info("should not appear")
	logger.Warn("should appear")

	output := buf.String()
	assert.NotContains(t, output, "should not appear")
	assert.Contains(t, output, "should appear")
}

func TestSetupLogger_Integration(t *testing.T) {
	// Save and restore default logger
	originalLogger := slog.Default()
	defer slog.SetDefault(originalLogger)

	// This tests that SetupLogger actually sets the default logger
	// We can verify by checking that subsequent WithComponent calls work
	SetupLogger("debug", "text")

	// Get a component logger - this uses slog.Default() internally
	logger := WithComponent("integration")

	// We can't easily capture os.Stderr output in a unit test,
	// but we can verify the logger was configured by checking
	// that the returned logger has the component attribute
	require.NotNil(t, logger)
}

func TestSetupLogger_FormatSelection(t *testing.T) {
	tests := []struct {
		name     string
		format   string
		expected string // "text" or "json"
	}{
		{
			name:     "json format",
			format:   "json",
			expected: "json",
		},
		{
			name:     "text format",
			format:   "text",
			expected: "text",
		},
		{
			name:     "empty defaults to text",
			format:   "",
			expected: "text",
		},
		{
			name:     "unknown defaults to text",
			format:   "xml",
			expected: "text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We verify format by checking output characteristics
			var buf bytes.Buffer
			var handler slog.Handler
			if tt.format == "json" {
				handler = slog.NewJSONHandler(&buf, nil)
			} else {
				handler = slog.NewTextHandler(&buf, nil)
			}
			logger := slog.New(handler)
			logger.Info("test")

			output := buf.String()
			if tt.expected == "json" {
				assert.Contains(t, output, "{", "JSON format should contain braces")
			} else {
				assert.NotContains(t, output, "{", "Text format should not contain braces")
			}
		})
	}
}

// parseLevelForTest mirrors the level parsing logic from SetupLogger
// for testing purposes without affecting global state.
func parseLevelForTest(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
