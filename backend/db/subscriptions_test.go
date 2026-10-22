package db_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jonathan-skantz/calendar/backend/db"
	"github.com/jonathan-skantz/calendar/backend/internal/testutil"
)

func subscribedEvent(title string, startsAt time.Time) db.SubscribedEvent {
	return db.SubscribedEvent{Event: db.Event{
		Title: title, StartsAt: startsAt, EndsAt: startsAt.Add(time.Hour), Kind: db.EventKindAttendable,
	}}
}

func TestSubscriptionEventsAreProjectedToTheirOwnerOnly(t *testing.T) {
	ctx := context.Background()
	store := testutil.NewStore(t)
	userID := registerUser(t, store, "user@example.com")
	otherID := registerUser(t, store, "other@example.com")
	startsAt := time.Date(2026, 10, 19, 9, 0, 0, 0, time.UTC)

	// A weekly series whose second occurrence moved a day earlier.
	rule, zone := "FREQ=WEEKLY", "UTC"
	series := subscribedEvent("Lecture", startsAt)
	series.RecurrenceRule, series.RecurrenceTimeZone = &rule, &zone
	movedFrom := startsAt.AddDate(0, 0, 7)
	movedTo, movedEnd := movedFrom.AddDate(0, 0, -1), movedFrom.AddDate(0, 0, -1).Add(time.Hour)
	series.Exceptions = []db.OccurrenceException{{OriginalStart: movedFrom, StartsAt: &movedTo, EndsAt: &movedEnd}}

	subscription, err := store.CreateSubscription(ctx, db.Subscription{
		UserID: userID, Name: "School", URL: "https://example.com/school.ics", Color: "#336699", TimeZone: "UTC",
	}, []db.SubscribedEvent{series, subscribedEvent("Exam", startsAt.Add(2*time.Hour))})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateSubscription(ctx, db.Subscription{
		UserID: userID, Name: "Again", URL: subscription.URL, Color: "#336699", TimeZone: "UTC",
	}, nil); !errors.Is(err, db.ErrSubscriptionExists) {
		t.Fatalf("expected a duplicate URL to be refused, got %v", err)
	}

	entries, err := store.ListTimeline(ctx, userID, startsAt.AddDate(0, 0, 5), startsAt.AddDate(0, 0, 9))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Subscription == nil || entries[0].Subscription.Name != "School" ||
		len(entries[0].Exceptions) != 1 || len(entries[0].Tags) != 0 {
		t.Fatalf("expected the series with its exception and no tags, got %+v", entries)
	}
	if others, err := store.ListTimeline(ctx, otherID, startsAt, startsAt.AddDate(0, 0, 9)); err != nil || len(others) != 0 {
		t.Fatalf("expected nothing for another user, got %+v, %v", others, err)
	}

	if _, err := store.ReplaceSubscriptionEvents(ctx, otherID, subscription.ID, nil); !errors.Is(err, db.ErrSubscriptionNotFound) {
		t.Fatalf("expected another user's refresh to be refused, got %v", err)
	}
	failed, err := store.MarkSubscriptionFailed(ctx, userID, subscription.ID, "the calendar answered 404")
	if err != nil || failed.LastError == nil {
		t.Fatalf("expected the error recorded, got %+v, %v", failed, err)
	}
	refreshed, err := store.ReplaceSubscriptionEvents(ctx, userID, subscription.ID,
		[]db.SubscribedEvent{subscribedEvent("Only one", startsAt)})
	if err != nil || refreshed.LastError != nil {
		t.Fatalf("expected the error cleared, got %+v, %v", refreshed, err)
	}
	entries, err = store.ListTimeline(ctx, userID, startsAt, startsAt.AddDate(0, 0, 9))
	if err != nil || len(entries) != 1 || entries[0].Title != "Only one" {
		t.Fatalf("expected the events replaced, got %+v, %v", entries, err)
	}

	stale, err := store.StaleSubscriptions(ctx, time.Now().Add(time.Hour))
	if err != nil || len(stale) != 1 {
		t.Fatalf("expected one stale subscription, got %+v, %v", stale, err)
	}
	if stale, err := store.StaleSubscriptions(ctx, time.Now().Add(-time.Hour)); err != nil || len(stale) != 0 {
		t.Fatalf("expected no stale subscriptions, got %+v, %v", stale, err)
	}

	if err := store.DeleteSubscription(ctx, otherID, subscription.ID); !errors.Is(err, db.ErrSubscriptionNotFound) {
		t.Fatalf("expected another user's delete to be refused, got %v", err)
	}
	if err := store.DeleteSubscription(ctx, userID, subscription.ID); err != nil {
		t.Fatal(err)
	}
	if entries, err := store.ListTimeline(ctx, userID, startsAt, startsAt.AddDate(0, 0, 9)); err != nil || len(entries) != 0 {
		t.Fatalf("expected the events removed with the subscription, got %+v, %v", entries, err)
	}
}
