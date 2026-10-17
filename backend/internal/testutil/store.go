package testutil

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jonathan-skantz/calendar/backend/db"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

var migrationNames = []string{
	"1_add_users.sql",
	"2_add_sessions.sql",
	"3_add_timeline_demo.sql",
	"4_add_event_sharing.sql",
	"5_remove_redundant_share_owner.sql",
	"6_add_tags.sql",
	"7_add_default_event_tags.sql",
	"8_defer_default_tag_fk.sql",
	"9_add_tag_colors.sql",
	"10_add_event_main_tags.sql",
	"11_add_all_day_events.sql",
	"12_add_transit_card_days.sql",
	"13_add_user_display_name.sql",
	"14_restrict_event_commute.sql",
	"15_add_households.sql",
}

func NewStore(t *testing.T) *db.Store {
	return db.NewStore(newTestPool(t, migrationNames))
}

func newTestPool(t *testing.T, migrations []string) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()
	container, err := postgres.Run(
		ctx,
		"postgres:15.3-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpassword"),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = container.Terminate(ctx) })

	connectionString, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}

	pool := connect(t, ctx, connectionString)
	t.Cleanup(pool.Close)
	applyMigrations(t, ctx, pool, migrations)
	return pool
}

func connect(t *testing.T, ctx context.Context, connectionString string) *pgxpool.Pool {
	t.Helper()
	var pool *pgxpool.Pool
	var err error
	for range 20 {
		pool, err = db.NewDBPool(ctx, connectionString)
		if err == nil {
			return pool
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatalf("failed to connect to test database: %v", err)
	return nil
}

func applyMigrations(t *testing.T, ctx context.Context, pool *pgxpool.Pool, migrations []string) {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to locate test utility source")
	}
	migrationDirectory := filepath.Join(filepath.Dir(filename), "..", "..", "db", "migrations")
	for _, migrationName := range migrations {
		migration, err := os.ReadFile(filepath.Join(migrationDirectory, migrationName))
		if err != nil {
			t.Fatal(err)
		}
		upMigration, _, _ := strings.Cut(string(migration), "-- +goose Down")
		if _, err := pool.Exec(ctx, upMigration); err != nil {
			t.Fatal(err)
		}
	}
}
