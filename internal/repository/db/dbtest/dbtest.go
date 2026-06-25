// Package dbtest provides a transactional test helper backed by the
// pixelpresent_test database. Each call to Tx hands out a transaction that is
// rolled back when the test finishes, so tests never see each other's writes.
package dbtest

import (
	"context"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"database/sql"

	"github.com/JairoRiver/pixelpresent/internal/repository/db/migrations"
	"github.com/JairoRiver/pixelpresent/internal/util"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
)

const testDBName = "pixelpresent_test"

var (
	pool      *pgxpool.Pool
	setupErr  error
	setupOnce sync.Once
)

// Pool returns a shared connection pool to pixelpresent_test, creating the
// database and applying migrations on first use. Most tests should use Tx
// instead so their writes are isolated.
func Pool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	setupOnce.Do(setup)
	require.NoError(t, setupErr)
	return pool
}

// Tx begins a transaction on the test pool and registers a cleanup that rolls
// it back when the test ends, isolating each test from the others.
func Tx(t *testing.T) pgx.Tx {
	t.Helper()
	p := Pool(t)

	tx, err := p.Begin(context.Background())
	require.NoError(t, err)

	t.Cleanup(func() {
		// Best-effort rollback; the connection may already be gone.
		_ = tx.Rollback(context.Background())
	})

	return tx
}

func setup() {
	root, err := projectRoot()
	if err != nil {
		setupErr = err
		return
	}

	config, err := util.LoadConfig(filepath.Join(root, util.DefaultConfigPath))
	if err != nil {
		setupErr = err
		return
	}

	testDSN, err := withDatabase(config.Database.DSN, testDBName)
	if err != nil {
		setupErr = err
		return
	}

	if err := ensureTestDatabase(config.Database.DSN); err != nil {
		setupErr = err
		return
	}

	if err := applyMigrations(testDSN); err != nil {
		setupErr = err
		return
	}

	pool, setupErr = pgxpool.New(context.Background(), testDSN)
}

// ensureTestDatabase creates pixelpresent_test if it does not exist yet,
// connecting to the default "postgres" maintenance database.
func ensureTestDatabase(mainDSN string) error {
	adminDSN, err := withDatabase(mainDSN, "postgres")
	if err != nil {
		return err
	}

	conn, err := pgx.Connect(context.Background(), adminDSN)
	if err != nil {
		return err
	}
	defer conn.Close(context.Background())

	var exists bool
	if err := conn.QueryRow(context.Background(),
		"SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)", testDBName,
	).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return nil
	}

	// Database names cannot be parameterized; testDBName is a fixed constant.
	_, err = conn.Exec(context.Background(), "CREATE DATABASE "+testDBName)
	return err
}

func applyMigrations(testDSN string) error {
	db, err := sql.Open("pgx", testDSN)
	if err != nil {
		return err
	}
	defer db.Close()

	source, err := iofs.New(migrations.MigrationsFS, ".")
	if err != nil {
		return err
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithInstance("iofs", source, "pixelpresent", driver)
	if err != nil {
		return err
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}

// withDatabase returns dsn with its database path replaced by name.
func withDatabase(dsn, name string) (string, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return "", err
	}
	u.Path = "/" + name
	return u.String(), nil
}

// projectRoot walks up from the working directory until it finds go.mod, so
// the helper locates config.yaml regardless of which package's tests call it.
func projectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("dbtest: go.mod not found walking up from working directory")
		}
		dir = parent
	}
}
