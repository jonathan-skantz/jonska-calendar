package ical

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/jonathan-skantz/calendar/backend/db"
	"github.com/jonathan-skantz/calendar/backend/internal/testutil"
)

func feedRequestTo(t *testing.T, handle http.HandlerFunc, method, feedID string, body map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(method, "/api/feeds", bytes.NewReader(encoded))
	request.SetPathValue("feedID", feedID)
	response := httptest.NewRecorder()
	handle(response, request)
	return response
}

func decodeFeed(t *testing.T, response *httptest.ResponseRecorder, status int) feedResponse {
	t.Helper()
	if response.Code != status {
		t.Fatalf("expected status %d, got %d: %s", status, response.Code, response.Body.String())
	}
	var feed feedResponse
	if err := json.NewDecoder(response.Body).Decode(&feed); err != nil {
		t.Fatal(err)
	}
	return feed
}

func serveFeed(handler *Handler, path string) *httptest.ResponseRecorder {
	token := strings.TrimSuffix(strings.TrimPrefix(path, "/api/feeds/"), "/calendar.ics")
	request := httptest.NewRequest(http.MethodGet, path, nil)
	request.SetPathValue("token", token)
	response := httptest.NewRecorder()
	handler.ServeFeed(response, request)
	return response
}

func TestFeedManagementRequiresAuthentication(t *testing.T) {
	handler := New(nil, func(*http.Request) (uuid.UUID, error) { return uuid.Nil, errors.New("not signed in") })
	for name, handle := range map[string]http.HandlerFunc{
		"list": handler.ListFeeds, "create": handler.CreateFeed, "update": handler.UpdateFeed,
		"reset": handler.ResetFeedToken, "delete": handler.DeleteFeed,
	} {
		if code := feedRequestTo(t, handle, http.MethodPost, uuid.NewString(), nil).Code; code != http.StatusUnauthorized {
			t.Errorf("%s: expected %d, got %d", name, http.StatusUnauthorized, code)
		}
	}
}

func TestCreateFeedRejectsBadRequests(t *testing.T) {
	handler := New(nil, testutil.SignedInAs(uuid.New()))
	for name, body := range map[string]map[string]any{
		"blank name":     {"name": "  ", "time_zone": "UTC"},
		"long name":      {"name": strings.Repeat("x", maxFeedNameLength+1), "time_zone": "UTC"},
		"no time zone":   {"name": "All"},
		"local zone":     {"name": "All", "time_zone": "Local"},
		"wrong tag type": {"name": "All", "time_zone": "UTC", "tag_id": 4},
	} {
		if code := feedRequestTo(t, handler.CreateFeed, http.MethodPost, "", body).Code; code != http.StatusBadRequest {
			t.Errorf("%s: expected %d, got %d", name, http.StatusBadRequest, code)
		}
	}
}

func TestFeedServesOwnEventsWithoutASession(t *testing.T) {
	ctx := context.Background()
	store := testutil.NewStore(t)
	ownerID, err := store.Register(ctx, "owner@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	otherID, err := store.Register(ctx, "other@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	startsAt := time.Date(2026, 10, 24, 10, 0, 0, 0, time.UTC)
	eventID, err := store.CreateEvent(ctx, ownerID, db.Event{
		Title: "Football", StartsAt: startsAt, EndsAt: startsAt.Add(time.Hour), Kind: db.EventKindAttendable,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateReminder(ctx, ownerID, eventID, 900); err != nil {
		t.Fatal(err)
	}
	owner := New(store, testutil.SignedInAs(ownerID))

	feed := decodeFeed(t, feedRequestTo(t, owner.CreateFeed, http.MethodPost, "",
		map[string]any{"name": " Family ", "time_zone": "Europe/Stockholm"}), http.StatusCreated)
	if feed.Name != "Family" || feed.IncludeAlarms || !strings.HasSuffix(feed.Path, "/calendar.ics") {
		t.Fatalf("unexpected feed %+v", feed)
	}

	// The feed is served to a request without any session.
	anonymous := New(store, func(*http.Request) (uuid.UUID, error) { return uuid.Nil, errors.New("not signed in") })
	response := serveFeed(anonymous, feed.Path)
	if response.Code != http.StatusOK || !strings.HasPrefix(response.Header().Get("Content-Type"), "text/calendar") {
		t.Fatalf("expected a calendar, got %d: %s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	if !strings.Contains(body, "X-WR-CALNAME:Family") || !strings.Contains(body, "SUMMARY:Football") || strings.Contains(body, "BEGIN:VALARM") {
		t.Fatalf("expected the named feed without alarms, got:\n%s", body)
	}
	parsed, err := Parse(strings.NewReader(body), time.UTC)
	if err != nil || len(parsed.Events) != 1 {
		t.Fatalf("expected the feed to parse back to one event, got %+v, %v", parsed, err)
	}

	// Another user cannot change or delete the feed.
	other := New(store, testutil.SignedInAs(otherID))
	if code := feedRequestTo(t, other.UpdateFeed, http.MethodPatch, feed.ID.String(), map[string]any{"name": "Mine"}).Code; code != http.StatusNotFound {
		t.Fatalf("expected %d for another user's feed, got %d", http.StatusNotFound, code)
	}
	if code := feedRequestTo(t, other.DeleteFeed, http.MethodDelete, feed.ID.String(), nil).Code; code != http.StatusNotFound {
		t.Fatalf("expected %d for another user's feed, got %d", http.StatusNotFound, code)
	}

	updated := decodeFeed(t, feedRequestTo(t, owner.UpdateFeed, http.MethodPatch, feed.ID.String(),
		map[string]any{"name": "Family", "include_alarms": true}), http.StatusOK)
	if body := serveFeed(anonymous, updated.Path).Body.String(); !strings.Contains(body, "BEGIN:VALARM") {
		t.Fatalf("expected alarms once included, got:\n%s", body)
	}

	reset := decodeFeed(t, feedRequestTo(t, owner.ResetFeedToken, http.MethodPost, feed.ID.String(), nil), http.StatusOK)
	if reset.Path == feed.Path {
		t.Fatal("expected a new path after reset")
	}
	if code := serveFeed(anonymous, feed.Path).Code; code != http.StatusNotFound {
		t.Fatalf("expected the old path to answer %d, got %d", http.StatusNotFound, code)
	}
	if code := feedRequestTo(t, owner.DeleteFeed, http.MethodDelete, feed.ID.String(), nil).Code; code != http.StatusNoContent {
		t.Fatalf("expected %d, got %d", http.StatusNoContent, code)
	}
	if code := serveFeed(anonymous, reset.Path).Code; code != http.StatusNotFound {
		t.Fatalf("expected a deleted feed to answer %d, got %d", http.StatusNotFound, code)
	}
}
