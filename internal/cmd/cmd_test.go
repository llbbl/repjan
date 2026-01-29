// SPDX-FileCopyrightText: 2026 Logan Lindquist Land
// SPDX-License-Identifier: FSL-1.1-MIT

package cmd

import (
	"testing"
)

func TestGetOwner_DefaultEmpty(t *testing.T) {
	// Reset the flag value for testing
	owner = ""

	got := GetOwner()
	if got != "" {
		t.Errorf("GetOwner() = %q, want empty string", got)
	}
}

func TestIsFabricEnabled_DefaultFalse(t *testing.T) {
	// Reset the flag value for testing
	fabric = false

	if IsFabricEnabled() {
		t.Error("IsFabricEnabled() = true, want false")
	}
}

func TestGetFabricPath_DefaultValue(t *testing.T) {
	// Reset the flag value for testing
	fabricPath = "fabric"

	got := GetFabricPath()
	if got != "fabric" {
		t.Errorf("GetFabricPath() = %q, want \"fabric\"", got)
	}
}

func TestGetConfig_NilBeforeExecute(t *testing.T) {
	// Before Execute() is called, cfg should be nil
	cfg = nil

	got := GetConfig()
	if got != nil {
		t.Error("GetConfig() should return nil before Execute()")
	}
}

func TestVersionVariable(t *testing.T) {
	// Version should have a default value
	if Version == "" {
		t.Error("Version should have a default value")
	}
}
