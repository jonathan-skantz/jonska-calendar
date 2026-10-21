package db_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jonathan-skantz/calendar/backend/db"
	"github.com/jonathan-skantz/calendar/backend/internal/testutil"
)

func registerUser(t *testing.T, store *db.Store, email string) uuid.UUID {
	t.Helper()
	userID, err := store.Register(context.Background(), email, "hash")
	if err != nil {
		t.Fatal(err)
	}
	return userID
}

func importedEvent(uid, title string, startsAt time.Time) db.ImportedEvent {
	return db.ImportedEvent{
		Event: db.Event{
			Title:    title,
			StartsAt: startsAt,
			EndsAt:   startsAt.Add(time.Hour),
			Kind:     db.EventKindAttendable,
		},
		ICalUID: uid,
	}
}

func TestImportEventsSkipsKnownUIDs(t *testing.T) {
	ctx := context.Background()
	store := testutil.NewStore(t)
	userID := registerUser(t, store, "owner@example.com")
	startsAt := time.Now().Add(24 * time.Hour).Truncate(time.Minute).UTC()

	ownID, err := store.CreateEvent(ctx, userID, db.Event{
		Title: "Made in the app", StartsAt: startsAt, EndsAt: startsAt.Add(time.Hour), Kind: db.EventKindAttendable,
	})
	if err != nil {
		t.Fatal(err)
	}
	events := []db.ImportedEvent{
		importedEvent("a@example.com", "First", startsAt),
		importedEvent("a@example.com", "Repeated in the file", startsAt),
		importedEvent(ownID.String()+db.ExportUIDSuffix, "Own export", startsAt),
		importedEvent("", "No UID", startsAt),
	}

	preview, err := store.ImportEvents(ctx, userID, uuid.Nil, events, true)
	if err != nil {
		t.Fatal(err)
	}
	want := []bool{true, false, false, true}
	if !slices.Equal(preview, want) {
		t.Fatalf("dry run: expected %v, got %v", want, preview)
	}
	if exported, err := store.ListEventsForExport(ctx, userID); err != nil || len(exported) != 1 {
		t.Fatalf("dry run must not write: got %d events, err %v", len(exported), err)
	}

	created, err := store.ImportEvents(ctx, userID, uuid.Nil, events, false)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(created, want) {
		t.Fatalf("import: expected %v, got %v", want, created)
	}
	again, err := store.ImportEvents(ctx, userID, uuid.Nil, events, false)
	if err != nil {
		t.Fatal(err)
	}
	if want := []bool{false, false, false, true}; !slices.Equal(again, want) {
		t.Fatalf("re-import: expected %v, got %v", want, again)
	}

	// Another user importing the same file gets their own events.
	otherID := registerUser(t, store, "other@example.com")
	forOther, err := store.ImportEvents(ctx, otherID, uuid.Nil, events, false)
	if err != nil {
		t.Fatal(err)
	}
	if want := []bool{true, false, true, true}; !slices.Equal(forOther, want) {
		t.Fatalf("other user: expected %v, got %v", want, forOther)
	}
}

func TestImportEventsAttachesOwnTagsAndReminders(t *testing.T) {
	ctx := context.Background()
	store := testutil.NewStore(t)
	userID := registerUser(t, store, "owner@example.com")
	otherID := registerUser(t, store, "other@example.com")
	mainTag, err := store.CreateTag(ctx, userID, "imported")
	if err != nil {
		t.Fatal(err)
	}
	extraTag, err := store.CreateTag(ctx, userID, "family")
	if err != nil {
		t.Fatal(err)
	}
	foreignTag, err := store.CreateTag(ctx, otherID, "foreign")
	if err != nil {
		t.Fatal(err)
	}

	future := time.Now().Add(48 * time.Hour).Truncate(time.Minute).UTC()
	upcoming := importedEvent("upcoming", "Upcoming", future)
	upcoming.TagIDs = []uuid.UUID{extraTag.ID, foreignTag.ID}
	upcoming.ReminderOffsets = []int{900, 900, 86400}

	if _, err := store.ImportEvents(ctx, userID, mainTag.ID, []db.ImportedEvent{upcoming}, false); err != nil {
		t.Fatal(err)
	}

	exported, err := store.ListEventsForExport(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(exported) != 1 || exported[0].UID != "upcoming" {
		t.Fatalf("unexpected export %+v", exported)
	}
	if got := exported[0]; got.MainTagID != mainTag.ID ||
		!slices.Equal(got.TagNames, []string{"imported", "family"}) ||
		!slices.Equal(got.ReminderOffsets, []int{900, 86400}) {
		t.Fatalf("unexpected tags or reminders %+v", got)
	}

	if _, err := store.ImportEvents(ctx, userID, foreignTag.ID, []db.ImportedEvent{importedEvent("x", "X", future)}, false); !errors.Is(err, db.ErrTagNotFound) {
		t.Fatalf("foreign main tag: expected ErrTagNotFound, got %v", err)
	}
}

func TestImportEventsMarksPassedRemindersDelivered(t *testing.T) {
	ctx := context.Background()
	store := testutil.NewStore(t)
	userID := registerUser(t, store, "owner@example.com")
	// The event has not ended, so its reminders are listed, but the
	// one-hour reminder's time has passed.
	startsAt := time.Now().Add(30 * time.Minute).Truncate(time.Minute).UTC()
	event := importedEvent("soon", "Soon", startsAt)
	event.ReminderOffsets = []int{3600, 0}
	if _, err := store.ImportEvents(ctx, userID, uuid.Nil, []db.ImportedEvent{event}, false); err != nil {
		t.Fatal(err)
	}
	reminders, err := store.ListReminders(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(reminders) != 2 || reminders[0].NotifiedAt == nil || reminders[1].NotifiedAt != nil {
		t.Fatalf("expected only the passed reminder delivered, got %+v", reminders)
	}
}

func TestListEventsForExportIsOwnEventsOnly(t *testing.T) {
	ctx := context.Background()
	store := testutil.NewStore(t)
	ownerID := registerUser(t, store, "owner@example.com")
	recipientID := registerUser(t, store, "recipient@example.com")
	startsAt := time.Now().Add(24 * time.Hour).Truncate(time.Minute).UTC()
	eventID, err := store.CreateEvent(ctx, ownerID, db.Event{
		Title: "Shared", StartsAt: startsAt, EndsAt: startsAt.Add(time.Hour), Kind: db.EventKindAttendable,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.ShareEvent(ctx, ownerID, eventID, recipientID, db.ShareFields{}); err != nil {
		t.Fatal(err)
	}

	owned, err := store.ListEventsForExport(ctx, ownerID)
	if err != nil {
		t.Fatal(err)
	}
	if len(owned) != 1 || owned[0].UID != eventID.String()+db.ExportUIDSuffix || !slices.Equal(owned[0].TagNames, []string{"general"}) {
		t.Fatalf("unexpected owner export %+v", owned)
	}
	shared, err := store.ListEventsForExport(ctx, recipientID)
	if err != nil {
		t.Fatal(err)
	}
	if len(shared) != 0 {
		t.Fatalf("recipient must not export shared events, got %+v", shared)
	}
}
