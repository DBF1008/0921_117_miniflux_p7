// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package config // import "miniflux.app/v2/internal/config"

import (
	"os"
	"path/filepath"
	"testing"
)

func preserveGlobals(t *testing.T) {
	t.Helper()

	reloadMu.Lock()
	savedOpts := Opts
	savedConfigFilePath := configFilePath
	savedCallbacks := changeCallbacks
	configFilePath = ""
	changeCallbacks = nil
	reloadMu.Unlock()

	t.Cleanup(func() {
		reloadMu.Lock()
		Opts = savedOpts
		configFilePath = savedConfigFilePath
		changeCallbacks = savedCallbacks
		reloadMu.Unlock()
	})
}

func TestReloadPicksUpEnvironmentVariableChanges(t *testing.T) {
	preserveGlobals(t)

	t.Setenv("DATABASE_MAX_CONNS", "10")

	if err := Reload(); err != nil {
		t.Fatalf("Initial reload failed: %v", err)
	}

	if got := Opts.DatabaseMaxConns(); got != 10 {
		t.Fatalf("Expected DATABASE_MAX_CONNS=10, got %d", got)
	}

	// Simulate a runtime environment variable change.
	t.Setenv("DATABASE_MAX_CONNS", "42")

	if got := Opts.DatabaseMaxConns(); got != 10 {
		t.Fatalf("Expected Opts to remain unchanged before reload, got %d", got)
	}

	if err := Reload(); err != nil {
		t.Fatalf("Reload failed: %v", err)
	}

	if got := Opts.DatabaseMaxConns(); got != 42 {
		t.Fatalf("Expected DATABASE_MAX_CONNS=42 after reload, got %d", got)
	}
}

func TestReloadNotifiesChangeCallbacks(t *testing.T) {
	preserveGlobals(t)

	var notified int
	OnConfigChange(func() { notified++ })
	OnConfigChange(func() { notified++ })

	if err := Reload(); err != nil {
		t.Fatalf("Reload failed: %v", err)
	}

	if notified != 2 {
		t.Fatalf("Expected 2 callback notifications, got %d", notified)
	}
}

func TestReloadKeepsPreviousConfigurationOnError(t *testing.T) {
	preserveGlobals(t)

	t.Setenv("DATABASE_MAX_CONNS", "10")

	if err := Reload(); err != nil {
		t.Fatalf("Initial reload failed: %v", err)
	}

	var notified int
	OnConfigChange(func() { notified++ })

	// DATABASE_MIN_CONNS > DATABASE_MAX_CONNS is invalid.
	t.Setenv("DATABASE_MIN_CONNS", "100")

	if err := Reload(); err == nil {
		t.Fatal("Expected reload to fail with invalid configuration")
	}

	if got := Opts.DatabaseMaxConns(); got != 10 {
		t.Fatalf("Expected previous configuration to be kept, got DATABASE_MAX_CONNS=%d", got)
	}

	if notified != 0 {
		t.Fatalf("Expected no callback notification on failed reload, got %d", notified)
	}
}

func TestReloadReadsConfigFile(t *testing.T) {
	preserveGlobals(t)

	configFile := filepath.Join(t.TempDir(), "miniflux.conf")
	if err := os.WriteFile(configFile, []byte("DATABASE_MAX_CONNS=25\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	SetConfigFilePath(configFile)

	if err := Reload(); err != nil {
		t.Fatalf("Reload failed: %v", err)
	}

	if got := Opts.DatabaseMaxConns(); got != 25 {
		t.Fatalf("Expected DATABASE_MAX_CONNS=25 from config file, got %d", got)
	}

	// Simulate a runtime configuration file change.
	if err := os.WriteFile(configFile, []byte("DATABASE_MAX_CONNS=30\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := Reload(); err != nil {
		t.Fatalf("Reload failed: %v", err)
	}

	if got := Opts.DatabaseMaxConns(); got != 30 {
		t.Fatalf("Expected DATABASE_MAX_CONNS=30 after config file change, got %d", got)
	}
}
