// Package testutil gives each test its own migrated PostgreSQL database.
//
// One container is started per test binary, the first time a test needs a
// database. The migrations are applied once to a template database, and each
// test gets a fresh copy of it, so tests stay isolated without paying for a
// container each. A package that uses the database calls Run from TestMain so
// the container is stopped when its tests finish.
package testutil

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jonathan-skantz/calendar/backend/db"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

const templateDatabase = "migrated_template"

// migrationNames lists the migration files in the order they apply.
var migrationNames = loadMigrationNames()

var (
	startOnce    sync.Once
	container    *postgres.PostgresContainer
	adminURL     string
	startErr     error
	databaseSeq  atomic.Int64
	templateOnce sync.Once
	templateErr  error
)

// Run runs a package's tests and then stops the shared container, if one was
// started. Call it from TestMain.
func Run(m *testing.M) {
	code := m.Run()
	if container != nil {
		if err := container.Terminate(context.Background()); err != nil {
			log.Printf("stopping test database: %v", err)
		}
	}
	os.Exit(code)
}

// SignedInAs stubs a handler's current-user lookup with a fixed user.
func SignedInAs(userID uuid.UUID) func(*http.Request) (uuid.UUID, error) {
	return func(*http.Request) (uuid.UUID, error) { return userID, nil }
}

// NewStore returns a store backed by a fresh database with every migration
// applied.
func NewStore(t *testing.T) *db.Store {
	t.Helper()
	templateOnce.Do(func() { templateErr = createTemplate() })
	if templateErr != nil {
		t.Fatalf("preparing migrated template database: %v", templateErr)
	}
	return db.NewStore(newDatabase(t, templateDatabase))
}

// newDatabase creates an empty database, or a copy of template, that is
// dropped when the test ends.
func newDatabase(t *testing.T, template string) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()
	admin := adminConnection(t)
	name := fmt.Sprintf("test_%d", databaseSeq.Add(1))
	if _, err := admin.Exec(ctx, fmt.Sprintf("CREATE DATABASE %s TEMPLATE %s", name, template)); err != nil {
		t.Fatalf("creating test database: %v", err)
	}
	pool, err := db.NewDBPool(ctx, databaseURL(name))
	if err != nil {
		t.Fatalf("connecting to test database: %v", err)
	}
	t.Cleanup(func() {
		pool.Close()
		if _, err := admin.Exec(ctx, fmt.Sprintf("DROP DATABASE %s", name)); err != nil {
			t.Errorf("dropping test database: %v", err)
		}
	})
	return pool
}

// createTemplate applies every migration to the template database once.
// Postgres copies a template only while nothing is connected to it, so the
// connection is closed before any test clones it.
func createTemplate() error {
	ctx := context.Background()
	if err := startContainer(); err != nil {
		return err
	}
	admin, err := pgx.Connect(ctx, adminURL)
	if err != nil {
		return err
	}
	defer admin.Close(ctx)
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+templateDatabase); err != nil {
		return err
	}
	conn, err := pgx.Connect(ctx, databaseURL(templateDatabase))
	if err != nil {
		return err
	}
	defer conn.Close(ctx)
	for _, name := range migrationNames {
		if err := applyMigration(ctx, conn, name); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
	}
	return nil
}

func startContainer() error {
	startOnce.Do(func() {
		ctx := context.Background()
		container, startErr = postgres.Run(ctx,
			"postgres:15.3-alpine",
			postgres.WithDatabase("postgres"),
			postgres.WithUsername("testuser"),
			postgres.WithPassword("testpassword"),
			postgres.BasicWaitStrategies(),
		)
		if startErr != nil {
			return
		}
		adminURL, startErr = container.ConnectionString(ctx, "sslmode=disable")
	})
	return startErr
}

// adminConnection connects to the maintenance database, which creates and
// drops the per-test databases.
func adminConnection(t *testing.T) *pgx.Conn {
	t.Helper()
	if err := startContainer(); err != nil {
		t.Fatalf("starting test database container: %v", err)
	}
	conn, err := pgx.Connect(context.Background(), adminURL)
	if err != nil {
		t.Fatalf("connecting to test database server: %v", err)
	}
	// Registered first so it closes last, after the cleanup that drops the
	// test's database through it.
	t.Cleanup(func() { _ = conn.Close(context.Background()) })
	return conn
}

// databaseURL points the admin URL at another database on the same server.
func databaseURL(name string) string {
	base, query, _ := strings.Cut(adminURL, "?")
	return base[:strings.LastIndex(base, "/")+1] + name + "?" + query
}

// execer is a connection or pool that can run a migration.
type execer interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// newTestPool returns a fresh database with only the given migrations
// applied, for tests of the migrations themselves.
func newTestPool(t *testing.T, migrations []string) *pgxpool.Pool {
	t.Helper()
	pool := newDatabase(t, "template0")
	applyMigrations(t, context.Background(), pool, migrations)
	return pool
}

func applyMigrations(t *testing.T, ctx context.Context, conn execer, migrations []string) {
	t.Helper()
	for _, name := range migrations {
		if err := applyMigration(ctx, conn, name); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
}

func applyMigration(ctx context.Context, conn execer, name string) error {
	migration, err := os.ReadFile(filepath.Join(migrationDirectory(), name))
	if err != nil {
		return err
	}
	upMigration, _, _ := strings.Cut(string(migration), "-- +goose Down")
	_, err = conn.Exec(ctx, upMigration)
	return err
}

func migrationDirectory() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "..", "db", "migrations")
}

// loadMigrationNames reads the migration files, ordered by the number Goose
// expects in front of each name.
func loadMigrationNames() []string {
	entries, err := os.ReadDir(migrationDirectory())
	if err != nil {
		panic(fmt.Sprintf("reading migrations: %v", err))
	}
	var names []string
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".sql") {
			names = append(names, entry.Name())
		}
	}
	version := func(name string) int {
		prefix, _, _ := strings.Cut(name, "_")
		number, err := strconv.Atoi(prefix)
		if err != nil {
			panic(fmt.Sprintf("migration %s does not start with a number", name))
		}
		return number
	}
	sort.Slice(names, func(i, j int) bool { return version(names[i]) < version(names[j]) })
	return names
}
