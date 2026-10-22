package timeline

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

type listedReceived struct {
	Title         string            `json:"title"`
	Location      *string           `json:"location"`
	StartsAt      time.Time         `json:"starts_at"`
	Customization *db.Customization `json:"customization"`
	Original      *ReceivedValues   `json:"original"`
}

func listReceived(t *testing.T, handler *Handler, from, to time.Time) []listedReceived {
	t.Helper()
	response := httptest.NewRecorder()
	handler.ListTimeline(response, httptest.NewRequest(http.MethodGet,
		"/api/timeline?from="+from.UTC().Format(time.RFC3339)+"&to="+to.UTC().Format(time.RFC3339), nil))
	if response.Code != http.StatusOK {
		t.Fatalf("list: expected %d, got %d: %s", http.StatusOK, response.Code, response.Body.String())
	}
	var items []listedReceived
	if err := json.NewDecoder(response.Body).Decode(&items); err != nil {
		t.Fatal(err)
	}
	return items
}

func sendCustomization(fn http.HandlerFunc, method string, eventID uuid.UUID, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, "/api/events/"+eventID.String()+"/customization", bytes.NewBufferString(body))
	request.SetPathValue("eventID", eventID.String())
	response := httptest.NewRecorder()
	fn(response, request)
	return response
}

func TestRecipientCustomizesASharedEvent(t *testing.T) {
	store, ownerID := setupTestStore(t)
	ctx := context.Background()
	recipientID, err := store.Register(ctx, "recipient@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	startsAt := time.Date(2026, 10, 20, 9, 0, 0, 0, time.UTC)
	eventID, err := store.CreateEvent(ctx, ownerID, db.Event{
		Title: "Lecture", Location: new("Hall"), StartsAt: startsAt, EndsAt: startsAt.Add(2 * time.Hour),
		Kind: db.EventKindAttendable,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.ShareEvent(ctx, ownerID, eventID, recipientID, db.ShareFields{Title: new("Lecture for you")}); err != nil {
		t.Fatal(err)
	}
	currentUser := recipientID
	handler := New(store, func(*http.Request) (uuid.UUID, error) { return currentUser, nil })
	from, to := startsAt.Truncate(24*time.Hour), startsAt.Truncate(24*time.Hour).AddDate(0, 0, 1)

	for _, body := range []string{`{"start_shift_seconds": 30}`, `{"end_shift_seconds": 90000}`, `not json`} {
		if response := sendCustomization(handler.CustomizeSharedEvent, http.MethodPut, eventID, body); response.Code != http.StatusBadRequest {
			t.Fatalf("%s: expected %d, got %d", body, http.StatusBadRequest, response.Code)
		}
	}
	body := `{"title": " My lecture ", "location": "  ", "note": "Zoom link", "start_shift_seconds": 900}`
	if response := sendCustomization(handler.CustomizeSharedEvent, http.MethodPut, eventID, body); response.Code != http.StatusNoContent {
		t.Fatalf("customize: expected %d, got %d: %s", http.StatusNoContent, response.Code, response.Body.String())
	}

	items := listReceived(t, handler, from, to)
	if len(items) != 1 {
		t.Fatalf("expected one event, got %+v", items)
	}
	got := items[0]
	if got.Title != "My lecture" || *got.Location != "Hall" || !got.StartsAt.Equal(startsAt.Add(15*time.Minute)) ||
		got.Customization == nil || *got.Customization.Note != "Zoom link" || got.Customization.Location != nil {
		t.Fatalf("expected the trimmed customization applied, got %+v", got)
	}
	// The original is what the owner sent this recipient, not their own title.
	if got.Original == nil || got.Original.Title != "Lecture for you" || !got.Original.StartsAt.Equal(startsAt) {
		t.Fatalf("expected the received values as the original, got %+v", got.Original)
	}

	strangerID, err := store.Register(ctx, "stranger@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	for _, userID := range []uuid.UUID{ownerID, strangerID} {
		currentUser = userID
		if response := sendCustomization(handler.CustomizeSharedEvent, http.MethodPut, eventID, `{}`); response.Code != http.StatusNotFound {
			t.Fatalf("expected only the recipient to customize, got %d", response.Code)
		}
		if response := sendCustomization(handler.UncustomizeSharedEvent, http.MethodDelete, eventID, ""); response.Code != http.StatusNotFound {
			t.Fatalf("expected only the recipient to remove a customization, got %d", response.Code)
		}
	}

	currentUser = recipientID
	if response := sendCustomization(handler.UncustomizeSharedEvent, http.MethodDelete, eventID, ""); response.Code != http.StatusNoContent {
		t.Fatalf("uncustomize: expected %d, got %d", http.StatusNoContent, response.Code)
	}
	if items := listReceived(t, handler, from, to); len(items) != 1 || items[0].Title != "Lecture for you" ||
		items[0].Customization != nil || items[0].Original != nil {
		t.Fatalf("expected the received event back, got %+v", items)
	}
}

func TestCustomizingASubscribedEventNeedsItsUID(t *testing.T) {
	store, userID := setupTestStore(t)
	ctx := context.Background()
	startsAt := time.Date(2026, 10, 20, 9, 0, 0, 0, time.UTC)
	event := func(title string, uid *string) db.SubscribedEvent {
		return db.SubscribedEvent{UID: uid, Event: db.Event{
			Title: title, StartsAt: startsAt, EndsAt: startsAt.Add(time.Hour), Kind: db.EventKindAttendable,
		}}
	}
	if _, err := store.CreateSubscription(ctx, db.Subscription{
		UserID: userID, Name: "School", URL: "https://example.com/school.ics", Color: "#336699", TimeZone: "UTC",
	}, []db.SubscribedEvent{event("Lecture", new("lecture@school")), event("No UID", nil)}); err != nil {
		t.Fatal(err)
	}
	handler := New(store, func(*http.Request) (uuid.UUID, error) { return userID, nil })
	from, to := startsAt.Truncate(24*time.Hour), startsAt.Truncate(24*time.Hour).AddDate(0, 0, 1)
	entries, err := store.ListTimeline(ctx, userID, from, to)
	if err != nil || len(entries) != 2 {
		t.Fatalf("expected two events, got %+v, %v", entries, err)
	}
	ids := map[string]uuid.UUID{}
	for _, entry := range entries {
		ids[entry.Title] = entry.ID
	}

	if response := sendCustomization(handler.CustomizeSubscribedEvent, http.MethodPut, ids["No UID"], `{"title": "Mine"}`); response.Code != http.StatusConflict {
		t.Fatalf("expected an event without a UID to be refused, got %d", response.Code)
	}
	if response := sendCustomization(handler.CustomizeSubscribedEvent, http.MethodPut, uuid.New(), `{"title": "Mine"}`); response.Code != http.StatusNotFound {
		t.Fatalf("expected an unknown event to be not found, got %d", response.Code)
	}
	if response := sendCustomization(handler.CustomizeSubscribedEvent, http.MethodPut, ids["Lecture"], `{"title": "Mine"}`); response.Code != http.StatusNoContent {
		t.Fatalf("customize: expected %d, got %d: %s", http.StatusNoContent, response.Code, response.Body.String())
	}
	titles := map[string]bool{}
	for _, item := range listReceived(t, handler, from, to) {
		titles[item.Title] = true
	}
	if !titles["Mine"] || !titles["No UID"] {
		t.Fatalf("expected the subscribed event customized, got %v", titles)
	}
	if response := sendCustomization(handler.UncustomizeSubscribedEvent, http.MethodDelete, ids["Lecture"], ""); response.Code != http.StatusNoContent {
		t.Fatalf("uncustomize: expected %d, got %d", http.StatusNoContent, response.Code)
	}
}
