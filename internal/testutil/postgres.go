//go:build integration

// Package testutil spins up real infrastructure (Postgres via
// testcontainers-go, with migrations applied) for integration tests. It is
// built only under the "integration" tag so `go test ./...` in normal CI
// runs never require Docker.
package testutil

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	testcontainers "github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// StartPostgres launches a disposable Postgres container, applies every
// migration in /migrations, and returns a connection pool. The container
// and pool are torn down automatically via t.Cleanup.
func StartPostgres(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()

	container, err := tcpostgres.Run(ctx, "postgres:17-alpine",
		tcpostgres.WithDatabase("vn_platform_test"),
		tcpostgres.WithUsername("vn_platform"),
		tcpostgres.WithPassword("vn_platform_test_password"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = container.Terminate(ctx) })

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	// Migrations are applied via an os.DirFS + iofs source instance rather
	// than a "file://" URL: golang-migrate's URL-based file source parses
	// the path through net/url, which mishandles Windows drive letters
	// (e.g. "file:///D:/repo/migrations"); iofs sidesteps that entirely.
	sourceDriver, err := iofs.New(os.DirFS(migrationsDir(t)), ".")
	require.NoError(t, err)

	m, err := migrate.NewWithSourceInstance("iofs", sourceDriver, dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = m.Close() })

	require.NoError(t, m.Up())

	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	return pool
}

// migrationsDir locates the repo's /migrations directory relative to this
// source file, so tests work regardless of the working directory `go test`
// is invoked from.
func migrationsDir(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok, "failed to determine caller for migrations path")
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "migrations")
}
