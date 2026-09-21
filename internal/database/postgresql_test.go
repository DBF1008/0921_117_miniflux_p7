// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package database // import "miniflux.app/v2/internal/database"

import (
	"database/sql"
	"testing"
	"time"
)

func TestConfigureConnectionPool(t *testing.T) {
	db, err := sql.Open("postgres", "postgres://localhost:5432/miniflux_test?sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ConfigureConnectionPool(db, 3, 17, time.Minute)

	if got := db.Stats().MaxOpenConnections; got != 17 {
		t.Fatalf("Expected MaxOpenConnections=17, got %d", got)
	}

	// Reconfiguring the pool at runtime must apply the new settings without
	// rebuilding the pool.
	ConfigureConnectionPool(db, 1, 42, 2*time.Minute)

	if got := db.Stats().MaxOpenConnections; got != 42 {
		t.Fatalf("Expected MaxOpenConnections=42 after reconfiguration, got %d", got)
	}
}

func TestNewConnectionPoolAppliesSettings(t *testing.T) {
	db, err := NewConnectionPool("postgres://localhost:5432/miniflux_test?sslmode=disable", 2, 23, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if got := db.Stats().MaxOpenConnections; got != 23 {
		t.Fatalf("Expected MaxOpenConnections=23, got %d", got)
	}
}
