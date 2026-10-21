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

var seriesStart = time.Date(2026, 10, 13, 8, 0, 0, 0, time.UTC)

// weeklySeries is a weekly series of one-hour events. The store keeps the
// rule as given; the caller works out the RRULE and the end.
func weeklySeries(title string) db.Event {
	return db.Event{
		Title: title, StartsAt: seriesStart, EndsAt: seriesStart.Add(time.Hour), Kind: db.EventKindAttendable,
		RecurrenceRule: new("FREQ=WEEKLY;INTERVAL=1;BYDAY=TU"), RecurrenceTimeZone: new("Europe/Stockholm"),
	}
}

func week(n int) time.Time { return seriesStart.AddDate(0, 0, 7*n) }

func movedException(n int, by time.Duration) db.OccurrenceException {
	starts, ends := week(n).Add(by), week(n).Add(by+time.Hour)
	return db.OccurrenceException{OriginalStart: week(n), StartsAt: &starts, EndsAt: &ends}
}

func originals(exceptions []db.OccurrenceException) []time.Time {
	var result []time.Time
	for _, exception := range exceptions {
		result = append(result, exception.OriginalStart.UTC())
	}
	return result
}

func TestSeriesAndExceptionsAreStoredForTheOwner(t *testing.T) {
	ctx := context.Background()
	store := testutil.NewStore(t)
	ownerID := registerUser(t, store, "owner@example.com")
	strangerID := registerUser(t, store, "stranger@example.com")
	eventID, err := store.CreateEvent(ctx, ownerID, weeklySeries("Lecture"))
	if err != nil {
		t.Fatal(err)
	}

	skipped := true
	if err := store.SetOccurrenceException(ctx, ownerID, eventID,
		db.OccurrenceException{OriginalStart: week(1), IsSkipped: &skipped}, nil); err != nil {
		t.Fatal(err)
	}
	// A second exception for the same occurrence replaces the first.
	endsAt := week(3).Add(2 * time.Hour)
	if err := store.SetOccurrenceException(ctx, ownerID, eventID, movedException(1, time.Hour), &endsAt); err != nil {
		t.Fatal(err)
	}
	if err := store.SetOccurrenceException(ctx, ownerID, eventID,
		db.OccurrenceException{OriginalStart: week(2), IsCancelled: true}, &endsAt); err != nil {
		t.Fatal(err)
	}

	entry, err := store.GetOwnEvent(ctx, ownerID, eventID)
	if err != nil {
		t.Fatal(err)
	}
	if *entry.RecurrenceRule != "FREQ=WEEKLY;INTERVAL=1;BYDAY=TU" || *entry.RecurrenceTimeZone != "Europe/Stockholm" ||
		!entry.RecurrenceEndsAt.Equal(endsAt) || len(entry.Tags) != 1 {
		t.Fatalf("unexpected series %+v", entry)
	}
	if !slices.Equal(originals(entry.Exceptions), []time.Time{week(1), week(2)}) ||
		entry.Exceptions[0].IsSkipped != nil || !entry.Exceptions[0].StartsAt.Equal(week(1).Add(time.Hour)) ||
		!entry.Exceptions[1].IsCancelled {
		t.Fatalf("unexpected exceptions %+v", entry.Exceptions)
	}
	if event := entry.Event(); event.RecurrenceRule == nil || event.RecurrenceTimeZone == nil {
		t.Fatalf("Event() must keep the rule, got %+v", event)
	}

	// UpdateSeries replaces the exceptions; dropping the rule ends the series.
	plain := weeklySeries("Lecture")
	plain.RecurrenceRule, plain.RecurrenceTimeZone = nil, nil
	if err := store.UpdateSeries(ctx, ownerID, eventID, plain, nil); err != nil {
		t.Fatal(err)
	}
	if entry, _ := store.GetOwnEvent(ctx, ownerID, eventID); entry.RecurrenceRule != nil || len(entry.Exceptions) != 0 {
		t.Fatalf("expected a plain event, got %+v", entry)
	}
	if err := store.SetOccurrenceException(ctx, ownerID, eventID,
		db.OccurrenceException{OriginalStart: week(1), IsCancelled: true}, nil); !errors.Is(err, db.ErrEventNotFound) {
		t.Fatalf("exception on a plain event: expected ErrEventNotFound, got %v", err)
	}

	if _, err := store.GetOwnEvent(ctx, strangerID, eventID); !errors.Is(err, db.ErrEventNotFound) {
		t.Fatalf("stranger: expected ErrEventNotFound, got %v", err)
	}
}

func TestStrangersCannotChangeSeries(t *testing.T) {
	ctx := context.Background()
	store := testutil.NewStore(t)
	ownerID := registerUser(t, store, "owner@example.com")
	strangerID := registerUser(t, store, "stranger@example.com")
	eventID, err := store.CreateEvent(ctx, ownerID, weeklySeries("Lecture"))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SetOccurrenceException(ctx, strangerID, eventID,
		db.OccurrenceException{OriginalStart: week(1), IsCancelled: true}, nil); !errors.Is(err, db.ErrEventNotFound) {
		t.Fatalf("exception: expected ErrEventNotFound, got %v", err)
	}
	if err := store.EndSeries(ctx, strangerID, eventID, week(1), "FREQ=WEEKLY;COUNT=1", nil); !errors.Is(err, db.ErrEventNotFound) {
		t.Fatalf("end: expected ErrEventNotFound, got %v", err)
	}
	if _, err := store.SplitSeries(ctx, strangerID, eventID, week(1), "FREQ=WEEKLY;COUNT=1", nil,
		weeklySeries("Taken"), nil); !errors.Is(err, db.ErrEventNotFound) {
		t.Fatalf("split: expected ErrEventNotFound, got %v", err)
	}
	if entries, _ := store.ListTimeline(ctx, strangerID); len(entries) != 0 {
		t.Fatalf("a failed split must create nothing, got %+v", entries)
	}
	entry, err := store.GetOwnEvent(ctx, ownerID, eventID)
	if err != nil || *entry.RecurrenceRule != "FREQ=WEEKLY;INTERVAL=1;BYDAY=TU" || len(entry.Exceptions) != 0 {
		t.Fatalf("the series must be unchanged, got %+v %v", entry, err)
	}
}

func TestRecipientsSeeSeriesWithoutTheOwnersSkips(t *testing.T) {
	ctx := context.Background()
	store := testutil.NewStore(t)
	ownerID := registerUser(t, store, "owner@example.com")
	recipientID := registerUser(t, store, "recipient@example.com")
	eventID, err := store.CreateEvent(ctx, ownerID, weeklySeries("Lecture"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.ShareEvent(ctx, ownerID, eventID, recipientID, db.ShareFields{}); err != nil {
		t.Fatal(err)
	}
	skipped := true
	moved := movedException(1, time.Hour)
	moved.IsSkipped = &skipped
	if err := store.SetOccurrenceException(ctx, ownerID, eventID, moved, nil); err != nil {
		t.Fatal(err)
	}

	entries, err := store.ListTimeline(ctx, recipientID)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].RecurrenceRule == nil || len(entries[0].Exceptions) != 1 {
		t.Fatalf("expected the shared series with its exception, got %+v", entries)
	}
	if exception := entries[0].Exceptions[0]; exception.IsSkipped != nil || exception.StartsAt == nil {
		t.Fatalf("recipients see moves but never skips, got %+v", exception)
	}
	owned, err := store.ListTimeline(ctx, ownerID)
	if err != nil {
		t.Fatal(err)
	}
	if exception := owned[0].Exceptions[0]; exception.IsSkipped == nil || !*exception.IsSkipped {
		t.Fatalf("the owner sees their skip, got %+v", exception)
	}
}

func TestEndAndSplitSeries(t *testing.T) {
	ctx := context.Background()
	store := testutil.NewStore(t)
	ownerID := registerUser(t, store, "owner@example.com")
	recipientID := registerUser(t, store, "recipient@example.com")
	family, err := store.CreateTag(ctx, ownerID, "family")
	if err != nil {
		t.Fatal(err)
	}
	eventID, err := store.CreateEvent(ctx, ownerID, weeklySeries("Lecture"))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.AddTagToItem(ctx, ownerID, eventID, family.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ShareEvent(ctx, ownerID, eventID, recipientID, db.ShareFields{Title: new("Lecture for you")}); err != nil {
		t.Fatal(err)
	}
	for userID, offset := range map[uuid.UUID]int{ownerID: 900, recipientID: 3600} {
		if _, err := store.CreateReminder(ctx, userID, eventID, offset); err != nil {
			t.Fatal(err)
		}
	}
	for n := 1; n <= 4; n++ {
		if err := store.SetOccurrenceException(ctx, ownerID, eventID,
			db.OccurrenceException{OriginalStart: week(n), IsCancelled: true}, nil); err != nil {
			t.Fatal(err)
		}
	}

	// Ending before week 4 drops the exceptions from it on.
	endsAt := week(3).Add(time.Hour)
	if err := store.EndSeries(ctx, ownerID, eventID, week(4), "FREQ=WEEKLY;INTERVAL=1;BYDAY=TU;COUNT=4", &endsAt); err != nil {
		t.Fatal(err)
	}
	entry, err := store.GetOwnEvent(ctx, ownerID, eventID)
	if err != nil {
		t.Fatal(err)
	}
	if *entry.RecurrenceRule != "FREQ=WEEKLY;INTERVAL=1;BYDAY=TU;COUNT=4" || !entry.RecurrenceEndsAt.Equal(endsAt) ||
		!slices.Equal(originals(entry.Exceptions), []time.Time{week(1), week(2), week(3)}) {
		t.Fatalf("unexpected ended series %+v", entry)
	}

	// Splitting at week 2 moves to a new series at 18:00, carrying week 3's
	// exception re-keyed for it.
	newSeries := weeklySeries("Evening lecture")
	newSeries.StartsAt, newSeries.EndsAt = week(2).Add(10*time.Hour), week(2).Add(11*time.Hour)
	carried := db.OccurrenceException{OriginalStart: week(3).Add(10 * time.Hour), IsCancelled: true}
	oldEnd := week(1).Add(time.Hour)
	newID, err := store.SplitSeries(ctx, ownerID, eventID, week(2), "FREQ=WEEKLY;INTERVAL=1;BYDAY=TU;COUNT=2", &oldEnd,
		newSeries, []db.OccurrenceException{carried})
	if err != nil {
		t.Fatal(err)
	}

	old, err := store.GetOwnEvent(ctx, ownerID, eventID)
	if err != nil {
		t.Fatal(err)
	}
	if *old.RecurrenceRule != "FREQ=WEEKLY;INTERVAL=1;BYDAY=TU;COUNT=2" ||
		!slices.Equal(originals(old.Exceptions), []time.Time{week(1)}) {
		t.Fatalf("unexpected old series %+v", old)
	}
	created, err := store.GetOwnEvent(ctx, ownerID, newID)
	if err != nil {
		t.Fatal(err)
	}
	var tagNames []string
	for _, tag := range created.Tags {
		tagNames = append(tagNames, tag.Name)
	}
	slices.Sort(tagNames)
	if created.Title != "Evening lecture" || !created.StartsAt.Equal(newSeries.StartsAt) ||
		!slices.Equal(tagNames, []string{"family", "general"}) || *created.MainTagID != *old.MainTagID ||
		!slices.Equal(originals(created.Exceptions), []time.Time{carried.OriginalStart}) || created.ShareCount != 1 {
		t.Fatalf("unexpected new series %+v", created)
	}

	shares, err := store.ListEventShares(ctx, ownerID, newID)
	if err != nil {
		t.Fatal(err)
	}
	if len(shares) != 1 || shares[0].RecipientUserID != recipientID || *shares[0].Title != "Lecture for you" {
		t.Fatalf("expected the share to carry over, got %+v", shares)
	}
	for userID, want := range map[uuid.UUID]int{ownerID: 900, recipientID: 3600} {
		reminders, err := store.ListReminders(ctx, userID)
		if err != nil {
			t.Fatal(err)
		}
		if !slices.ContainsFunc(reminders, func(reminder db.Reminder) bool {
			return reminder.EventID == newID && reminder.OffsetSeconds == want
		}) {
			t.Fatalf("expected the %d s reminder to carry over, got %+v", want, reminders)
		}
	}
}

func TestSeriesRuleNeedsATimeZone(t *testing.T) {
	store := testutil.NewStore(t)
	ownerID := registerUser(t, store, "owner@example.com")
	event := weeklySeries("Lecture")
	event.RecurrenceTimeZone = nil
	if _, err := store.CreateEvent(context.Background(), ownerID, event); err == nil {
		t.Fatal("expected the database to reject a rule without a time zone")
	}
}
