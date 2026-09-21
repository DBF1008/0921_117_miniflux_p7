// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package storage // import "miniflux.app/v2/internal/storage"

import (
	"context"
	"database/sql"
	"time"
)

// Storage handles all operations related to the database.
type Storage struct {
	db *sql.DB
}

// NewStorage returns a new Storage.
func NewStorage(db *sql.DB) *Storage {
	return &Storage{db}
}

// DatabaseVersion returns the version of the database which is in use.
func (s *Storage) DatabaseVersion() string {
	var dbVersion string
	err := s.db.QueryRow(`SELECT current_setting('server_version')`).Scan(&dbVersion)
	if err != nil {
		return err.Error()
	}

	return dbVersion
}

// Ping checks if the database connection works.
func (s *Storage) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return s.db.PingContext(ctx)
}

// ConnectionPinger reuses a single dedicated database connection to perform
// repeated health checks without acquiring a new pooled connection and
// creating a new timeout context on every check.
type ConnectionPinger struct {
	conn *sql.Conn
}

// NewConnectionPinger acquires a dedicated connection from the pool that can
// be reused for repeated ping checks (e.g. the systemd watchdog).
func (s *Storage) NewConnectionPinger(ctx context.Context) (*ConnectionPinger, error) {
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return nil, err
	}

	return &ConnectionPinger{conn: conn}, nil
}

// Ping checks if the database connection works using the dedicated connection.
func (p *ConnectionPinger) Ping(ctx context.Context) error {
	return p.conn.PingContext(ctx)
}

// Close returns the dedicated connection to the pool.
func (p *ConnectionPinger) Close() error {
	return p.conn.Close()
}

// DBStats returns database statistics.
func (s *Storage) DBStats() sql.DBStats {
	return s.db.Stats()
}

// DBSize returns how much size the database is using in a pretty way.
func (s *Storage) DBSize() (string, error) {
	var size string

	err := s.db.QueryRow("SELECT pg_size_pretty(pg_database_size(current_database()))").Scan(&size)
	if err != nil {
		return "", err
	}

	return size, nil
}
