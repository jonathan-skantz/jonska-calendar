package subscription

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/jonathan-skantz/calendar/backend/db"
	"github.com/jonathan-skantz/calendar/backend/internal/testutil"
)

const schoolFeed = "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//Test//EN\r\nX-WR-CALNAME:School\r\n" +
	"BEGIN:VEVENT\r\nUID:lecture\r\nSUMMARY:Lecture\r\nDTSTART:20261026T080000Z\r\nDTEND:20261026T100000Z\r\n" +
	"X-JONSKA-COMMUTE-BEFORE:PT30M\r\nX-JONSKA-SKIPPED:TRUE\r\nCATEGORIES:Work\r\n" +
	"BEGIN:VALARM\r\nACTION:DISPLAY\r\nTRIGGER:-PT15M\r\nEND:VALARM\r\nEND:VEVENT\r\n" +
	"BEGIN:VEVENT\r\nUID:broken\r\nSUMMARY:Backwards\r\nDTSTART:20261026T080000Z\r\nDTEND:20261026T070000Z\r\nEND:VEVENT\r\n" +
	"END:VCALENDAR\r\n"

// feedServer serves body until it is told to fail.
func feedServer(t *testing.T, body string) (*httptest.Server, *atomic.Bool) {
	t.Helper()
	failing := &atomic.Bool{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if failing.Load() {
			http.Error(w, "gone", http.StatusGone)
			return
		}
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	return server, failing
}

func send(t *testing.T, handle http.HandlerFunc, subscriptionID string, body any) *httptest.ResponseRecorder {
	t.Helper()
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/subscriptions", bytes.NewReader(encoded))
	request.SetPathValue("subscriptionID", subscriptionID)
	response := httptest.NewRecorder()
	handle(response, request)
	return response
}

func decodeSubscription(t *testing.T, response *httptest.ResponseRecorder, status int) subscriptionResponse {
	t.Helper()
	if response.Code != status {
		t.Fatalf("expected status %d, got %d: %s", status, response.Code, response.Body.String())
	}
	var subscription subscriptionResponse
	if err := json.NewDecoder(response.Body).Decode(&subscription); err != nil {
		t.Fatal(err)
	}
	return subscription
}

func TestSubscriptionRoutesRequireAuthentication(t *testing.T) {
	handler := New(nil, func(*http.Request) (uuid.UUID, error) { return uuid.Nil, errors.New("not signed in") }, NewClient(true))
	for name, handle := range map[string]http.HandlerFunc{
		"list": handler.List, "create": handler.Create, "update": handler.Update,
		"refresh": handler.Refresh, "delete": handler.Delete,
	} {
		if code := send(t, handle, uuid.NewString(), nil).Code; code != http.StatusUnauthorized {
			t.Errorf("%s: expected %d, got %d", name, http.StatusUnauthorized, code)
		}
	}
}

func TestCreateRejectsBadRequests(t *testing.T) {
	server, _ := feedServer(t, "not a calendar")
	handler := New(nil, testutil.SignedInAs(uuid.New()), NewClient(true))
	for name, body := range map[string]map[string]any{
		"bad scheme":     {"url": "ftp://example.com/a.ics", "time_zone": "UTC"},
		"no time zone":   {"url": server.URL},
		"bad color":      {"url": server.URL, "time_zone": "UTC", "color": "red"},
		"long name":      {"url": server.URL, "time_zone": "UTC", "name": strings.Repeat("x", maxNameLength+1)},
		"not a calendar": {"url": server.URL, "time_zone": "UTC"},
	} {
		if code := send(t, handler.Create, "", body).Code; code != http.StatusBadRequest {
			t.Errorf("%s: expected %d, got %d", name, http.StatusBadRequest, code)
		}
	}
}

func TestSubscribeRefreshAndUnsubscribe(t *testing.T) {
	ctx := context.Background()
	store := testutil.NewStore(t)
	userID, err := store.Register(ctx, "user@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	otherID, err := store.Register(ctx, "other@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateTag(ctx, userID, "Work"); err != nil {
		t.Fatal(err)
	}
	server, failing := feedServer(t, schoolFeed)
	handler := New(store, testutil.SignedInAs(userID), NewClient(true))
	other := New(store, testutil.SignedInAs(otherID), NewClient(true))

	created := decodeSubscription(t, send(t, handler.Create, "",
		map[string]any{"url": server.URL, "time_zone": "Europe/Stockholm"}), http.StatusCreated)
	if created.Name != "School" || created.Color != defaultColor || created.LastError != nil {
		t.Fatalf("unexpected subscription %+v", created)
	}
	if code := send(t, handler.Create, "", map[string]any{"url": server.URL, "time_zone": "UTC"}).Code; code != http.StatusConflict {
		t.Fatalf("expected %d for a second subscription to the URL, got %d", http.StatusConflict, code)
	}

	from := time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC)
	entries, err := store.ListTimeline(ctx, userID, from, from.AddDate(0, 0, 1))
	if err != nil {
		t.Fatal(err)
	}
	// The event that ends before it starts is left out; the publisher's
	// commute, skip, categories and alarms are not taken over.
	if len(entries) != 1 {
		t.Fatalf("expected one subscribed event, got %+v", entries)
	}
	lecture := entries[0]
	if lecture.Subscription == nil || lecture.Subscription.ID != created.ID || lecture.CommuteBeforeSeconds != 0 ||
		lecture.IsSkipped || len(lecture.Tags) != 0 {
		t.Fatalf("unexpected subscribed event %+v", lecture)
	}
	reminders, err := store.ListReminders(ctx, userID)
	if err != nil || len(reminders) != 0 {
		t.Fatalf("expected no reminders from the feed, got %+v, %v", reminders, err)
	}

	id := created.ID.String()
	if code := send(t, other.Refresh, id, nil).Code; code != http.StatusNotFound {
		t.Fatalf("expected %d for another user's refresh, got %d", http.StatusNotFound, code)
	}
	if code := send(t, other.Update, id, map[string]any{"name": "Mine"}).Code; code != http.StatusNotFound {
		t.Fatalf("expected %d for another user's update, got %d", http.StatusNotFound, code)
	}

	renamed := decodeSubscription(t, send(t, handler.Update, id, map[string]any{"name": " Uni ", "color": "#112233"}), http.StatusOK)
	if renamed.Name != "Uni" || renamed.Color != "#112233" {
		t.Fatalf("unexpected update %+v", renamed)
	}

	// A failed refresh is reported and keeps the events.
	failing.Store(true)
	response := send(t, handler.Refresh, id, nil)
	if response.Code != http.StatusBadGateway || !strings.Contains(response.Body.String(), "410") {
		t.Fatalf("expected %d with the status, got %d: %s", http.StatusBadGateway, response.Code, response.Body.String())
	}
	subscription, err := store.GetSubscription(ctx, userID, created.ID)
	if err != nil || subscription.LastError == nil {
		t.Fatalf("expected the error recorded, got %+v, %v", subscription, err)
	}
	if entries, _ := store.ListTimeline(ctx, userID, from, from.AddDate(0, 0, 1)); len(entries) != 1 {
		t.Fatalf("expected the last events kept, got %+v", entries)
	}
	failing.Store(false)
	if refreshed := decodeSubscription(t, send(t, handler.Refresh, id, nil), http.StatusOK); refreshed.LastError != nil {
		t.Fatalf("expected the error cleared, got %+v", refreshed)
	}

	if code := send(t, other.Delete, id, nil).Code; code != http.StatusNotFound {
		t.Fatalf("expected %d for another user's delete, got %d", http.StatusNotFound, code)
	}
	if code := send(t, handler.Delete, id, nil).Code; code != http.StatusNoContent {
		t.Fatalf("expected %d, got %d", http.StatusNoContent, code)
	}
	if entries, _ := store.ListTimeline(ctx, userID, from, from.AddDate(0, 0, 1)); len(entries) != 0 {
		t.Fatalf("expected the events gone, got %+v", entries)
	}
}

func TestRefreshStaleRefreshesOnlyStaleSubscriptions(t *testing.T) {
	ctx := context.Background()
	store := testutil.NewStore(t)
	userID, err := store.Register(ctx, "user@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	server, failing := feedServer(t, schoolFeed)
	refresher := &Refresher{Store: store, Client: NewClient(true)}
	subscription, err := store.CreateSubscription(ctx, db.Subscription{
		UserID: userID, Name: "School", URL: server.URL, Color: defaultColor, TimeZone: "UTC",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Just created, so not stale: a failing feed is not even asked.
	failing.Store(true)
	refresher.refreshStale(ctx)
	if fresh, _ := store.GetSubscription(ctx, userID, subscription.ID); fresh.LastError != nil {
		t.Fatalf("expected a fresh subscription to be left alone, got %+v", fresh)
	}
}
