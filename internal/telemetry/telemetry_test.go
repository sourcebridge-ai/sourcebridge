// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package telemetry

import (
	"runtime"
	"testing"
)

func TestResolvePlatform_DefaultsToRuntime(t *testing.T) {
	t.Setenv("SOURCEBRIDGE_TELEMETRY_PLATFORM", "")
	got := resolvePlatform()
	want := runtime.GOOS + "/" + runtime.GOARCH
	if got != want {
		t.Errorf("resolvePlatform() = %q; want %q", got, want)
	}
}

func TestResolvePlatform_EnvOverrideWins(t *testing.T) {
	t.Setenv("SOURCEBRIDGE_TELEMETRY_PLATFORM", "test")
	if got := resolvePlatform(); got != "test" {
		t.Errorf("resolvePlatform() = %q; want %q", got, "test")
	}
}

func TestResolvePlatform_TrimsWhitespace(t *testing.T) {
	t.Setenv("SOURCEBRIDGE_TELEMETRY_PLATFORM", "  test  ")
	if got := resolvePlatform(); got != "test" {
		t.Errorf("resolvePlatform() = %q; want %q (trimmed)", got, "test")
	}
}

func TestResolvePlatform_BlankEnvFallsBackToRuntime(t *testing.T) {
	t.Setenv("SOURCEBRIDGE_TELEMETRY_PLATFORM", "   ")
	got := resolvePlatform()
	want := runtime.GOOS + "/" + runtime.GOARCH
	if got != want {
		t.Errorf("resolvePlatform() with whitespace-only env = %q; want runtime default %q", got, want)
	}
}
