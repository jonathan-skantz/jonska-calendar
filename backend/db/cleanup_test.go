package db_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/jonathan-skantz/calendar/backend/db"
	"github.com/jonathan-skantz/calendar/backend/internal/testutil"
)

func TestDeleteExpiredSessionsKeepsLiveOnes(t *testing.T) {
	ctx := context.Background()
	store := testutil.NewStore(t)
	userID := registerUser(t, store, "user@example.com")
	// Sessions cannot be created already expired, so this one expires soon.
	if err := store.CreateSession(ctx, userID, []byte("expired"), time.Now().Add(300*time.Millisecond)); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateSession(ctx, userID, []byte("live"), time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}

	time.Sleep(600 * time.Millisecond)

	deleted, err := store.DeleteExpiredSessions(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 1 {
		t.Fatalf("expected 1 deleted session, got %d", deleted)
	}
	if _, err := store.FindUserBySession(ctx, []byte("live")); err != nil {
		t.Fatalf("expected the live session to remain: %v", err)
	}
}

func TestDeleteWeatherFetchedBefore(t *testing.T) {
	ctx := context.Background()
	store := testutil.NewStore(t)
	now := time.Now()
	forecast := func(fetchedAt time.Time) db.WeatherForecast {
		return db.WeatherForecast{FetchedAt: fetchedAt, Hourly: json.RawMessage(`{}`)}
	}
	if err := store.UpsertWeatherForecast(ctx, 59.3, 18.1, forecast(now.AddDate(0, 0, -8))); err != nil {
		t.Fatal(err)
	}
	if err := store.UpsertWeatherForecast(ctx, 57.7, 12.0, forecast(now)); err != nil {
		t.Fatal(err)
	}

	deleted, err := store.DeleteWeatherFetchedBefore(ctx, now.AddDate(0, 0, -7))
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 1 {
		t.Fatalf("expected 1 deleted forecast, got %d", deleted)
	}
	if _, err := store.GetWeatherForecast(ctx, 59.3, 18.1); !errors.Is(err, db.ErrWeatherNotCached) {
		t.Fatalf("expected the old forecast to be gone, got %v", err)
	}
	if _, err := store.GetWeatherForecast(ctx, 57.7, 12.0); err != nil {
		t.Fatalf("expected the fresh forecast to remain: %v", err)
	}
}
