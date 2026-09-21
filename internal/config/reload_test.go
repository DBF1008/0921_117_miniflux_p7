// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package config // import "miniflux.app/v2/internal/config"

import (
	"os"
	"testing"
)

// sanitizeEnv unsets environment variables that may have been polluted by
// other tests (parser_test.go uses os.Setenv without cleanup) and restores
// them when the test ends.
func sanitizeEnv(t *testing.T, keys ...string) {
	t.Helper()

	for _, key := range keys {
		if value, ok := os.LookupEnv(key); ok {
			os.Unsetenv(key)
			t.Cleanup(func() { os.Setenv(key, value) })
		}
	}
}

func TestReloadUpdatesOptsAndNotifiesSubscribers(t *testing.T) {
	sanitizeEnv(t, "LOG_DATE_TIME", "ADMIN_PASSWORD_FILE", "LOG_FILE")
	t.Setenv("DATABASE_MAX_CONNS", "10")

	initialOpts, err := NewConfigParser().ParseEnvironmentVariables()
	if err != nil {
		t.Fatalf("unable to parse initial configuration: %v", err)
	}
	Opts = initialOpts

	notified := 0
	OnReload(func() {
		notified++
	})

	// Simulate a runtime configuration change.
	t.Setenv("DATABASE_MAX_CONNS", "42")

	if err := Reload(); err != nil {
		t.Fatalf("unable to reload configuration: %v", err)
	}

	if Opts.DatabaseMaxConns() != 42 {
		t.Errorf("Expected DATABASE_MAX_CONNS to be 42 after reload, got %d", Opts.DatabaseMaxConns())
	}

	if notified == 0 {
		t.Error("Expected reload callbacks to be notified")
	}
}

func TestReloadKeepsCurrentConfigOnInvalidValues(t *testing.T) {
	sanitizeEnv(t, "LOG_DATE_TIME", "ADMIN_PASSWORD_FILE", "LOG_FILE")
	t.Setenv("DATABASE_MIN_CONNS", "1")
	t.Setenv("DATABASE_MAX_CONNS", "10")

	initialOpts, err := NewConfigParser().ParseEnvironmentVariables()
	if err != nil {
		t.Fatalf("unable to parse initial configuration: %v", err)
	}
	Opts = initialOpts

	// DATABASE_MIN_CONNS > DATABASE_MAX_CONNS must be rejected.
	t.Setenv("DATABASE_MIN_CONNS", "50")

	if err := Reload(); err == nil {
		t.Fatal("Expected an error when reloading an invalid configuration")
	}

	if Opts != initialOpts {
		t.Error("Expected the current configuration to be preserved after a failed reload")
	}

	if Opts.DatabaseMinConns() != 1 {
		t.Errorf("Expected DATABASE_MIN_CONNS to remain 1, got %d", Opts.DatabaseMinConns())
	}
}
