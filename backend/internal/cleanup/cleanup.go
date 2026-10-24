// Package cleanup periodically deletes rows that are no longer used, so
// tables do not grow without bound on a long-running server.
package cleanup

import (
	"context"
	"log"
	"time"

	"github.com/jonathan-skantz/calendar/backend/db"
)

const (
	runEvery = time.Hour
	// weatherKeptFor is how long a cached forecast is kept. Older forecasts
	// are refetched before they are shown anyway.
	weatherKeptFor = 7 * 24 * time.Hour
)

// Run deletes expired rows now and then every hour, until ctx ends.
func Run(ctx context.Context, store *db.Store) {
	ticker := time.NewTicker(runEvery)
	defer ticker.Stop()
	for {
		Once(ctx, store)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// Once deletes expired demo users with everything they own, expired
// sessions, and old weather forecasts. Failures are logged, and the next run
// tries again.
func Once(ctx context.Context, store *db.Store) {
	if _, err := store.DeleteExpiredDemoUsers(ctx); err != nil {
		log.Printf("deleting expired demo users: %v", err)
	}
	if _, err := store.DeleteExpiredSessions(ctx); err != nil {
		log.Printf("deleting expired sessions: %v", err)
	}
	if _, err := store.DeleteWeatherFetchedBefore(ctx, time.Now().Add(-weatherKeptFor)); err != nil {
		log.Printf("deleting old weather forecasts: %v", err)
	}
}
