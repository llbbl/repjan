// SPDX-FileCopyrightText: 2026 Logan Lindquist Land
// SPDX-License-Identifier: FSL-1.1-MIT

package testutil

import "sync"

// MockExecutor implements github.CommandExecutor for testing.
// It allows configuring responses and records all calls for verification.
type MockExecutor struct {
	mu          sync.Mutex
	ExecuteFunc func(name string, args ...string) ([]byte, error)
	Calls       [][]string // Record all calls for verification
}

// NewMockExecutor creates a new MockExecutor with no default behavior.
func NewMockExecutor() *MockExecutor {
	return &MockExecutor{
		Calls: make([][]string, 0),
	}
}

// Execute runs the configured ExecuteFunc or returns nil if not set.
// All calls are recorded in the Calls slice for verification.
func (m *MockExecutor) Execute(name string, args ...string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Record the call: first element is the command name, rest are args
	call := append([]string{name}, args...)
	m.Calls = append(m.Calls, call)

	if m.ExecuteFunc != nil {
		return m.ExecuteFunc(name, args...)
	}
	return nil, nil
}

// Reset clears all recorded calls.
func (m *MockExecutor) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = make([][]string, 0)
}

// CallCount returns the number of Execute calls made.
func (m *MockExecutor) CallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.Calls)
}

// GetCall returns the call at the given index, or nil if out of range.
func (m *MockExecutor) GetCall(index int) []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if index < 0 || index >= len(m.Calls) {
		return nil
	}
	return m.Calls[index]
}
