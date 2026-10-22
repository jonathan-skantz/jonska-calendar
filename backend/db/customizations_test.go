package db_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jonathan-skantz/calendar/backend/db"
	"github.com/jonathan-skantz/calendar/backend/internal/testutil"
)

// receivedEntry returns the user's only timeline entry around startsAt.
// Received events are read a day around the range, so tests keep them
// further apart.
func receivedEntry(t *testing.T, store *db.Store, userID uuid.UUID, startsAt time.Time) db.TimelineEntry {
	t.Helper()
	entries, err := store.ListTimeline(context.Background(), userID, startsAt.Add(-time.Hour), startsAt.Add(2*time.Hour))
	if err != nil || len(entries) != 1 {
		t.Fatalf("expected one entry, got %+v, %v", entries, err)
	}
	return entries[0]
}

func TestSharedEventCustomizationIsTheRecipientsOwn(t *testing.T) {
	ctx := context.Background()
	store := testutil.NewStore(t)
	ownerID := registerUser(t, store, "owner@example.com")
	recipientID := registerUser(t, store, "recipient@example.com")
	strangerID := registerUser(t, store, "stranger@example.com")
	startsAt := time.Date(2026, 10, 19, 9, 0, 0, 0, time.UTC)
	eventID, err := store.CreateEvent(ctx, ownerID, db.Event{
		Title: "Lecture", StartsAt: startsAt, EndsAt: startsAt.Add(time.Hour), Kind: db.EventKindAttendable,
	})
	if err != nil {
		t.Fatal(err)
	}
	shareID, err := store.ShareEvent(ctx, ownerID, eventID, recipientID, db.ShareFields{})
	if err != nil {
		t.Fatal(err)
	}

	customization := db.Customization{Title: new("My lecture"), Note: new("Zoom"), StartShiftSeconds: 900}
	for _, userID := range []uuid.UUID{ownerID, strangerID} {
		if err := store.CustomizeSharedEvent(ctx, userID, eventID, customization); !errors.Is(err, db.ErrSharedEventNotFound) {
			t.Fatalf("expected only the recipient to customize, got %v", err)
		}
		if err := store.UncustomizeSharedEvent(ctx, userID, eventID); !errors.Is(err, db.ErrSharedEventNotFound) {
			t.Fatalf("expected only the recipient to remove a customization, got %v", err)
		}
	}
	if err := store.CustomizeSharedEvent(ctx, recipientID, eventID, db.Customization{Title: new("First")}); err != nil {
		t.Fatal(err)
	}
	if err := store.CustomizeSharedEvent(ctx, recipientID, eventID, customization); err != nil {
		t.Fatal(err)
	}

	got := receivedEntry(t, store, recipientID, startsAt).Customization
	if got == nil || *got.Title != "My lecture" || got.Description != nil || *got.Note != "Zoom" ||
		got.StartShiftSeconds != 900 || got.EndShiftSeconds != 0 {
		t.Fatalf("expected the second customization to replace the first, got %+v", got)
	}
	if own := receivedEntry(t, store, ownerID, startsAt); own.Customization != nil || own.Title != "Lecture" {
		t.Fatalf("expected the owner not to see the customization, got %+v", own)
	}

	if err := store.UncustomizeSharedEvent(ctx, recipientID, eventID); err != nil {
		t.Fatal(err)
	}
	if err := store.UncustomizeSharedEvent(ctx, recipientID, eventID); err != nil {
		t.Fatalf("expected removing a missing customization to succeed, got %v", err)
	}
	if got := receivedEntry(t, store, recipientID, startsAt).Customization; got != nil {
		t.Fatalf("expected the customization removed, got %+v", got)
	}

	if err := store.CustomizeSharedEvent(ctx, recipientID, eventID, customization); err != nil {
		t.Fatal(err)
	}
	if err := store.DeleteEventShare(ctx, ownerID, eventID, shareID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ShareEvent(ctx, ownerID, eventID, recipientID, db.ShareFields{}); err != nil {
		t.Fatal(err)
	}
	if got := receivedEntry(t, store, recipientID, startsAt).Customization; got != nil {
		t.Fatalf("expected the customization removed with the share, got %+v", got)
	}
}

func TestSubscribedEventCustomizationSurvivesARefresh(t *testing.T) {
	ctx := context.Background()
	store := testutil.NewStore(t)
	userID := registerUser(t, store, "user@example.com")
	otherID := registerUser(t, store, "other@example.com")
	startsAt := time.Date(2026, 10, 19, 9, 0, 0, 0, time.UTC)
	lecture := subscribedEvent("Lecture", startsAt)
	lecture.UID = new("lecture@school")
	exam := subscribedEvent("Exam", startsAt.AddDate(0, 0, 3))

	subscription, err := store.CreateSubscription(ctx, db.Subscription{
		UserID: userID, Name: "School", URL: "https://example.com/school.ics", Color: "#336699", TimeZone: "UTC",
	}, []db.SubscribedEvent{lecture, exam})
	if err != nil {
		t.Fatal(err)
	}
	lectureID := receivedEntry(t, store, userID, startsAt).ID
	examID := receivedEntry(t, store, userID, startsAt.AddDate(0, 0, 3)).ID

	customization := db.Customization{Location: new("Zoom"), EndShiftSeconds: -1800}
	if err := store.CustomizeSubscribedEvent(ctx, otherID, lectureID, customization); !errors.Is(err, db.ErrSubscribedEventNotFound) {
		t.Fatalf("expected another user to be refused, got %v", err)
	}
	if err := store.CustomizeSubscribedEvent(ctx, userID, examID, customization); !errors.Is(err, db.ErrEventWithoutUID) {
		t.Fatalf("expected an event without a UID to be refused, got %v", err)
	}
	if err := store.CustomizeSubscribedEvent(ctx, userID, lectureID, customization); err != nil {
		t.Fatal(err)
	}

	// The refresh gives the lecture a new row and moves it an hour.
	lecture.StartsAt, lecture.EndsAt = lecture.StartsAt.Add(time.Hour), lecture.EndsAt.Add(time.Hour)
	if _, err := store.ReplaceSubscriptionEvents(ctx, userID, subscription.ID, []db.SubscribedEvent{lecture}); err != nil {
		t.Fatal(err)
	}
	refreshed := receivedEntry(t, store, userID, lecture.StartsAt)
	if refreshed.ID == lectureID || refreshed.Customization == nil || *refreshed.Customization.Location != "Zoom" ||
		refreshed.Customization.EndShiftSeconds != -1800 {
		t.Fatalf("expected the customization kept by UID, got %+v", refreshed)
	}
	if err := store.CustomizeSubscribedEvent(ctx, userID, lectureID, customization); !errors.Is(err, db.ErrSubscribedEventNotFound) {
		t.Fatalf("expected the replaced row to be gone, got %v", err)
	}

	if err := store.UncustomizeSubscribedEvent(ctx, userID, refreshed.ID); err != nil {
		t.Fatal(err)
	}
	if got := receivedEntry(t, store, userID, lecture.StartsAt).Customization; got != nil {
		t.Fatalf("expected the customization removed, got %+v", got)
	}

	if err := store.CustomizeSubscribedEvent(ctx, userID, refreshed.ID, customization); err != nil {
		t.Fatal(err)
	}
	if err := store.DeleteSubscription(ctx, userID, subscription.ID); err != nil {
		t.Fatal(err)
	}
	again, err := store.CreateSubscription(ctx, db.Subscription{
		UserID: userID, Name: "School", URL: subscription.URL, Color: "#336699", TimeZone: "UTC",
	}, []db.SubscribedEvent{lecture})
	if err != nil {
		t.Fatal(err)
	}
	if got := receivedEntry(t, store, userID, lecture.StartsAt); got.Customization != nil || got.Subscription.ID != again.ID {
		t.Fatalf("expected the customization removed with the subscription, got %+v", got)
	}
}
