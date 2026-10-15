package testutil

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestMainTagMigrationBackfillsExistingEventTag(t *testing.T) {
	ctx := context.Background()
	pool := newTestPool(t, migrationNames[:6])

	userID := uuid.New()
	tagID := uuid.New()
	itemID := uuid.New()
	if _, err := pool.Exec(ctx, `
		INSERT INTO users (id, email, password_hash) VALUES ($1, $2, $3)
	`, userID, "migration@example.com", "hash"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO tags (id, user_id, name) VALUES ($1, $2, $3)
	`, tagID, userID, "custom"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO timeline_items (id, user_id, title) VALUES ($1, $2, $3)
	`, itemID, userID, "Existing event"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO events (timeline_item_id, starts_at, ends_at)
		VALUES ($1, $2, $3)
	`, itemID, time.Date(2026, 10, 15, 9, 0, 0, 0, time.UTC), time.Date(2026, 10, 15, 10, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO timeline_item_tags (timeline_item_id, tag_id) VALUES ($1, $2)
	`, itemID, tagID); err != nil {
		t.Fatal(err)
	}

	applyMigrations(t, ctx, pool, migrationNames[6:9])
	if _, err := pool.Exec(ctx, `
		DELETE FROM timeline_item_tags
		WHERE timeline_item_id = $1 AND tag_id <> $2
	`, itemID, tagID); err != nil {
		t.Fatal(err)
	}
	applyMigrations(t, ctx, pool, migrationNames[9:])

	var mainTagID uuid.UUID
	if err := pool.QueryRow(ctx, `
		SELECT main_tag_id FROM events WHERE timeline_item_id = $1
	`, itemID).Scan(&mainTagID); err != nil {
		t.Fatal(err)
	}
	if mainTagID != tagID {
		t.Fatalf("expected migration to select attached tag %s as main tag, got %s", tagID, mainTagID)
	}

	var mainTagAttached bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM timeline_item_tags
			WHERE timeline_item_id = $1 AND tag_id = $2
		)
	`, itemID, mainTagID).Scan(&mainTagAttached); err != nil {
		t.Fatal(err)
	}
	if !mainTagAttached {
		t.Fatalf("expected migrated main tag %s to remain attached", mainTagID)
	}
}
