package timeline

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/jonathan-skantz/calendar/backend/db"
)

func receivedEvent(title string, startsAt time.Time, customization *db.Customization) db.TimelineEntry {
	endsAt := startsAt.Add(time.Hour)
	return db.TimelineEntry{
		ID: uuid.New(), Type: "event", Title: title, Location: new("Hall"), IsShared: true,
		StartsAt: &startsAt, EndsAt: &endsAt, Customization: customization,
	}
}

func TestExpandTimelineAppliesCustomizations(t *testing.T) {
	from := time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 0, 7)
	shifted := &db.Customization{
		Title: new("My lecture"), Description: new("Mine"), Note: new("Zoom"),
		StartShiftSeconds: 15 * 60, EndShiftSeconds: -30 * 60,
	}
	lecture := receivedEvent("Lecture", from.Add(9*time.Hour), shifted)
	// Ends 15 minutes before the range; the shift moves it in.
	early := receivedEvent("Early", from.Add(-75*time.Minute), &db.Customization{StartShiftSeconds: 3600, EndShiftSeconds: 3600})
	// Outside the range and not shifted, as the store reads received events
	// around it.
	outside := receivedEvent("Outside", from.Add(-2*time.Hour), nil)
	allDay := receivedEvent("Holiday", from.AddDate(0, 0, 1), &db.Customization{Title: new("Day off"), StartShiftSeconds: 3600})
	allDay.IsAllDay = true
	// The owner shortened the event so the shifts would turn it inside out.
	inverted := receivedEvent("Short", from.Add(30*time.Hour), &db.Customization{StartShiftSeconds: 3600, EndShiftSeconds: -3600})

	items, err := ExpandTimeline([]db.TimelineEntry{lecture, early, outside, allDay, inverted}, from, to)
	if err != nil {
		t.Fatal(err)
	}
	byTitle := map[string]TimelineItem{}
	for _, item := range items {
		byTitle[item.Title] = item
	}
	if len(items) != 4 {
		t.Fatalf("expected the event outside the range left out, got %+v", byTitle)
	}

	got := byTitle["My lecture"]
	if *got.Description != "Mine" || *got.Location != "Hall" ||
		!got.StartsAt.Equal(from.Add(9*time.Hour+15*time.Minute)) || !got.EndsAt.Equal(from.Add(9*time.Hour+30*time.Minute)) {
		t.Fatalf("expected the customized values, got %+v", got)
	}
	if got.Original == nil || got.Original.Title != "Lecture" || got.Original.Description != nil ||
		!got.Original.StartsAt.Equal(*lecture.StartsAt) || !got.Original.EndsAt.Equal(*lecture.EndsAt) {
		t.Fatalf("expected the received values kept, got %+v", got.Original)
	}
	if got := byTitle["Early"]; !got.StartsAt.Equal(from.Add(-15 * time.Minute)) {
		t.Fatalf("expected the early event shifted into the range, got %+v", got)
	}
	if got := byTitle["Day off"]; !got.StartsAt.Equal(*allDay.StartsAt) {
		t.Fatalf("expected an all-day event not shifted, got %+v", got)
	}
	if got := byTitle["Short"]; !got.StartsAt.Equal(*inverted.StartsAt) || !got.EndsAt.Equal(*inverted.EndsAt) {
		t.Fatalf("expected shifts that turn the event inside out left out, got %+v", got)
	}
}

func TestExpandTimelineShiftsEachOccurrence(t *testing.T) {
	from := time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC)
	rule, zone := "FREQ=DAILY", "UTC"
	// Daily at 23:30 from the day before the range; the shift moves each
	// one past midnight.
	series := receivedEvent("Night shift", from.Add(-30*time.Minute), &db.Customization{StartShiftSeconds: 3600, EndShiftSeconds: 3600})
	series.RecurrenceRule, series.RecurrenceTimeZone = &rule, &zone

	items, err := ExpandTimeline([]db.TimelineEntry{series}, from, from.AddDate(0, 0, 2))
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("expected the occurrences that start in the range once shifted, got %+v", items)
	}
	for day, item := range items {
		original := from.AddDate(0, 0, day).Add(-30 * time.Minute)
		if !item.OccurrenceStart.Equal(original) || !item.StartsAt.Equal(original.Add(time.Hour)) ||
			!item.Original.StartsAt.Equal(original) {
			t.Fatalf("expected occurrence %d shifted and identified by its own start, got %+v", day, item)
		}
	}
}

func TestReceivedEventsShiftedIntoTheRangeAreListed(t *testing.T) {
	store, ownerID := setupTestStore(t)
	ctx := context.Background()
	recipientID, err := store.Register(ctx, "recipient@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	from := time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC)
	startsAt := from.Add(-20 * time.Hour)
	eventID, err := store.CreateEvent(ctx, ownerID, db.Event{
		Title: "Late", StartsAt: startsAt, EndsAt: startsAt.Add(time.Hour), Kind: db.EventKindAttendable,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.ShareEvent(ctx, ownerID, eventID, recipientID, db.ShareFields{}); err != nil {
		t.Fatal(err)
	}
	if err := store.CustomizeSharedEvent(ctx, recipientID, eventID, db.Customization{
		StartShiftSeconds: db.MaxShiftSeconds, EndShiftSeconds: db.MaxShiftSeconds,
	}); err != nil {
		t.Fatal(err)
	}

	currentUser := recipientID
	handler := New(store, func(*http.Request) (uuid.UUID, error) { return currentUser, nil })
	items := listRange(t, handler, from, from.AddDate(0, 0, 7))
	if len(items) != 1 || !items[0].StartsAt.Equal(startsAt.Add(24*time.Hour)) {
		t.Fatalf("expected the event shifted a day into the range, got %+v", items)
	}
	currentUser = ownerID
	if items := listRange(t, handler, from, from.AddDate(0, 0, 7)); len(items) != 0 {
		t.Fatalf("expected the owner's event to stay where it is, got %+v", items)
	}
}
