#!/bin/sh
# Unit test script for the configuration hot-reload, database connection
# pool, and systemd watchdog changes.
#
# Usage: ./test.sh
#
# Sections 1-4 run automatically. Section 5 is a manual test checklist for
# the scenarios that require a live PostgreSQL server and/or systemd.

set -e

echo "==> 1/4 Building all packages"
go build ./...

echo "==> 2/4 Running go vet"
go vet ./...

echo "==> 3/4 Checking gofmt (informational)"
# Note: this checkout uses CRLF line endings, which gofmt normalizes to LF,
# so the check is informational instead of fatal (same as "make lint").
gofmt -l . || true

echo "==> 4/4 Running unit tests"
# Targeted tests for the modified packages.
go test -v -count=1 ./internal/config/ ./internal/database/ ./internal/storage/

# Full unit test suite.
go test -cover -race -count=1 ./...

cat <<'MANUAL'

================================================================
Manual test checklist (requires PostgreSQL and/or systemd)
================================================================

1. Configuration hot-reload (SIGHUP)

   a. Start Miniflux with a configuration file:
        DATABASE_URL=postgres://postgres:postgres@localhost/miniflux?sslmode=disable \
        RUN_MIGRATIONS=1 CREATE_ADMIN=1 ADMIN_USERNAME=admin ADMIN_PASSWORD=test123 \
        ./miniflux -c /tmp/miniflux.conf &
        echo $! > /tmp/miniflux.pid

   b. Edit /tmp/miniflux.conf and change DATABASE_MAX_CONNS (e.g. from 20 to 5).

   c. Trigger a configuration reload:
        kill -HUP $(cat /tmp/miniflux.pid)

   d. Check the logs. Expected output:
        "Received SIGHUP, reloading configuration"
        "Configuration reloaded successfully"
        "Database connection pool reconfigured after configuration reload" max_conns=5

   e. Confirm the new pool limit is applied without restart:
        curl -s http://localhost:8080/metrics | grep miniflux_db_connections_max
      (with METRICS_COLLECTOR=1 and METRICS_* credentials configured)

   f. Set an invalid combination (DATABASE_MIN_CONNS > DATABASE_MAX_CONNS)
      and send SIGHUP again: the reload must fail with an error log and the
      previous configuration must remain active.

2. Migration connection pool limits

   a. Run the migrations with a low pool limit:
        DATABASE_MAX_CONNS=2 DATABASE_MIN_CONNS=1 ./miniflux -migrate
   b. Migrations must succeed; the pool used by Migrate is bounded by
      SetMaxOpenConns/SetConnMaxLifetime instead of the driver defaults.

3. Systemd watchdog connection reuse

   a. Install the unit file (packaging/systemd/miniflux.service) which sets
      WatchdogSec=60s, then:
        systemctl daemon-reload
        systemctl start miniflux
   b. The watchdog goroutine acquires one dedicated connection
      (Storage.NewConnectionPinger) and reuses it for every ping.
   c. Verify the service stays alive and the watchdog is armed:
        systemctl show miniflux -p WatchdogUSec
        journalctl -u miniflux -f
      No "Unable to ping database" errors should appear, and the number of
      PostgreSQL connections must stay constant over time:
        watch "psql -c 'select count(*) from pg_stat_activity' miniflux"
   d. Stop PostgreSQL temporarily: watchdog pings must fail and systemd
      must restart the service after WatchdogSec expires.

================================================================
MANUAL

echo "All automated tests passed."
