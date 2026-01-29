// SPDX-FileCopyrightText: 2026 Logan Lindquist Land
// SPDX-License-Identifier: FSL-1.1-MIT

package github

import "os/exec"

// CommandExecutor is an interface for running shell commands.
// This abstraction enables mocking in tests without hitting real external commands.
type CommandExecutor interface {
	Execute(name string, args ...string) ([]byte, error)
}

// RealExecutor implements CommandExecutor using os/exec.
type RealExecutor struct{}

// Execute runs the command and returns its combined output.
func (r *RealExecutor) Execute(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).Output()
}
