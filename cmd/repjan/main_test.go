// SPDX-FileCopyrightText: 2026 Logan Lindquist Land
// SPDX-License-Identifier: FSL-1.1-MIT

package main

import "testing"

// TestMainPackage ensures the main package compiles and can be tested.
// The actual main() function launches the TUI which is tested in internal/cmd.
func TestMainPackage(t *testing.T) {
	// This test exists to ensure the package has test coverage.
	// The main() function itself delegates to cmd.Execute() which is tested
	// in the internal/cmd package.
}
