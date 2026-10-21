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
	"github.com/jonathan-skantz/calendar/backend/internal/recurrence"
	"github.com/jonathan-skantz/calendar/backend/internal/testutil"
)

var stockholm = func() *time.Location {
	zone, err := time.LoadLocation("Europe/Stockholm")
	if err != nil {
		panic(err)
	}
	return zone
}()

type listedOccurrence struct {
	ID              uuid.UUID        `json:"id"`
	Title           string           `json:"title"`
	StartsAt        time.Time        `json:"starts_at"`
	EndsAt          time.Time        `json:"ends_at"`
	IsSkipped       bool             `json:"is_skipped"`
	IsShared        bool             `json:"is_shared"`
	OccurrenceStart *time.Time       `json:"occurrence_start"`
	Recurrence      *recurrence.Rule `json:"recurrence"`
}

func listRange(t *testing.T, handler *Handler, from, to time.Time) []listedOccurrence {
	t.Helper()
	response := httptest.NewRecorder()
	handler.ListTimeline(response, httptest.NewRequest(http.MethodGet,
		"/api/timeline?from="+from.UTC().Format(time.RFC3339)+"&to="+to.UTC().Format(time.RFC3339), nil))
	if response.Code != http.StatusOK {
		t.Fatalf("list: expected %d, got %d: %s", http.StatusOK, response.Code, response.Body.String())
	}
	var items []listedOccurrence
	if err := json.NewDecoder(response.Body).Decode(&items); err != nil {
		t.Fatal(err)
	}
	return items
}

func sendEvent(t *testing.T, handler *Handler, method, eventID string, body map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(method, "/api/events/"+eventID, bytes.NewReader(encoded))
	response := httptest.NewRecorder()
	if method == http.MethodPost {
		handler.CreateEvent(response, request)
	} else {
		request.SetPathValue("eventID", eventID)
		handler.UpdateEvent(response, request)
	}
	return response
}

// lecture is Tuesdays and Thursdays at 10:00 in Stockholm from 20 October
// 2026, across the end of summer time on the 25th.
func lecture(recurrenceRule map[string]any) map[string]any {
	return map[string]any{
		"title": "Lecture", "kind": "attendable",
		"starts_at":  time.Date(2026, 10, 20, 10, 0, 0, 0, stockholm),
		"ends_at":    time.Date(2026, 10, 20, 12, 0, 0, 0, stockholm),
		"recurrence": recurrenceRule, "time_zone": "Europe/Stockholm",
	}
}

func createdID(t *testing.T, response *httptest.ResponseRecorder) uuid.UUID {
	t.Helper()
	if response.Code != http.StatusCreated {
		t.Fatalf("create: expected %d, got %d: %s", http.StatusCreated, response.Code, response.Body.String())
	}
	var created createResponse
	if err := json.NewDecoder(response.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	return created.ID
}

func TestListTimelineNeedsARange(t *testing.T) {
	handler := New(nil, testutil.SignedInAs(uuid.New()))
	for name, query := range map[string]string{
		"only from": "?from=2026-10-01T00:00:00Z",
		"not times": "?from=monday&to=friday",
		"reversed":  "?from=2026-10-02T00:00:00Z&to=2026-10-01T00:00:00Z",
		"too long":  "?from=2026-01-01T00:00:00Z&to=2026-04-03T00:00:01Z",
	} {
		t.Run(name, func(t *testing.T) {
			response := httptest.NewRecorder()
			handler.ListTimeline(response, httptest.NewRequest(http.MethodGet, "/api/timeline"+query, nil))
			if response.Code != http.StatusBadRequest || response.Body.String() != ErrInvalidTimelineRange.Error()+"\n" {
				t.Fatalf("expected the range error, got %d: %s", response.Code, response.Body.String())
			}
		})
	}
}

func TestRepeatingEventsAreListedAsOccurrences(t *testing.T) {
	ctx := context.Background()
	store, ownerID := setupTestStore(t)
	recipientID, err := store.Register(ctx, "recipient@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	owner := New(store, testutil.SignedInAs(ownerID))
	eventID := createdID(t, sendEvent(t, owner, http.MethodPost, "", lecture(map[string]any{
		"frequency": "weekly", "interval": 1, "weekdays": []string{"TH", "TU"}, "count": 4,
	})))
	if _, err := store.ShareEvent(ctx, ownerID, eventID, recipientID, db.ShareFields{}); err != nil {
		t.Fatal(err)
	}
	skipped := true
	if err := store.SetOccurrenceException(ctx, ownerID, eventID, db.OccurrenceException{
		OriginalStart: time.Date(2026, 10, 22, 10, 0, 0, 0, stockholm), IsSkipped: &skipped,
	}, nil); err != nil {
		t.Fatal(err)
	}

	from, to := time.Date(2026, 10, 19, 0, 0, 0, 0, stockholm), time.Date(2026, 11, 2, 0, 0, 0, 0, stockholm)
	items := listRange(t, owner, from, to)
	wantStarts := []time.Time{
		time.Date(2026, 10, 20, 10, 0, 0, 0, stockholm), time.Date(2026, 10, 22, 10, 0, 0, 0, stockholm),
		time.Date(2026, 10, 27, 10, 0, 0, 0, stockholm), time.Date(2026, 10, 29, 10, 0, 0, 0, stockholm),
	}
	if len(items) != len(wantStarts) {
		t.Fatalf("expected %d occurrences, got %+v", len(wantStarts), items)
	}
	for index, item := range items {
		if item.ID != eventID || !item.StartsAt.Equal(wantStarts[index]) || item.EndsAt.Sub(item.StartsAt) != 2*time.Hour ||
			item.OccurrenceStart == nil || !item.OccurrenceStart.Equal(wantStarts[index]) {
			t.Fatalf("occurrence %d: unexpected %+v", index, item)
		}
		if item.IsSkipped != (index == 1) {
			t.Fatalf("occurrence %d: expected only the 22nd skipped, got %+v", index, item)
		}
	}
	if rule := items[0].Recurrence; rule == nil || rule.Frequency != recurrence.Weekly ||
		len(rule.Weekdays) != 2 || rule.Weekdays[0] != "TU" || rule.Count != 4 {
		t.Fatalf("expected the normalized rule, got %+v", items[0].Recurrence)
	}

	// Recipients see the occurrences, but never the owner's skip.
	for _, item := range listRange(t, New(store, testutil.SignedInAs(recipientID)), from, to) {
		if !item.IsShared || item.IsSkipped {
			t.Fatalf("unexpected shared occurrence %+v", item)
		}
	}

	// A range after the last occurrence has none.
	if items := listRange(t, owner, to, to.AddDate(0, 0, 14)); len(items) != 0 {
		t.Fatalf("expected no occurrences after the count, got %+v", items)
	}
}

func TestRepeatValidation(t *testing.T) {
	store, ownerID := setupTestStore(t)
	handler := New(store, testutil.SignedInAs(ownerID))
	for name, test := range map[string]struct {
		body map[string]any
		want error
	}{
		"no time zone": {
			func() map[string]any {
				body := lecture(map[string]any{"frequency": "daily", "interval": 1})
				delete(body, "time_zone")
				return body
			}(),
			ErrInvalidTimeZone,
		},
		"start's weekday missing": {lecture(map[string]any{"frequency": "weekly", "interval": 1, "weekdays": []string{"WE"}}), recurrence.ErrWeekdays},
		"unknown frequency":       {lecture(map[string]any{"frequency": "hourly", "interval": 1}), recurrence.ErrInvalidRule},
		"ends before the start":   {lecture(map[string]any{"frequency": "daily", "interval": 1, "until": "2026-10-01"}), recurrence.ErrUntil},
	} {
		t.Run(name, func(t *testing.T) {
			response := sendEvent(t, handler, http.MethodPost, "", test.body)
			if response.Code != http.StatusBadRequest || response.Body.String() != test.want.Error()+"\n" {
				t.Fatalf("expected %q, got %d: %s", test.want, response.Code, response.Body.String())
			}
		})
	}
}

func TestUpdatingARuleKeepsOnlyMatchingExceptions(t *testing.T) {
	ctx := context.Background()
	store, ownerID := setupTestStore(t)
	handler := New(store, testutil.SignedInAs(ownerID))
	eventID := createdID(t, sendEvent(t, handler, http.MethodPost, "", lecture(map[string]any{
		"frequency": "weekly", "interval": 1, "weekdays": []string{"TU", "TH"},
	})))
	for _, day := range []int{22, 27} {
		if err := store.SetOccurrenceException(ctx, ownerID, eventID, db.OccurrenceException{
			OriginalStart: time.Date(2026, 10, day, 10, 0, 0, 0, stockholm), IsCancelled: true,
		}, nil); err != nil {
			t.Fatal(err)
		}
	}

	// Thursdays are dropped, so the 22nd's exception goes; the 27th's stays.
	response := sendEvent(t, handler, http.MethodPut, eventID.String(), lecture(map[string]any{
		"frequency": "weekly", "interval": 1, "weekdays": []string{"TU"}, "until": "2026-11-10",
	}))
	if response.Code != http.StatusNoContent {
		t.Fatalf("update: expected %d, got %d: %s", http.StatusNoContent, response.Code, response.Body.String())
	}
	entry, err := store.GetOwnEvent(ctx, ownerID, eventID)
	if err != nil {
		t.Fatal(err)
	}
	if len(entry.Exceptions) != 1 || !entry.Exceptions[0].OriginalStart.Equal(time.Date(2026, 10, 27, 10, 0, 0, 0, stockholm)) {
		t.Fatalf("expected only the 27th's exception, got %+v", entry.Exceptions)
	}
	if entry.RecurrenceEndsAt == nil || !entry.RecurrenceEndsAt.Equal(time.Date(2026, 11, 10, 12, 0, 0, 0, stockholm)) {
		t.Fatalf("expected the series to end on 10 November at 12:00, got %v", entry.RecurrenceEndsAt)
	}

	// Without a rule it is a plain event again.
	plain := lecture(nil)
	delete(plain, "recurrence")
	if response := sendEvent(t, handler, http.MethodPut, eventID.String(), plain); response.Code != http.StatusNoContent {
		t.Fatalf("plain update: expected %d, got %d: %s", http.StatusNoContent, response.Code, response.Body.String())
	}
	if entry, _ := store.GetOwnEvent(ctx, ownerID, eventID); entry.RecurrenceRule != nil || len(entry.Exceptions) != 0 {
		t.Fatalf("expected a plain event, got %+v", entry)
	}

	if response := sendEvent(t, handler, http.MethodPut, uuid.NewString(), plain); response.Code != http.StatusNotFound {
		t.Fatalf("unknown event: expected %d, got %d", http.StatusNotFound, response.Code)
	}
}
