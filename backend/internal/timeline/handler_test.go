package timeline

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jonathan-skantz/calendar/backend/db"
	"github.com/jonathan-skantz/calendar/backend/internal/testutil"
)

func TestEventMainTagCannotBeRemoved(t *testing.T) {
	store, ownerID := setupTestStore(t)
	ctx := context.Background()
	mainTag, err := store.CreateTag(ctx, ownerID, "Main")
	if err != nil {
		t.Fatal(err)
	}
	extraTag, err := store.CreateTag(ctx, ownerID, "Extra")
	if err != nil {
		t.Fatal(err)
	}
	eventID, err := store.CreateEvent(ctx, ownerID, db.Event{
		Title:     "Main tag removal",
		StartsAt:  time.Date(2026, 10, 9, 10, 0, 0, 0, time.UTC),
		EndsAt:    time.Date(2026, 10, 9, 11, 0, 0, 0, time.UTC),
		Kind:      "informational",
		MainTagID: mainTag.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.AddTagToItem(ctx, ownerID, eventID, extraTag.ID); err != nil {
		t.Fatal(err)
	}
	handler := New(store, testutil.SignedInAs(ownerID))
	remove := func(tagID uuid.UUID) int {
		request := httptest.NewRequest(http.MethodDelete, "/api/timeline/"+eventID.String()+"/tags/"+tagID.String(), nil)
		request.SetPathValue("itemID", eventID.String())
		request.SetPathValue("tagID", tagID.String())
		response := httptest.NewRecorder()
		handler.RemoveTagFromItem(response, request)
		return response.Code
	}
	if status := remove(mainTag.ID); status != http.StatusConflict {
		t.Fatalf("main tag: expected status %d, got %d", http.StatusConflict, status)
	}
	if status := remove(extraTag.ID); status != http.StatusNoContent {
		t.Fatalf("non-main tag: expected status %d, got %d", http.StatusNoContent, status)
	}
}

// testTimelinePath lists the weeks around the dates these tests use.
const testTimelinePath = "/api/timeline?from=2026-10-01T00:00:00Z&to=2026-11-01T00:00:00Z"

func setupTestStore(t *testing.T) (*db.Store, uuid.UUID) {
	ctx := context.Background()
	store := testutil.NewStore(t)
	userID, err := store.Register(ctx, "test@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	return store, userID
}

func TestCreateRequiresAuthentication(t *testing.T) {
	handler := New(nil, func(*http.Request) (uuid.UUID, error) {
		return uuid.Nil, errors.New("not authenticated")
	})
	request := httptest.NewRequest(http.MethodPost, "/api/events", bytes.NewBufferString(`{}`))
	response := httptest.NewRecorder()

	handler.CreateEvent(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, response.Code)
	}
}

func TestRequireUserWritesUnauthorizedResponse(t *testing.T) {
	handler := New(nil, func(*http.Request) (uuid.UUID, error) {
		return uuid.Nil, errors.New("not authenticated")
	})
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/timeline", nil)

	userID, ok := handler.requireUser(response, request)

	if ok || userID != uuid.Nil {
		t.Fatalf("expected no user, got %s", userID)
	}
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, response.Code)
	}
	if response.Body.String() != "authentication required\n" {
		t.Fatalf("unexpected response body %q", response.Body.String())
	}
}

func TestRequireUserReturnsAuthenticatedUser(t *testing.T) {
	expectedID := uuid.New()
	handler := New(nil, func(*http.Request) (uuid.UUID, error) {
		return expectedID, nil
	})
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/timeline", nil)

	userID, ok := handler.requireUser(response, request)

	if !ok || userID != expectedID {
		t.Fatalf("expected user %s, got %s", expectedID, userID)
	}
	if response.Code != http.StatusOK {
		t.Fatalf("expected no response before handler work, got status %d", response.Code)
	}
}

func TestCreateEventValidatesInput(t *testing.T) {
	handler := New(nil, func(*http.Request) (uuid.UUID, error) { return uuid.New(), nil })
	request := httptest.NewRequest(http.MethodPost, "/api/events", bytes.NewBufferString(`{
		"title":"Meeting",
		"starts_at":"2026-10-09T10:00:00Z",
		"ends_at":"2026-10-09T09:00:00Z",
		"kind":"attendable"
	}`))
	response := httptest.NewRecorder()

	handler.CreateEvent(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}
}

func TestCreateEventRejectsCommuteOutsideTimedAttendableEvents(t *testing.T) {
	handler := New(nil, func(*http.Request) (uuid.UUID, error) { return uuid.New(), nil })
	for name, body := range map[string]string{
		"informational":  `{"title":"Info","starts_at":"2026-10-09T10:00:00Z","ends_at":"2026-10-09T11:00:00Z","kind":"informational","commute_before_seconds":600}`,
		"all-day":        `{"title":"Trip","starts_at":"2026-10-09T00:00:00Z","ends_at":"2026-10-10T00:00:00Z","is_all_day":true,"kind":"attendable","commute_after_seconds":600}`,
		"negative":       `{"title":"Meet","starts_at":"2026-10-09T10:00:00Z","ends_at":"2026-10-09T11:00:00Z","kind":"attendable","commute_before_seconds":-60}`,
		"partial minute": `{"title":"Meet","starts_at":"2026-10-09T10:00:00Z","ends_at":"2026-10-09T11:00:00Z","kind":"attendable","commute_before_seconds":90}`,
		"over a day":     `{"title":"Meet","starts_at":"2026-10-09T10:00:00Z","ends_at":"2026-10-09T11:00:00Z","kind":"attendable","commute_after_seconds":6000000000}`,
	} {
		t.Run(name, func(t *testing.T) {
			response := httptest.NewRecorder()
			handler.CreateEvent(response, httptest.NewRequest(http.MethodPost, "/api/events", bytes.NewBufferString(body)))
			if response.Code != http.StatusBadRequest {
				t.Fatalf("expected status %d, got %d", http.StatusBadRequest, response.Code)
			}
		})
	}
}

func TestCreateEventRejectsApproximateAllDayEvents(t *testing.T) {
	handler := New(nil, func(*http.Request) (uuid.UUID, error) { return uuid.New(), nil })
	response := httptest.NewRecorder()
	handler.CreateEvent(response, httptest.NewRequest(http.MethodPost, "/api/events", bytes.NewBufferString(
		`{"title":"Trip","starts_at":"2026-10-09T00:00:00Z","ends_at":"2026-10-10T00:00:00Z","is_all_day":true,"kind":"attendable","end_is_approximate":true}`,
	)))
	if response.Code != http.StatusBadRequest || response.Body.String() != ErrInvalidApproximate.Error()+"\n" {
		t.Fatalf("expected approximate error, got %d: %s", response.Code, response.Body.String())
	}
}

func TestCreateEventRejectsGatheringOutsideTimedAttendableEvents(t *testing.T) {
	handler := New(nil, func(*http.Request) (uuid.UUID, error) { return uuid.New(), nil })
	for name, body := range map[string]string{
		"informational":  `{"title":"Info","starts_at":"2026-10-09T10:00:00Z","ends_at":"2026-10-09T11:00:00Z","kind":"informational","gathering_seconds":600}`,
		"all-day":        `{"title":"Trip","starts_at":"2026-10-09T00:00:00Z","ends_at":"2026-10-10T00:00:00Z","is_all_day":true,"kind":"attendable","gathering_seconds":600}`,
		"negative":       `{"title":"Meet","starts_at":"2026-10-09T10:00:00Z","ends_at":"2026-10-09T11:00:00Z","kind":"attendable","gathering_seconds":-60}`,
		"partial minute": `{"title":"Meet","starts_at":"2026-10-09T10:00:00Z","ends_at":"2026-10-09T11:00:00Z","kind":"attendable","gathering_seconds":90}`,
		"label only":     `{"title":"Meet","starts_at":"2026-10-09T10:00:00Z","ends_at":"2026-10-09T11:00:00Z","kind":"attendable","gathering_label":"Warm-up"}`,
		"optional only":  `{"title":"Meet","starts_at":"2026-10-09T10:00:00Z","ends_at":"2026-10-09T11:00:00Z","kind":"attendable","gathering_is_optional":true}`,
		"long label":     `{"title":"Meet","starts_at":"2026-10-09T10:00:00Z","ends_at":"2026-10-09T11:00:00Z","kind":"attendable","gathering_seconds":600,"gathering_label":"` + strings.Repeat("å", 41) + `"}`,
	} {
		t.Run(name, func(t *testing.T) {
			response := httptest.NewRecorder()
			handler.CreateEvent(response, httptest.NewRequest(http.MethodPost, "/api/events", bytes.NewBufferString(body)))
			if response.Code != http.StatusBadRequest || response.Body.String() != ErrInvalidGathering.Error()+"\n" {
				t.Fatalf("expected gathering error, got %d: %s", response.Code, response.Body.String())
			}
		})
	}
}

func TestGatheringIsStoredAndShared(t *testing.T) {
	store, ownerID := setupTestStore(t)
	ctx := context.Background()
	recipientID, err := store.Register(ctx, "recipient@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	currentUser := ownerID
	handler := New(store, func(*http.Request) (uuid.UUID, error) { return currentUser, nil })

	response := httptest.NewRecorder()
	handler.CreateEvent(response, httptest.NewRequest(http.MethodPost, "/api/events", bytes.NewBufferString(`{
		"title":"Football",
		"starts_at":"2026-10-09T18:00:00Z",
		"ends_at":"2026-10-09T20:00:00Z",
		"kind":"attendable",
		"gathering_seconds":900,
		"gathering_label":"   ",
		"commute_before_seconds":1200
	}`)))
	if response.Code != http.StatusCreated {
		t.Fatalf("create: expected status %d, got %d: %s", http.StatusCreated, response.Code, response.Body.String())
	}
	var created createResponse
	if err := json.NewDecoder(response.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}

	update := httptest.NewRequest(http.MethodPut, "/api/events/"+created.ID.String(), bytes.NewBufferString(`{
		"title":"Football",
		"starts_at":"2026-10-09T18:00:00Z",
		"ends_at":"2026-10-09T20:00:00Z",
		"kind":"attendable",
		"gathering_seconds":1800,
		"gathering_is_optional":true,
		"gathering_label":" Mingle ",
		"commute_before_seconds":1200
	}`))
	update.SetPathValue("eventID", created.ID.String())
	response = httptest.NewRecorder()
	handler.UpdateEvent(response, update)
	if response.Code != http.StatusNoContent {
		t.Fatalf("update: expected status %d, got %d: %s", http.StatusNoContent, response.Code, response.Body.String())
	}

	shareRequest := httptest.NewRequest(http.MethodPost, "/api/events/"+created.ID.String()+"/shares",
		bytes.NewBufferString(`{"recipient_email":"recipient@example.com"}`))
	shareRequest.SetPathValue("eventID", created.ID.String())
	response = httptest.NewRecorder()
	handler.ShareEvent(response, shareRequest)
	if response.Code != http.StatusCreated {
		t.Fatalf("share: expected status %d, got %d: %s", http.StatusCreated, response.Code, response.Body.String())
	}

	for name, test := range map[string]struct {
		userID        uuid.UUID
		commuteBefore int
	}{
		"owner":     {userID: ownerID, commuteBefore: 1200},
		"recipient": {userID: recipientID, commuteBefore: 0},
	} {
		currentUser = test.userID
		response = httptest.NewRecorder()
		handler.ListTimeline(response, httptest.NewRequest(http.MethodGet, testTimelinePath, nil))
		var entries []db.TimelineEntry
		if err := json.NewDecoder(response.Body).Decode(&entries); err != nil {
			t.Fatal(err)
		}
		if len(entries) != 1 || entries[0].GatheringSeconds != 1800 || !entries[0].GatheringIsOptional ||
			entries[0].GatheringLabel == nil || *entries[0].GatheringLabel != "Mingle" ||
			entries[0].CommuteBeforeSeconds != test.commuteBefore {
			t.Fatalf("%s: unexpected gathering or commute %+v", name, entries)
		}
	}
}

func TestCreateEventRejectsSkippedInformationalEvent(t *testing.T) {
	handler := New(nil, func(*http.Request) (uuid.UUID, error) { return uuid.New(), nil })
	response := httptest.NewRecorder()
	handler.CreateEvent(response, httptest.NewRequest(http.MethodPost, "/api/events", bytes.NewBufferString(
		`{"title":"Release","starts_at":"2026-10-09T10:00:00Z","ends_at":"2026-10-09T11:00:00Z","kind":"informational","is_skipped":true}`,
	)))
	if response.Code != http.StatusBadRequest || response.Body.String() != ErrInvalidSkip.Error()+"\n" {
		t.Fatalf("expected skip error, got %d: %s", response.Code, response.Body.String())
	}
}

func TestSkippedEventKeepsCommuteAndIsNotShared(t *testing.T) {
	store, ownerID := setupTestStore(t)
	ctx := context.Background()
	recipientID, err := store.Register(ctx, "recipient@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	currentUser := ownerID
	handler := New(store, func(*http.Request) (uuid.UUID, error) { return currentUser, nil })

	response := httptest.NewRecorder()
	handler.CreateEvent(response, httptest.NewRequest(http.MethodPost, "/api/events", bytes.NewBufferString(`{
		"title":"Lecture",
		"starts_at":"2026-10-09T10:00:00Z",
		"ends_at":"2026-10-09T12:00:00Z",
		"kind":"attendable",
		"commute_before_seconds":1200
	}`)))
	if response.Code != http.StatusCreated {
		t.Fatalf("create: expected status %d, got %d: %s", http.StatusCreated, response.Code, response.Body.String())
	}
	var created createResponse
	if err := json.NewDecoder(response.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}

	update := httptest.NewRequest(http.MethodPut, "/api/events/"+created.ID.String(), bytes.NewBufferString(`{
		"title":"Lecture",
		"starts_at":"2026-10-09T10:00:00Z",
		"ends_at":"2026-10-09T12:00:00Z",
		"kind":"attendable",
		"is_skipped":true,
		"commute_before_seconds":1200
	}`))
	update.SetPathValue("eventID", created.ID.String())
	response = httptest.NewRecorder()
	handler.UpdateEvent(response, update)
	if response.Code != http.StatusNoContent {
		t.Fatalf("update: expected status %d, got %d: %s", http.StatusNoContent, response.Code, response.Body.String())
	}

	shareRequest := httptest.NewRequest(http.MethodPost, "/api/events/"+created.ID.String()+"/shares",
		bytes.NewBufferString(`{"recipient_email":"recipient@example.com"}`))
	shareRequest.SetPathValue("eventID", created.ID.String())
	response = httptest.NewRecorder()
	handler.ShareEvent(response, shareRequest)
	if response.Code != http.StatusCreated {
		t.Fatalf("share: expected status %d, got %d: %s", http.StatusCreated, response.Code, response.Body.String())
	}

	timeline := func(userID uuid.UUID) []db.TimelineEntry {
		currentUser = userID
		response := httptest.NewRecorder()
		handler.ListTimeline(response, httptest.NewRequest(http.MethodGet, testTimelinePath, nil))
		var entries []db.TimelineEntry
		if err := json.NewDecoder(response.Body).Decode(&entries); err != nil {
			t.Fatal(err)
		}
		return entries
	}
	if owned := timeline(ownerID); len(owned) != 1 || !owned[0].IsSkipped || owned[0].CommuteBeforeSeconds != 1200 {
		t.Fatalf("owner: unexpected entries %+v", owned)
	}
	if shared := timeline(recipientID); len(shared) != 1 || shared[0].IsSkipped {
		t.Fatalf("recipient: unexpected entries %+v", shared)
	}
}

func TestApproximateTimesAreStoredAndShared(t *testing.T) {
	store, ownerID := setupTestStore(t)
	ctx := context.Background()
	recipientID, err := store.Register(ctx, "recipient@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	currentUser := ownerID
	handler := New(store, func(*http.Request) (uuid.UUID, error) { return currentUser, nil })

	response := httptest.NewRecorder()
	handler.CreateEvent(response, httptest.NewRequest(http.MethodPost, "/api/events", bytes.NewBufferString(`{
		"title":"Workshop",
		"starts_at":"2026-10-09T10:00:00Z",
		"ends_at":"2026-10-09T18:00:00Z",
		"kind":"informational",
		"start_is_approximate":true,
		"end_is_approximate":true
	}`)))
	if response.Code != http.StatusCreated {
		t.Fatalf("create: expected status %d, got %d: %s", http.StatusCreated, response.Code, response.Body.String())
	}
	var created createResponse
	if err := json.NewDecoder(response.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}

	update := httptest.NewRequest(http.MethodPut, "/api/events/"+created.ID.String(), bytes.NewBufferString(`{
		"title":"Workshop",
		"starts_at":"2026-10-09T10:00:00Z",
		"ends_at":"2026-10-09T18:00:00Z",
		"kind":"informational",
		"end_is_approximate":true
	}`))
	update.SetPathValue("eventID", created.ID.String())
	response = httptest.NewRecorder()
	handler.UpdateEvent(response, update)
	if response.Code != http.StatusNoContent {
		t.Fatalf("update: expected status %d, got %d: %s", http.StatusNoContent, response.Code, response.Body.String())
	}

	shareRequest := httptest.NewRequest(http.MethodPost, "/api/events/"+created.ID.String()+"/shares",
		bytes.NewBufferString(`{"recipient_email":"recipient@example.com"}`))
	shareRequest.SetPathValue("eventID", created.ID.String())
	response = httptest.NewRecorder()
	handler.ShareEvent(response, shareRequest)
	if response.Code != http.StatusCreated {
		t.Fatalf("share: expected status %d, got %d: %s", http.StatusCreated, response.Code, response.Body.String())
	}

	for name, userID := range map[string]uuid.UUID{"owner": ownerID, "recipient": recipientID} {
		currentUser = userID
		response = httptest.NewRecorder()
		handler.ListTimeline(response, httptest.NewRequest(http.MethodGet, testTimelinePath, nil))
		var entries []db.TimelineEntry
		if err := json.NewDecoder(response.Body).Decode(&entries); err != nil {
			t.Fatal(err)
		}
		if len(entries) != 1 || entries[0].StartIsApproximate || !entries[0].EndIsApproximate {
			t.Fatalf("%s: unexpected approximate flags %+v", name, entries)
		}
	}
}

func TestCreateEventAndTask(t *testing.T) {
	store, userID := setupTestStore(t)
	handler := New(store, testutil.SignedInAs(userID))

	eventRequest := httptest.NewRequest(http.MethodPost, "/api/events", bytes.NewBufferString(`{
		"title":"Family dinner",
		"description":"At home",
		"starts_at":"2026-10-09T18:00:00Z",
		"ends_at":"2026-10-09T19:00:00Z",
		"kind":"attendable",
		"location":"Home",
		"commute_before_seconds":900,
		"commute_after_seconds":900
	}`))
	eventResponse := httptest.NewRecorder()
	handler.CreateEvent(eventResponse, eventRequest)
	if eventResponse.Code != http.StatusCreated {
		t.Fatalf("event: expected status %d, got %d: %s", http.StatusCreated, eventResponse.Code, eventResponse.Body.String())
	}
	var createdEventID createResponse
	if err := json.NewDecoder(eventResponse.Body).Decode(&createdEventID); err != nil {
		t.Fatal(err)
	}
	tags, err := store.ListTags(context.Background(), userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(tags) != 1 || !tags[0].IsDefault {
		t.Fatalf("expected a default tag, got %+v", tags)
	}
	tagCount, err := store.CountItemTags(context.Background(), userID, createdEventID.ID)
	if err != nil {
		t.Fatal(err)
	}
	if tagCount < 1 {
		t.Fatalf("created event %s has no tags", createdEventID.ID)
	}

	taskRequest := httptest.NewRequest(http.MethodPost, "/api/tasks", bytes.NewBufferString(`{
		"title":"Buy ingredients",
		"deadline_at":"2026-10-09T16:00:00Z"
	}`))
	taskResponse := httptest.NewRecorder()
	handler.CreateTask(taskResponse, taskRequest)
	if taskResponse.Code != http.StatusCreated {
		t.Fatalf("task: expected status %d, got %d: %s", http.StatusCreated, taskResponse.Code, taskResponse.Body.String())
	}
}

func TestUpdateEventAndTask(t *testing.T) {
	store, ownerID := setupTestStore(t)
	eventID, err := store.CreateEvent(context.Background(), ownerID, db.Event{
		Title:    "Before",
		StartsAt: time.Date(2026, 10, 9, 10, 0, 0, 0, time.UTC),
		EndsAt:   time.Date(2026, 10, 9, 11, 0, 0, 0, time.UTC),
		Kind:     "informational",
	})
	if err != nil {
		t.Fatal(err)
	}
	taskID, err := store.CreateTask(context.Background(), ownerID, db.Task{
		Title:      "Task before",
		DeadlineAt: time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	handler := New(store, testutil.SignedInAs(ownerID))

	eventRequest := httptest.NewRequest(http.MethodPut, "/api/events/"+eventID.String(), bytes.NewBufferString(`{
		"title":"After",
		"starts_at":"2026-10-09T13:00:00Z",
		"ends_at":"2026-10-09T14:00:00Z",
		"kind":"attendable",
		"location":"Office"
	}`))
	eventRequest.SetPathValue("eventID", eventID.String())
	eventResponse := httptest.NewRecorder()
	handler.UpdateEvent(eventResponse, eventRequest)
	if eventResponse.Code != http.StatusNoContent {
		t.Fatalf("update event: expected status %d, got %d", http.StatusNoContent, eventResponse.Code)
	}

	taskRequest := httptest.NewRequest(http.MethodPut, "/api/tasks/"+taskID.String(), bytes.NewBufferString(`{
		"title":"Task after",
		"deadline_at":"2026-10-09T15:00:00Z"
	}`))
	taskRequest.SetPathValue("taskID", taskID.String())
	taskResponse := httptest.NewRecorder()
	handler.UpdateTask(taskResponse, taskRequest)
	if taskResponse.Code != http.StatusNoContent {
		t.Fatalf("update task: expected status %d, got %d", http.StatusNoContent, taskResponse.Code)
	}

	otherUserID, err := store.Register(context.Background(), "other@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	otherHandler := New(store, testutil.SignedInAs(otherUserID))
	eventRequest = httptest.NewRequest(http.MethodPut, "/api/events/"+eventID.String(), bytes.NewBufferString(`{
		"title":"Unauthorized",
		"starts_at":"2026-10-09T13:00:00Z",
		"ends_at":"2026-10-09T14:00:00Z",
		"kind":"informational"
	}`))
	eventRequest.SetPathValue("eventID", eventID.String())
	eventResponse = httptest.NewRecorder()
	otherHandler.UpdateEvent(eventResponse, eventRequest)
	if eventResponse.Code != http.StatusNotFound || eventResponse.Body.String() != "event not found\n" {
		t.Fatalf("other user update: expected 404 event not found, got %d %q", eventResponse.Code, eventResponse.Body.String())
	}

	taskRequest = httptest.NewRequest(http.MethodPut, "/api/tasks/"+taskID.String(), bytes.NewBufferString(`{
		"title":"Unauthorized",
		"deadline_at":"2026-10-09T15:00:00Z"
	}`))
	taskRequest.SetPathValue("taskID", taskID.String())
	taskResponse = httptest.NewRecorder()
	otherHandler.UpdateTask(taskResponse, taskRequest)
	if taskResponse.Code != http.StatusNotFound || taskResponse.Body.String() != "task not found\n" {
		t.Fatalf("other user task update: expected 404 task not found, got %d %q", taskResponse.Code, taskResponse.Body.String())
	}
}

func TestEventMainTagMustBeOwnedAndAttached(t *testing.T) {
	store, ownerID := setupTestStore(t)
	otherUserID, err := store.Register(context.Background(), "other@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	otherTag, err := store.CreateTag(context.Background(), otherUserID, "Other")
	if err != nil {
		t.Fatal(err)
	}
	ownerTag, err := store.CreateTag(context.Background(), ownerID, "Owner")
	if err != nil {
		t.Fatal(err)
	}
	eventID, err := store.CreateEvent(context.Background(), ownerID, db.Event{
		Title:    "Main tag test",
		StartsAt: time.Date(2026, 10, 9, 10, 0, 0, 0, time.UTC),
		EndsAt:   time.Date(2026, 10, 9, 11, 0, 0, 0, time.UTC),
		Kind:     "informational",
	})
	if err != nil {
		t.Fatal(err)
	}
	handler := New(store, testutil.SignedInAs(ownerID))
	update := func(tagID uuid.UUID) int {
		request := httptest.NewRequest(http.MethodPut, "/api/events/"+eventID.String(), bytes.NewBufferString(`{
			"title":"Main tag test",
			"starts_at":"2026-10-09T10:00:00Z",
			"ends_at":"2026-10-09T11:00:00Z",
			"kind":"informational",
			"main_tag_id":"`+tagID.String()+`"
		}`))
		request.SetPathValue("eventID", eventID.String())
		response := httptest.NewRecorder()
		handler.UpdateEvent(response, request)
		return response.Code
	}
	if status := update(otherTag.ID); status != http.StatusNotFound {
		t.Fatalf("other user's main tag: expected status %d, got %d", http.StatusNotFound, status)
	}
	if status := update(ownerTag.ID); status != http.StatusNotFound {
		t.Fatalf("unattached main tag: expected status %d, got %d", http.StatusNotFound, status)
	}
	if err := store.AddTagToItem(context.Background(), ownerID, eventID, ownerTag.ID); err != nil {
		t.Fatal(err)
	}
	if status := update(ownerTag.ID); status != http.StatusNoContent {
		t.Fatalf("attached main tag: expected status %d, got %d", http.StatusNoContent, status)
	}
}

func TestSetTaskCompleted(t *testing.T) {
	store, ownerID := setupTestStore(t)
	taskID, err := store.CreateTask(context.Background(), ownerID, db.Task{
		Title:      "Task",
		DeadlineAt: time.Date(2026, 10, 9, 16, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	handler := New(store, testutil.SignedInAs(ownerID))

	request := httptest.NewRequest(http.MethodPatch, "/api/tasks/"+taskID.String(), bytes.NewBufferString(`{"is_completed":true}`))
	request.SetPathValue("taskID", taskID.String())
	response := httptest.NewRecorder()
	handler.SetTaskCompleted(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("complete task: expected status %d, got %d", http.StatusNoContent, response.Code)
	}

	listResponse := httptest.NewRecorder()
	handler.ListTimeline(listResponse, httptest.NewRequest(http.MethodGet, testTimelinePath, nil))
	var entries []db.TimelineEntry
	if err := json.NewDecoder(listResponse.Body).Decode(&entries); err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || !entries[0].IsCompleted {
		t.Fatalf("expected completed task, got %+v", entries)
	}

	request = httptest.NewRequest(http.MethodPatch, "/api/tasks/"+taskID.String(), bytes.NewBufferString(`{"is_completed":false}`))
	request.SetPathValue("taskID", taskID.String())
	response = httptest.NewRecorder()
	handler.SetTaskCompleted(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("reopen task: expected status %d, got %d", http.StatusNoContent, response.Code)
	}

	otherUserID, err := store.Register(context.Background(), "other@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	otherHandler := New(store, testutil.SignedInAs(otherUserID))
	request = httptest.NewRequest(http.MethodPatch, "/api/tasks/"+taskID.String(), bytes.NewBufferString(`{"is_completed":true}`))
	request.SetPathValue("taskID", taskID.String())
	response = httptest.NewRecorder()
	otherHandler.SetTaskCompleted(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("other user update: expected status %d, got %d", http.StatusNotFound, response.Code)
	}
}

func TestDeleteTimelineItem(t *testing.T) {
	store, ownerID := setupTestStore(t)
	itemID, err := store.CreateTask(context.Background(), ownerID, db.Task{
		Title:      "Delete me",
		DeadlineAt: time.Date(2026, 10, 9, 16, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	handler := New(store, testutil.SignedInAs(ownerID))

	request := httptest.NewRequest(http.MethodDelete, "/api/timeline/"+itemID.String(), nil)
	request.SetPathValue("itemID", itemID.String())
	response := httptest.NewRecorder()
	handler.DeleteTimelineItem(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("delete item: expected status %d, got %d", http.StatusNoContent, response.Code)
	}

	listResponse := httptest.NewRecorder()
	handler.ListTimeline(listResponse, httptest.NewRequest(http.MethodGet, testTimelinePath, nil))
	var entries []db.TimelineEntry
	if err := json.NewDecoder(listResponse.Body).Decode(&entries); err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected deleted item to be absent, got %+v", entries)
	}

	otherUserID, err := store.Register(context.Background(), "other@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	otherItemID, err := store.CreateTask(context.Background(), ownerID, db.Task{
		Title:      "Keep me",
		DeadlineAt: time.Date(2026, 10, 9, 17, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	otherHandler := New(store, testutil.SignedInAs(otherUserID))
	request = httptest.NewRequest(http.MethodDelete, "/api/timeline/"+otherItemID.String(), nil)
	request.SetPathValue("itemID", otherItemID.String())
	response = httptest.NewRecorder()
	otherHandler.DeleteTimelineItem(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("other user delete: expected status %d, got %d", http.StatusNotFound, response.Code)
	}
}

func TestCreateTagStopsAtTheLimit(t *testing.T) {
	store, userID := setupTestStore(t)
	handler := New(store, testutil.SignedInAs(userID))
	existing, err := store.ListTags(context.Background(), userID)
	if err != nil {
		t.Fatal(err)
	}
	for i := len(existing); i < db.MaxTagsPerUser; i++ {
		if _, err := store.CreateTag(context.Background(), userID, "tag "+strconv.Itoa(i)); err != nil {
			t.Fatal(err)
		}
	}
	response := httptest.NewRecorder()
	handler.CreateTag(response, httptest.NewRequest(http.MethodPost, "/api/tags", strings.NewReader(`{"name":"one more"}`)))
	if response.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d: %s", http.StatusConflict, response.Code, response.Body.String())
	}
}

func TestPersonalTags(t *testing.T) {
	store, userID := setupTestStore(t)
	handler := New(store, testutil.SignedInAs(userID))

	eventID, err := store.CreateEvent(context.Background(), userID, db.Event{
		Title:    "Tagged event",
		StartsAt: time.Date(2026, 10, 9, 18, 0, 0, 0, time.UTC),
		EndsAt:   time.Date(2026, 10, 9, 19, 0, 0, 0, time.UTC),
		Kind:     "informational",
	})
	if err != nil {
		t.Fatal(err)
	}

	createTagRequest := httptest.NewRequest(http.MethodPost, "/api/tags", bytes.NewBufferString(`{"name":"Family"}`))
	createTagResponse := httptest.NewRecorder()
	handler.CreateTag(createTagResponse, createTagRequest)
	if createTagResponse.Code != http.StatusCreated {
		t.Fatalf("create tag: expected status %d, got %d", http.StatusCreated, createTagResponse.Code)
	}
	var tag db.Tag
	if err := json.NewDecoder(createTagResponse.Body).Decode(&tag); err != nil {
		t.Fatal(err)
	}

	listTagsResponse := httptest.NewRecorder()
	handler.ListTags(listTagsResponse, httptest.NewRequest(http.MethodGet, "/api/tags", nil))
	if listTagsResponse.Code != http.StatusOK {
		t.Fatalf("list tags: expected status %d, got %d", http.StatusOK, listTagsResponse.Code)
	}
	var listedTags []db.Tag
	if err := json.NewDecoder(listTagsResponse.Body).Decode(&listedTags); err != nil {
		t.Fatal(err)
	}
	if len(listedTags) != 2 {
		t.Fatalf("expected general and Family tags, got %+v", listedTags)
	}
	var foundDefault bool
	var defaultTagID uuid.UUID
	for _, listedTag := range listedTags {
		if listedTag.Name == "general" && listedTag.IsDefault {
			foundDefault = true
			defaultTagID = listedTag.ID
		}
	}
	if !foundDefault {
		t.Fatalf("expected general to be default, got %+v", listedTags)
	}

	addTagRequest := httptest.NewRequest(http.MethodPost, "/api/timeline/"+eventID.String()+"/tags", bytes.NewBufferString(`{"tag_id":"`+tag.ID.String()+`"}`))
	addTagRequest.SetPathValue("itemID", eventID.String())
	addTagResponse := httptest.NewRecorder()
	handler.AddTagToItem(addTagResponse, addTagRequest)
	if addTagResponse.Code != http.StatusNoContent {
		t.Fatalf("add tag: expected status %d, got %d", http.StatusNoContent, addTagResponse.Code)
	}

	listResponse := httptest.NewRecorder()
	handler.ListTimeline(listResponse, httptest.NewRequest(http.MethodGet, testTimelinePath, nil))
	if listResponse.Code != http.StatusOK {
		t.Fatalf("list timeline: expected status %d, got %d", http.StatusOK, listResponse.Code)
	}
	var entries []db.TimelineEntry
	if err := json.NewDecoder(listResponse.Body).Decode(&entries); err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || len(entries[0].Tags) != 2 {
		t.Fatalf("expected tag on timeline entry, got %+v", entries)
	}
	var hasFamily bool
	for _, entryTag := range entries[0].Tags {
		hasFamily = hasFamily || entryTag.Name == "Family"
	}
	if !hasFamily {
		t.Fatalf("expected Family tag on timeline entry, got %+v", entries[0].Tags)
	}

	removeTagRequest := httptest.NewRequest(http.MethodDelete, "/api/timeline/"+eventID.String()+"/tags/"+tag.ID.String(), nil)
	removeTagRequest.SetPathValue("itemID", eventID.String())
	removeTagRequest.SetPathValue("tagID", tag.ID.String())
	removeTagResponse := httptest.NewRecorder()
	handler.RemoveTagFromItem(removeTagResponse, removeTagRequest)
	if removeTagResponse.Code != http.StatusNoContent {
		t.Fatalf("remove tag: expected status %d, got %d", http.StatusNoContent, removeTagResponse.Code)
	}

	eventWithDeletedMainTagID, err := store.CreateEvent(context.Background(), userID, db.Event{
		Title:     "Event with tag to delete",
		StartsAt:  time.Date(2026, 10, 10, 18, 0, 0, 0, time.UTC),
		EndsAt:    time.Date(2026, 10, 10, 19, 0, 0, 0, time.UTC),
		Kind:      "informational",
		MainTagID: tag.ID,
	})
	if err != nil {
		t.Fatal(err)
	}

	deleteTagRequest := httptest.NewRequest(http.MethodDelete, "/api/tags/"+tag.ID.String(), nil)
	deleteTagRequest.SetPathValue("tagID", tag.ID.String())
	deleteTagResponse := httptest.NewRecorder()
	handler.DeleteTag(deleteTagResponse, deleteTagRequest)
	if deleteTagResponse.Code != http.StatusNoContent {
		t.Fatalf("delete tag used as main tag: expected status %d, got %d: %s", http.StatusNoContent, deleteTagResponse.Code, deleteTagResponse.Body.String())
	}

	listResponse = httptest.NewRecorder()
	handler.ListTimeline(listResponse, httptest.NewRequest(http.MethodGet, testTimelinePath, nil))
	if listResponse.Code != http.StatusOK {
		t.Fatalf("list timeline after deleting main tag: expected status %d, got %d", http.StatusOK, listResponse.Code)
	}
	if err := json.NewDecoder(listResponse.Body).Decode(&entries); err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.ID != eventWithDeletedMainTagID {
			continue
		}
		if entry.MainTagID == nil || *entry.MainTagID != defaultTagID {
			t.Fatalf("expected replacement main tag %s, got %+v", defaultTagID, entry.MainTagID)
		}
		if len(entry.Tags) != 1 || entry.Tags[0].ID != defaultTagID {
			t.Fatalf("expected replacement tag to remain attached, got %+v", entry.Tags)
		}
		return
	}
	t.Fatalf("timeline did not contain event %s after deleting its main tag", eventWithDeletedMainTagID)
}

func TestShareEventWithPerRecipientFields(t *testing.T) {
	store, ownerID := setupTestStore(t)
	ctx := context.Background()
	xID, err := store.Register(ctx, "x@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	yID, err := store.Register(ctx, "y@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	currentUser := ownerID
	handler := New(store, func(*http.Request) (uuid.UUID, error) { return currentUser, nil })

	eventResponse := httptest.NewRecorder()
	handler.CreateEvent(eventResponse, httptest.NewRequest(http.MethodPost, "/api/events", bytes.NewBufferString(`{
		"title":"Secret owner title",
		"description":"Owner details",
		"starts_at":"2026-10-09T18:00:00Z",
		"ends_at":"2026-10-09T19:00:00Z",
		"kind":"attendable",
		"location":"Owner location"
	}`)))
	if eventResponse.Code != http.StatusCreated {
		t.Fatalf("event: expected status %d, got %d", http.StatusCreated, eventResponse.Code)
	}
	var created createResponse
	if err := json.NewDecoder(eventResponse.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	eventID := created.ID.String()

	request := func(method, path, body string, fn http.HandlerFunc, shareID string) *httptest.ResponseRecorder {
		if path == "" {
			path = "/api/events/" + eventID + "/shares/" + shareID
		}
		req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
		req.SetPathValue("eventID", eventID)
		req.SetPathValue("shareID", shareID)
		response := httptest.NewRecorder()
		fn(response, req)
		return response
	}
	share := func(body string) *httptest.ResponseRecorder {
		return request(http.MethodPost, "/api/events/"+eventID+"/shares", body, handler.ShareEvent, "")
	}
	timeline := func(userID uuid.UUID) (string, []db.TimelineEntry) {
		currentUser = userID
		defer func() { currentUser = ownerID }()
		response := httptest.NewRecorder()
		handler.ListTimeline(response, httptest.NewRequest(http.MethodGet, testTimelinePath, nil))
		body := response.Body.String()
		var entries []db.TimelineEntry
		if err := json.Unmarshal([]byte(body), &entries); err != nil {
			t.Fatal(err)
		}
		return body, entries
	}

	xShare := share(`{"recipient_email":"X@example.com","title":"Title for X","location":" Gate B ","url":"  "}`)
	if xShare.Code != http.StatusCreated {
		t.Fatalf("share with x: expected status %d, got %d: %s", http.StatusCreated, xShare.Code, xShare.Body.String())
	}
	var xShareID createResponse
	if err := json.NewDecoder(xShare.Body).Decode(&xShareID); err != nil {
		t.Fatal(err)
	}
	if code := share(`{"recipient_email":"y@example.com"}`).Code; code != http.StatusCreated {
		t.Fatalf("share with y: expected status %d, got %d", http.StatusCreated, code)
	}
	if code := share(`{"recipient_email":"y@example.com"}`).Code; code != http.StatusConflict {
		t.Fatalf("expected sharing twice to conflict, got %d", code)
	}
	if code := share(`{"recipient_email":"test@example.com"}`).Code; code != http.StatusBadRequest {
		t.Fatalf("expected sharing with yourself to fail, got %d", code)
	}
	if code := share(`{"recipient_email":"nobody@example.com"}`).Code; code != http.StatusNotFound {
		t.Fatalf("expected unknown recipient to fail, got %d", code)
	}

	list := request(http.MethodGet, "/api/events/"+eventID+"/shares", "", handler.ListEventShares, "")
	var shares []db.EventShare
	if err := json.NewDecoder(list.Body).Decode(&shares); err != nil {
		t.Fatal(err)
	}
	if len(shares) != 2 || shares[0].RecipientEmail != "x@example.com" || shares[0].RecipientDisplayName != "x" ||
		shares[0].Title == nil || *shares[0].Title != "Title for X" ||
		shares[0].Location == nil || *shares[0].Location != "Gate B" || shares[0].URL != nil ||
		shares[1].Title != nil {
		t.Fatalf("unexpected shares %+v", shares)
	}

	xBody, xEntries := timeline(xID)
	if len(xEntries) != 1 || xEntries[0].Title != "Title for X" || *xEntries[0].Location != "Gate B" ||
		*xEntries[0].Description != "Owner details" || !xEntries[0].IsShared ||
		xEntries[0].SharedBy == nil || *xEntries[0].SharedBy != "test" {
		t.Fatalf("unexpected x timeline %+v", xEntries)
	}
	if strings.Contains(xBody, "Secret owner title") || strings.Contains(xBody, "main_tag_id") ||
		strings.Contains(xBody, `"tags"`) || strings.Contains(xBody, "Owner location") {
		t.Fatalf("x timeline leaked owner data: %s", xBody)
	}
	if _, yEntries := timeline(yID); len(yEntries) != 1 || yEntries[0].Title != "Secret owner title" {
		t.Fatalf("unexpected y timeline %+v", yEntries)
	}
	if _, ownerEntries := timeline(ownerID); len(ownerEntries) != 1 || ownerEntries[0].ShareCount != 2 {
		t.Fatalf("expected owner to see two shares, got %+v", ownerEntries)
	}

	// Later edits to the event reach recipients unless the owner set their own value.
	updateEvent := httptest.NewRequest(http.MethodPut, "/api/events/"+eventID, bytes.NewBufferString(`{
		"title":"Renamed",
		"starts_at":"2026-10-09T20:00:00Z",
		"ends_at":"2026-10-09T21:00:00Z",
		"kind":"attendable"
	}`))
	updateEvent.SetPathValue("eventID", eventID)
	updateResponse := httptest.NewRecorder()
	handler.UpdateEvent(updateResponse, updateEvent)
	if updateResponse.Code != http.StatusNoContent {
		t.Fatalf("update event: expected status %d, got %d", http.StatusNoContent, updateResponse.Code)
	}
	if _, xEntries := timeline(xID); xEntries[0].Title != "Title for X" || xEntries[0].StartsAt.UTC().Hour() != 20 {
		t.Fatalf("x should keep their title and get the new time, got %+v", xEntries[0])
	}
	if _, yEntries := timeline(yID); yEntries[0].Title != "Renamed" {
		t.Fatalf("y should follow the event title, got %+v", yEntries[0])
	}

	if code := request(http.MethodPut, "", `{"title":"","location":"Gate C"}`, handler.UpdateEventShare, xShareID.ID.String()).Code; code != http.StatusNoContent {
		t.Fatalf("update share: expected status %d, got %d", http.StatusNoContent, code)
	}
	if _, xEntries := timeline(xID); xEntries[0].Title != "Renamed" || *xEntries[0].Location != "Gate C" {
		t.Fatalf("x should follow the event title again, got %+v", xEntries[0])
	}

	// Recipients cannot manage shares, even for an event shared with them.
	currentUser = xID
	if code := request(http.MethodGet, "", "", handler.ListEventShares, "").Code; code != http.StatusNotFound {
		t.Fatalf("recipient listed shares: got %d", code)
	}
	if code := request(http.MethodPut, "", `{"title":"Mine"}`, handler.UpdateEventShare, xShareID.ID.String()).Code; code != http.StatusNotFound {
		t.Fatalf("recipient updated share: got %d", code)
	}
	if code := request(http.MethodDelete, "", "", handler.DeleteEventShare, xShareID.ID.String()).Code; code != http.StatusNotFound {
		t.Fatalf("recipient deleted share: got %d", code)
	}
	currentUser = ownerID

	if code := request(http.MethodDelete, "", "", handler.DeleteEventShare, xShareID.ID.String()).Code; code != http.StatusNoContent {
		t.Fatalf("delete share: expected status %d, got %d", http.StatusNoContent, code)
	}
	if _, xEntries := timeline(xID); len(xEntries) != 0 {
		t.Fatalf("expected x to lose the event, got %+v", xEntries)
	}
}

func TestStoreFailuresAreNotReportedAsNotFound(t *testing.T) {
	store, userID := setupTestStore(t)
	handler := New(store, testutil.SignedInAs(userID))
	setDefault := func(ctx context.Context, tagID uuid.UUID) *httptest.ResponseRecorder {
		request := httptest.NewRequestWithContext(ctx, http.MethodPut, "/api/tags/default",
			bytes.NewBufferString(`{"tag_id":"`+tagID.String()+`"}`))
		response := httptest.NewRecorder()
		handler.SetDefaultTag(response, request)
		return response
	}

	if response := setDefault(context.Background(), uuid.New()); response.Code != http.StatusNotFound {
		t.Fatalf("unknown tag: expected status %d, got %d", http.StatusNotFound, response.Code)
	}

	// A cancelled request makes the store fail for a reason other than a
	// missing tag, which must surface as a server error.
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if response := setDefault(cancelled, uuid.New()); response.Code != http.StatusInternalServerError {
		t.Fatalf("store failure: expected status %d, got %d", http.StatusInternalServerError, response.Code)
	}
}
