package subscription

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"github.com/jonathan-skantz/calendar/backend/db"
	"github.com/jonathan-skantz/calendar/backend/internal/ical"
	"github.com/jonathan-skantz/calendar/backend/internal/recurrence"
	"github.com/jonathan-skantz/calendar/backend/internal/testutil"
	"github.com/jonathan-skantz/calendar/backend/internal/timeline"
)

// One user publishes a feed and another subscribes to it, through the same
// handlers the server routes to.
func TestSubscribingToAPublishedFeed(t *testing.T) {
	ctx := context.Background()
	store := testutil.NewStore(t)
	publisherID, err := store.Register(ctx, "publisher@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	subscriberID, err := store.Register(ctx, "subscriber@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	stockholm, err := time.LoadLocation("Europe/Stockholm")
	if err != nil {
		t.Fatal(err)
	}
	start := time.Date(2026, 10, 20, 18, 0, 0, 0, stockholm)
	rule := recurrence.Rule{Frequency: recurrence.Weekly, Interval: 1}.RRULE(start, false)
	practiceID, err := store.CreateEvent(ctx, publisherID, db.Event{
		Title: "Practice", StartsAt: start, EndsAt: start.Add(90 * time.Minute), Kind: db.EventKindAttendable,
		CommuteBeforeSeconds: 1200, GatheringSeconds: 900, RecurrenceRule: &rule, RecurrenceTimeZone: new("Europe/Stockholm"),
	})
	if err != nil {
		t.Fatal(err)
	}
	// The second practice is cancelled.
	if err := store.SetOccurrenceException(ctx, publisherID, practiceID,
		db.OccurrenceException{OriginalStart: start.AddDate(0, 0, 7), IsCancelled: true}, nil); err != nil {
		t.Fatal(err)
	}
	feed, err := store.CreateFeed(ctx, publisherID, db.FeedFields{Name: "Club"}, "club-token", "Europe/Stockholm")
	if err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/feeds/{token}/calendar.ics", ical.New(store, nil).ServeFeed)
	server := httptest.NewServer(mux)
	defer server.Close()

	handler := New(store, testutil.SignedInAs(subscriberID), NewClient(true))
	subscription := decodeSubscription(t, send(t, handler.Create, "", map[string]any{
		"url":       server.URL + "/api/feeds/" + feed.Token + "/calendar.ics",
		"time_zone": "Europe/Stockholm",
	}), http.StatusCreated)
	if subscription.Name != "Club" {
		t.Fatalf("expected the feed's name, got %+v", subscription)
	}

	from := start.AddDate(0, 0, -1)
	to := from.AddDate(0, 0, 21)
	entries, err := store.ListTimeline(ctx, subscriberID, from, to)
	if err != nil {
		t.Fatal(err)
	}
	items, err := timeline.ExpandTimeline(entries, from, to)
	if err != nil {
		t.Fatal(err)
	}
	var starts []string
	for _, item := range items {
		// The file starts the practice at its gathering; the subscriber
		// reads its own start and the gathering back.
		if item.Subscription == nil || item.CommuteBeforeSeconds != 0 || item.GatheringSeconds != 900 {
			t.Fatalf("expected a subscribed occurrence with gathering and without commute, got %+v", item)
		}
		starts = append(starts, item.StartsAt.In(stockholm).Format("2006-01-02 15:04"))
	}
	// 18:00 stays 18:00 across the end of summer time, and the cancelled
	// practice is left out.
	if want := []string{"2026-10-20 18:00", "2026-11-03 18:00"}; !slices.Equal(starts, want) {
		t.Fatalf("expected %v, got %v", want, starts)
	}

	// The feed's UID keeps the subscriber's customization across refreshes.
	if err := store.CustomizeSubscribedEvent(ctx, subscriberID, items[0].ID, db.Customization{Note: new("Bring shoes")}); err != nil {
		t.Fatal(err)
	}
	stored, err := store.GetSubscription(ctx, subscriberID, subscription.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := (&Refresher{Store: store, Client: NewClient(true)}).Refresh(ctx, stored); err != nil {
		t.Fatal(err)
	}
	entries, err = store.ListTimeline(ctx, subscriberID, from, to)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].ID == items[0].ID || entries[0].Customization == nil ||
		*entries[0].Customization.Note != "Bring shoes" {
		t.Fatalf("expected the customization kept after the refresh, got %+v", entries)
	}
}
