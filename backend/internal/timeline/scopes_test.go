package timeline

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/jonathan-skantz/calendar/backend/internal/recurrence"
	"github.com/jonathan-skantz/calendar/backend/internal/testutil"
)

func at(day, hour, minute int) time.Time {
	return time.Date(2026, 10, day, hour, minute, 0, 0, stockholm)
}

func scopeQuery(scope string, occurrence time.Time) string {
	return "?" + url.Values{"scope": {scope}, "occurrence": {occurrence.UTC().Format(time.RFC3339)}}.Encode()
}

func updateScoped(t *testing.T, handler *Handler, eventID uuid.UUID, query string, body map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPut, "/api/events/"+eventID.String()+query, bytes.NewReader(encoded))
	request.SetPathValue("eventID", eventID.String())
	response := httptest.NewRecorder()
	handler.UpdateEvent(response, request)
	return response
}

func deleteScoped(t *testing.T, handler *Handler, eventID uuid.UUID, query string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodDelete, "/api/timeline/"+eventID.String()+query, nil)
	request.SetPathValue("itemID", eventID.String())
	response := httptest.NewRecorder()
	handler.DeleteTimelineItem(response, request)
	return response
}

// occurrenceBody is the lecture as the form sends it for one occurrence.
func occurrenceBody(starts, ends time.Time, rule map[string]any) map[string]any {
	body := lecture(rule)
	body["starts_at"], body["ends_at"] = starts, ends
	return body
}

var tuesdaysAndThursdays = map[string]any{"frequency": "weekly", "interval": 1, "weekdays": []string{"TU", "TH"}}

type listed struct {
	id     uuid.UUID
	title  string
	starts time.Time
}

func listOctober(t *testing.T, handler *Handler) []listed {
	t.Helper()
	var result []listed
	for _, item := range listRange(t, handler, at(19, 0, 0), time.Date(2026, 11, 2, 0, 0, 0, 0, stockholm)) {
		result = append(result, listed{item.ID, item.Title, item.StartsAt})
	}
	return result
}

func expectStarts(t *testing.T, got []listed, want ...time.Time) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("expected %v, got %+v", want, got)
	}
	for index := range want {
		if !got[index].starts.Equal(want[index]) {
			t.Fatalf("expected %v, got %+v", want, got)
		}
	}
}

func TestScopeRequestsAreChecked(t *testing.T) {
	store, ownerID := setupTestStore(t)
	handler := New(store, testutil.SignedInAs(ownerID))
	seriesID := createdID(t, sendEvent(t, handler, http.MethodPost, "", lecture(tuesdaysAndThursdays)))
	plain := lecture(nil)
	delete(plain, "recurrence")
	plainID := createdID(t, sendEvent(t, handler, http.MethodPost, "", plain))
	body := occurrenceBody(at(22, 10, 0), at(22, 12, 0), tuesdaysAndThursdays)

	for name, test := range map[string]struct {
		id     uuid.UUID
		query  string
		body   map[string]any
		status int
		want   error
	}{
		"unknown scope":     {seriesID, "?scope=some&occurrence=2026-10-22T08:00:00Z", body, http.StatusBadRequest, ErrInvalidScope},
		"no occurrence":     {seriesID, "?scope=this", body, http.StatusBadRequest, ErrInvalidScope},
		"plain event":       {plainID, scopeQuery("this", at(20, 10, 0)), body, http.StatusBadRequest, ErrNotRepeating},
		"not an occurrence": {seriesID, scopeQuery("this", at(21, 10, 0)), body, http.StatusNotFound, ErrOccurrenceNotFound},
		"title for one": {seriesID, scopeQuery("this", at(22, 10, 0)), func() map[string]any {
			changed := occurrenceBody(at(22, 10, 0), at(22, 12, 0), tuesdaysAndThursdays)
			changed["title"] = "Guest lecture"
			return changed
		}(), http.StatusBadRequest, ErrOccurrenceFields},
	} {
		t.Run(name, func(t *testing.T) {
			response := updateScoped(t, handler, test.id, test.query, test.body)
			if response.Code != test.status || response.Body.String() != test.want.Error()+"\n" {
				t.Fatalf("expected %d %q, got %d: %s", test.status, test.want, response.Code, response.Body.String())
			}
		})
	}

	stranger := New(store, testutil.SignedInAs(uuid.New()))
	if code := updateScoped(t, stranger, seriesID, scopeQuery("this", at(22, 10, 0)), body).Code; code != http.StatusNotFound {
		t.Fatalf("stranger: expected %d, got %d", http.StatusNotFound, code)
	}
	if code := deleteScoped(t, stranger, seriesID, scopeQuery("this", at(22, 10, 0))).Code; code != http.StatusNotFound {
		t.Fatalf("stranger delete: expected %d, got %d", http.StatusNotFound, code)
	}
}

func TestChangeOneOccurrence(t *testing.T) {
	store, ownerID := setupTestStore(t)
	handler := New(store, testutil.SignedInAs(ownerID))
	seriesID := createdID(t, sendEvent(t, handler, http.MethodPost, "", lecture(tuesdaysAndThursdays)))

	moved := occurrenceBody(at(23, 14, 0), at(23, 15, 0), tuesdaysAndThursdays)
	moved["is_skipped"] = true
	if response := updateScoped(t, handler, seriesID, scopeQuery("this", at(22, 10, 0)), moved); response.Code != http.StatusNoContent {
		t.Fatalf("move: expected %d, got %d: %s", http.StatusNoContent, response.Code, response.Body.String())
	}
	items := listRange(t, handler, at(19, 0, 0), time.Date(2026, 11, 2, 0, 0, 0, 0, stockholm))
	var starts []time.Time
	for _, item := range items {
		starts = append(starts, item.StartsAt)
		if item.StartsAt.Equal(at(23, 14, 0)) && (!item.IsSkipped || !item.OccurrenceStart.Equal(at(22, 10, 0))) {
			t.Fatalf("expected the moved occurrence skipped and keyed by its original start, got %+v", item)
		}
	}
	if !slices.EqualFunc(starts, []time.Time{at(20, 10, 0), at(23, 14, 0), at(27, 10, 0), at(29, 10, 0)}, time.Time.Equal) {
		t.Fatalf("unexpected starts %v", starts)
	}

	// Deleting one occurrence leaves the others.
	if code := deleteScoped(t, handler, seriesID, scopeQuery("this", at(27, 10, 0))).Code; code != http.StatusNoContent {
		t.Fatalf("delete one: expected %d, got %d", http.StatusNoContent, code)
	}
	expectStarts(t, listOctober(t, handler), at(20, 10, 0), at(23, 14, 0), at(29, 10, 0))
}

func TestChangeAllOccurrencesFromOne(t *testing.T) {
	ctx := t.Context()
	store, ownerID := setupTestStore(t)
	handler := New(store, testutil.SignedInAs(ownerID))
	seriesID := createdID(t, sendEvent(t, handler, http.MethodPost, "", lecture(tuesdaysAndThursdays)))
	if code := deleteScoped(t, handler, seriesID, scopeQuery("this", at(29, 10, 0))).Code; code != http.StatusNoContent {
		t.Fatalf("delete one: expected %d, got %d", http.StatusNoContent, code)
	}

	// Moving the occurrence on Tuesday the 27th (after summer time ended) to
	// Wednesday at 11:00 moves the series to Wednesdays and Fridays at 11:00.
	body := occurrenceBody(at(28, 11, 0), at(28, 13, 0), tuesdaysAndThursdays)
	body["title"] = "Moved lecture"
	if response := updateScoped(t, handler, seriesID, scopeQuery("all", at(27, 10, 0)), body); response.Code != http.StatusNoContent {
		t.Fatalf("all: expected %d, got %d: %s", http.StatusNoContent, response.Code, response.Body.String())
	}
	got := listOctober(t, handler)
	// The cancelled Thursday the 29th moved with the series to Friday the 30th.
	expectStarts(t, got, at(21, 11, 0), at(23, 11, 0), at(28, 11, 0))
	if got[0].title != "Moved lecture" {
		t.Fatalf("expected the new title on every occurrence, got %+v", got)
	}
	entry, err := store.GetOwnEvent(ctx, ownerID, seriesID)
	if err != nil {
		t.Fatal(err)
	}
	if len(entry.Exceptions) != 1 || !entry.Exceptions[0].OriginalStart.Equal(at(30, 11, 0)) {
		t.Fatalf("expected the cancellation re-keyed to the 30th, got %+v", entry.Exceptions)
	}
}

func TestChangeFollowingOccurrences(t *testing.T) {
	ctx := t.Context()
	store, ownerID := setupTestStore(t)
	handler := New(store, testutil.SignedInAs(ownerID))
	sixTimes := map[string]any{"frequency": "weekly", "interval": 1, "weekdays": []string{"TU", "TH"}, "count": 6}
	seriesID := createdID(t, sendEvent(t, handler, http.MethodPost, "", lecture(sixTimes)))

	body := occurrenceBody(at(27, 18, 0), at(27, 20, 0), sixTimes)
	body["title"] = "Evening lecture"
	response := updateScoped(t, handler, seriesID, scopeQuery("following", at(27, 10, 0)), body)
	if response.Code != http.StatusOK {
		t.Fatalf("following: expected %d, got %d: %s", http.StatusOK, response.Code, response.Body.String())
	}
	var split createResponse
	if err := json.NewDecoder(response.Body).Decode(&split); err != nil {
		t.Fatal(err)
	}
	newID := split.ID

	got := listOctober(t, handler)
	expectStarts(t, got, at(20, 10, 0), at(22, 10, 0), at(27, 18, 0), at(29, 18, 0))
	if got[1].id != seriesID || got[2].id != newID || got[2].title != "Evening lecture" {
		t.Fatalf("expected the old series, then the new one, got %+v", got)
	}
	created, err := store.GetOwnEvent(ctx, ownerID, newID)
	if err != nil {
		t.Fatal(err)
	}
	// Two of six happened before the split, so four are left.
	if *created.RecurrenceRule != (recurrence.Rule{Frequency: recurrence.Weekly, Interval: 1, Weekdays: []string{"TU", "TH"}, Count: 4}).RRULE(at(27, 18, 0), false) {
		t.Fatalf("expected four occurrences left, got %s", *created.RecurrenceRule)
	}

	// Deleting from the 29th on ends the new series after the 27th; deleting
	// from the first occurrence deletes the old series.
	if code := deleteScoped(t, handler, newID, scopeQuery("following", at(29, 18, 0))).Code; code != http.StatusNoContent {
		t.Fatalf("delete following: expected %d, got %d", http.StatusNoContent, code)
	}
	if code := deleteScoped(t, handler, seriesID, scopeQuery("following", at(20, 10, 0))).Code; code != http.StatusNoContent {
		t.Fatalf("delete from the start: expected %d, got %d", http.StatusNoContent, code)
	}
	expectStarts(t, listOctober(t, handler), at(27, 18, 0))
	if _, err := store.GetOwnEvent(ctx, ownerID, seriesID); err == nil {
		t.Fatal("expected the old series to be deleted")
	}
}
