// SPDX-FileCopyrightText: 2026 Logan Lindquist Land
// SPDX-License-Identifier: FSL-1.1-MIT

package github

import (
	"context"
	"os/exec"
	"time"
)

// DefaultTimeout is the default timeout for command execution.
const DefaultTimeout = 60 * time.Second

// CommandExecutor is an interface for running shell commands.
// This abstraction enables mocking in tests without hitting real external commands.
type CommandExecutor interface {
	Execute(name string, args ...string) ([]byte, error)
}

// RealExecutor implements CommandExecutor using os/exec.
type RealExecutor struct {
	Timeout time.Duration
}

// Execute runs the command and returns its combined output.
// Commands are executed with a timeout to prevent indefinite blocking.
func (r *RealExecutor) Execute(name string, args ...string) ([]byte, error) {
	timeout := r.Timeout
	if timeout == 0 {
		timeout = DefaultTimeout
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return exec.CommandContext(ctx, name, args...).Output()
}
