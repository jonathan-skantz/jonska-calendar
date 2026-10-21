package ical

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/jonathan-skantz/calendar/backend/db"
	"github.com/jonathan-skantz/calendar/backend/internal/testutil"
	"github.com/jonathan-skantz/calendar/backend/internal/timeline"
)

func postImport(t *testing.T, handler *Handler, request map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.Import(response, httptest.NewRequest(http.MethodPost, "/api/import/ics", bytes.NewReader(body)))
	return response
}

func decodeImport(t *testing.T, response *httptest.ResponseRecorder) importResponse {
	t.Helper()
	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, response.Code, response.Body.String())
	}
	var result importResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	return result
}

func TestImportAndExportRequireAuthentication(t *testing.T) {
	handler := New(nil, func(*http.Request) (uuid.UUID, error) { return uuid.Nil, errors.New("not signed in") })
	if code := postImport(t, handler, map[string]any{}).Code; code != http.StatusUnauthorized {
		t.Fatalf("import: expected %d, got %d", http.StatusUnauthorized, code)
	}
	response := httptest.NewRecorder()
	handler.Export(response, httptest.NewRequest(http.MethodGet, "/api/export.ics?time_zone=UTC", nil))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("export: expected %d, got %d", http.StatusUnauthorized, response.Code)
	}
}

func TestImportRejectsBadRequests(t *testing.T) {
	handler := New(nil, testutil.SignedInAs(uuid.New()))
	file := calendar("DTSTART:20261024T100000Z")
	for name, test := range map[string]struct {
		request map[string]any
		status  int
	}{
		"missing time zone": {map[string]any{"ics": file}, http.StatusBadRequest},
		"local time zone":   {map[string]any{"ics": file, "time_zone": "Local"}, http.StatusBadRequest},
		"unknown zone":      {map[string]any{"ics": file, "time_zone": "Mars/Base"}, http.StatusBadRequest},
		"not a calendar":    {map[string]any{"ics": "hello", "time_zone": "UTC"}, http.StatusBadRequest},
		"too large":         {map[string]any{"ics": strings.Repeat("x", maxImportBodyBytes), "time_zone": "UTC"}, http.StatusRequestEntityTooLarge},
	} {
		t.Run(name, func(t *testing.T) {
			if code := postImport(t, handler, test.request).Code; code != test.status {
				t.Fatalf("expected %d, got %d", test.status, code)
			}
		})
	}
}

func TestImportPreviewsThenImports(t *testing.T) {
	ctx := context.Background()
	store := testutil.NewStore(t)
	userID, err := store.Register(ctx, "owner@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	family, err := store.CreateTag(ctx, userID, "Family")
	if err != nil {
		t.Fatal(err)
	}
	handler := New(store, testutil.SignedInAs(userID))
	file := calendar(
		"UID:dinner\nSUMMARY:Dinner\nDTSTART:20261024T160000Z\nDTEND:20261024T190000Z\nCATEGORIES:family,Unknown",
		"UID:trip\nSUMMARY:Trip\nDTSTART;VALUE=DATE:20261024\nX-JONSKA-COMMUTE-BEFORE:PT30M",
		"UID:weekly\nSUMMARY:Weekly\nDTSTART:20261024T100000Z\nRRULE:FREQ=WEEKLY",
	)
	request := map[string]any{"ics": file, "time_zone": "Europe/Stockholm", "dry_run": true}

	preview := decodeImport(t, postImport(t, handler, request))
	wantSkipped := []Skipped{
		{Title: "Weekly", Reason: ReasonRecurring},
		{Title: "Trip", Reason: timeline.ErrInvalidCommute.Error()},
	}
	if preview.Imported != 1 || !slices.Equal(preview.Skipped, wantSkipped) || len(preview.Warnings) != 0 {
		t.Fatalf("unexpected preview %+v", preview)
	}
	if events, _ := store.ListEventsForExport(ctx, userID); len(events) != 0 {
		t.Fatalf("a preview must not write, got %+v", events)
	}

	request["dry_run"] = false
	if result := decodeImport(t, postImport(t, handler, request)); result.Imported != 1 {
		t.Fatalf("unexpected import %+v", result)
	}
	events, err := store.ListEventsForExport(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Title != "Dinner" || !slices.Equal(events[0].TagNames, []string{"general", "Family"}) {
		t.Fatalf("expected Dinner with the default and Family tags, got %+v", events)
	}

	again := decodeImport(t, postImport(t, handler, request))
	if again.Imported != 0 || !slices.Contains(again.Skipped, Skipped{Title: "Dinner", Reason: ReasonDuplicate}) {
		t.Fatalf("unexpected re-import %+v", again)
	}

	// A chosen main tag must be the user's own.
	request["tag_id"] = uuid.New()
	request["ics"] = calendar("UID:other\nSUMMARY:Other\nDTSTART:20261024T100000Z")
	if code := postImport(t, handler, request).Code; code != http.StatusNotFound {
		t.Fatalf("foreign tag: expected %d, got %d", http.StatusNotFound, code)
	}
	request["tag_id"] = family.ID
	decodeImport(t, postImport(t, handler, request))
	events, err = store.ListEventsForExport(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	if index := slices.IndexFunc(events, func(event db.ExportEvent) bool { return event.Title == "Other" }); index < 0 ||
		events[index].MainTagID != family.ID {
		t.Fatalf("expected Other with Family as main tag, got %+v", events)
	}
}

func TestExportDownloadsOwnEvents(t *testing.T) {
	ctx := context.Background()
	store := testutil.NewStore(t)
	ownerID, err := store.Register(ctx, "owner@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	recipientID, err := store.Register(ctx, "recipient@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	startsAt := time.Date(2026, 10, 24, 16, 0, 0, 0, time.UTC)
	eventID, err := store.CreateEvent(ctx, ownerID, db.Event{
		Title: "Concert", StartsAt: startsAt, EndsAt: startsAt.Add(2 * time.Hour), Kind: db.EventKindAttendable,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.ShareEvent(ctx, ownerID, eventID, recipientID, db.ShareFields{}); err != nil {
		t.Fatal(err)
	}

	export := func(userID uuid.UUID, timeZone string) *httptest.ResponseRecorder {
		response := httptest.NewRecorder()
		New(store, testutil.SignedInAs(userID)).Export(response,
			httptest.NewRequest(http.MethodGet, "/api/export.ics?time_zone="+timeZone, nil))
		return response
	}
	if code := export(ownerID, "").Code; code != http.StatusBadRequest {
		t.Fatalf("missing time zone: expected %d, got %d", http.StatusBadRequest, code)
	}

	response := export(ownerID, "Europe/Stockholm")
	if response.Code != http.StatusOK ||
		response.Header().Get("Content-Type") != "text/calendar; charset=utf-8" ||
		!strings.Contains(response.Header().Get("Content-Disposition"), `filename="calendar.ics"`) {
		t.Fatalf("unexpected response %d %v", response.Code, response.Header())
	}
	parsed, err := Parse(response.Body, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed.Events) != 1 || parsed.Events[0].Title != "Concert" || parsed.Events[0].UID != eventID.String()+db.ExportUIDSuffix {
		t.Fatalf("unexpected export %+v", parsed)
	}

	shared, err := Parse(export(recipientID, "UTC").Body, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	if len(shared.Events) != 0 {
		t.Fatalf("events shared with the user must not be exported, got %+v", shared.Events)
	}
}
