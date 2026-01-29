// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

// Package logging provides structured logging configuration using slog.
package logging

import (
	"log/slog"
	"os"
)

// SetupLogger configures the default slog logger based on level and format.
// level: "debug", "info", "warn", "error"
// format: "text", "json"
func SetupLogger(level, format string) {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: lvl}

	var handler slog.Handler
	if format == "json" {
		handler = slog.NewJSONHandler(os.Stderr, opts)
	} else {
		handler = slog.NewTextHandler(os.Stderr, opts)
	}

	slog.SetDefault(slog.New(handler))
}

// WithComponent returns a logger with the component field set.
func WithComponent(component string) *slog.Logger {
	return slog.Default().With("component", component)
}
