package timeline

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jonathan-skantz/calendar/backend/db"
	"github.com/jonathan-skantz/calendar/backend/internal/testutil"
)

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

func TestCreateEventAndTask(t *testing.T) {
	store, userID := setupTestStore(t)
	handler := New(store, func(*http.Request) (uuid.UUID, error) { return userID, nil })

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
	tagCount, err := store.CountItemTags(context.Background(), createdEventID.ID)
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
	handler := New(store, func(*http.Request) (uuid.UUID, error) { return ownerID, nil })

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
	otherHandler := New(store, func(*http.Request) (uuid.UUID, error) { return otherUserID, nil })
	eventRequest = httptest.NewRequest(http.MethodPut, "/api/events/"+eventID.String(), bytes.NewBufferString(`{
		"title":"Unauthorized",
		"starts_at":"2026-10-09T13:00:00Z",
		"ends_at":"2026-10-09T14:00:00Z",
		"kind":"informational"
	}`))
	eventRequest.SetPathValue("eventID", eventID.String())
	eventResponse = httptest.NewRecorder()
	otherHandler.UpdateEvent(eventResponse, eventRequest)
	if eventResponse.Code != http.StatusNotFound {
		t.Fatalf("other user update: expected status %d, got %d", http.StatusNotFound, eventResponse.Code)
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
	handler := New(store, func(*http.Request) (uuid.UUID, error) { return ownerID, nil })
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
	handler := New(store, func(*http.Request) (uuid.UUID, error) { return ownerID, nil })

	request := httptest.NewRequest(http.MethodPatch, "/api/tasks/"+taskID.String(), bytes.NewBufferString(`{"is_completed":true}`))
	request.SetPathValue("taskID", taskID.String())
	response := httptest.NewRecorder()
	handler.SetTaskCompleted(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("complete task: expected status %d, got %d", http.StatusNoContent, response.Code)
	}

	listResponse := httptest.NewRecorder()
	handler.ListTimeline(listResponse, httptest.NewRequest(http.MethodGet, "/api/timeline", nil))
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
	otherHandler := New(store, func(*http.Request) (uuid.UUID, error) { return otherUserID, nil })
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
	handler := New(store, func(*http.Request) (uuid.UUID, error) { return ownerID, nil })

	request := httptest.NewRequest(http.MethodDelete, "/api/timeline/"+itemID.String(), nil)
	request.SetPathValue("itemID", itemID.String())
	response := httptest.NewRecorder()
	handler.DeleteTimelineItem(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("delete item: expected status %d, got %d", http.StatusNoContent, response.Code)
	}

	listResponse := httptest.NewRecorder()
	handler.ListTimeline(listResponse, httptest.NewRequest(http.MethodGet, "/api/timeline", nil))
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
	otherHandler := New(store, func(*http.Request) (uuid.UUID, error) { return otherUserID, nil })
	request = httptest.NewRequest(http.MethodDelete, "/api/timeline/"+otherItemID.String(), nil)
	request.SetPathValue("itemID", otherItemID.String())
	response = httptest.NewRecorder()
	otherHandler.DeleteTimelineItem(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("other user delete: expected status %d, got %d", http.StatusNotFound, response.Code)
	}
}

func TestPersonalTags(t *testing.T) {
	store, userID := setupTestStore(t)
	handler := New(store, func(*http.Request) (uuid.UUID, error) { return userID, nil })

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
	for _, listedTag := range listedTags {
		foundDefault = foundDefault || (listedTag.Name == "general" && listedTag.IsDefault)
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
	handler.ListTimeline(listResponse, httptest.NewRequest(http.MethodGet, "/api/timeline", nil))
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
}

func TestShareEventAndUpdateLocalAppearance(t *testing.T) {
	store, ownerID := setupTestStore(t)
	recipientID, err := store.Register(context.Background(), "recipient@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}

	ownerHandler := New(store, func(*http.Request) (uuid.UUID, error) { return ownerID, nil })
	eventRequest := httptest.NewRequest(http.MethodPost, "/api/events", bytes.NewBufferString(`{
		"title":"Original title",
		"description":"Original details",
		"url":"https://example.com/original",
		"starts_at":"2026-10-09T18:00:00Z",
		"ends_at":"2026-10-09T19:00:00Z",
		"kind":"informational",
		"location":"Original location"
	}`))
	eventResponse := httptest.NewRecorder()
	ownerHandler.CreateEvent(eventResponse, eventRequest)
	if eventResponse.Code != http.StatusCreated {
		t.Fatalf("event: expected status %d, got %d", http.StatusCreated, eventResponse.Code)
	}
	var created createResponse
	if err := json.NewDecoder(eventResponse.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}

	shareRequest := httptest.NewRequest(http.MethodPost, "/api/events/"+created.ID.String()+"/shares", bytes.NewBufferString(`{"recipient_email":"recipient@example.com"}`))
	shareRequest.SetPathValue("eventID", created.ID.String())
	shareResponse := httptest.NewRecorder()
	ownerHandler.ShareEvent(shareResponse, shareRequest)
	if shareResponse.Code != http.StatusCreated {
		t.Fatalf("share: expected status %d, got %d: %s", http.StatusCreated, shareResponse.Code, shareResponse.Body.String())
	}
	var share createResponse
	if err := json.NewDecoder(shareResponse.Body).Decode(&share); err != nil {
		t.Fatal(err)
	}

	recipientHandler := New(store, func(*http.Request) (uuid.UUID, error) { return recipientID, nil })
	getRequest := httptest.NewRequest(http.MethodGet, "/api/shared-events/"+share.ID.String(), nil)
	getRequest.SetPathValue("shareID", share.ID.String())
	getResponse := httptest.NewRecorder()
	recipientHandler.GetSharedEvent(getResponse, getRequest)
	if getResponse.Code != http.StatusOK {
		t.Fatalf("get shared event: expected status %d, got %d", http.StatusOK, getResponse.Code)
	}
	var shared sharedEventResponse
	if err := json.NewDecoder(getResponse.Body).Decode(&shared); err != nil {
		t.Fatal(err)
	}
	if shared.OriginalTitle != "Original title" || shared.LocalTitle != nil {
		t.Fatalf("unexpected initial shared event: %+v", shared)
	}

	updateRequest := httptest.NewRequest(http.MethodPatch, "/api/shared-events/"+share.ID.String(), bytes.NewBufferString(`{
		"title":"Local title",
		"location":"Local location",
		"custom_info":"Private note"
	}`))
	updateRequest.SetPathValue("shareID", share.ID.String())
	updateResponse := httptest.NewRecorder()
	recipientHandler.UpdateSharedEvent(updateResponse, updateRequest)
	if updateResponse.Code != http.StatusNoContent {
		t.Fatalf("update shared event: expected status %d, got %d", http.StatusNoContent, updateResponse.Code)
	}

	getRequest = httptest.NewRequest(http.MethodGet, "/api/shared-events/"+share.ID.String(), nil)
	getRequest.SetPathValue("shareID", share.ID.String())
	getResponse = httptest.NewRecorder()
	recipientHandler.GetSharedEvent(getResponse, getRequest)
	if getResponse.Code != http.StatusOK {
		t.Fatalf("get updated event: expected status %d, got %d", http.StatusOK, getResponse.Code)
	}
	if err := json.NewDecoder(getResponse.Body).Decode(&shared); err != nil {
		t.Fatal(err)
	}
	if shared.OriginalTitle != "Original title" || shared.LocalTitle == nil || *shared.LocalTitle != "Local title" {
		t.Fatalf("local title override was not preserved: %+v", shared)
	}
	if shared.OriginalLocation == nil || *shared.OriginalLocation != "Original location" || shared.LocalLocation == nil || *shared.LocalLocation != "Local location" {
		t.Fatalf("local location override was not preserved: %+v", shared)
	}
	if shared.PrivateCustomInfo == nil || *shared.PrivateCustomInfo != "Private note" {
		t.Fatalf("private custom info was not preserved: %+v", shared)
	}

	partialUpdate := httptest.NewRequest(http.MethodPatch, "/api/shared-events/"+share.ID.String(), bytes.NewBufferString(`{
		"location":"Updated local location"
	}`))
	partialUpdate.SetPathValue("shareID", share.ID.String())
	partialResponse := httptest.NewRecorder()
	recipientHandler.UpdateSharedEvent(partialResponse, partialUpdate)
	if partialResponse.Code != http.StatusNoContent {
		t.Fatalf("partial update: expected status %d, got %d", http.StatusNoContent, partialResponse.Code)
	}

	getRequest = httptest.NewRequest(http.MethodGet, "/api/shared-events/"+share.ID.String(), nil)
	getRequest.SetPathValue("shareID", share.ID.String())
	getResponse = httptest.NewRecorder()
	recipientHandler.GetSharedEvent(getResponse, getRequest)
	if err := json.NewDecoder(getResponse.Body).Decode(&shared); err != nil {
		t.Fatal(err)
	}
	if shared.LocalTitle == nil || *shared.LocalTitle != "Local title" ||
		shared.LocalLocation == nil || *shared.LocalLocation != "Updated local location" ||
		shared.PrivateCustomInfo == nil || *shared.PrivateCustomInfo != "Private note" {
		t.Fatalf("partial update erased existing overrides: %+v", shared)
	}

	timelineRequest := httptest.NewRequest(http.MethodGet, "/api/timeline", nil)
	timelineResponse := httptest.NewRecorder()
	recipientHandler.ListTimeline(timelineResponse, timelineRequest)
	if timelineResponse.Code != http.StatusOK {
		t.Fatalf("list timeline: expected status %d, got %d", http.StatusOK, timelineResponse.Code)
	}
	var timeline []db.TimelineEntry
	if err := json.NewDecoder(timelineResponse.Body).Decode(&timeline); err != nil {
		t.Fatal(err)
	}
	if len(timeline) != 1 || !timeline[0].IsShared || timeline[0].Title != "Local title" {
		t.Fatalf("unexpected recipient timeline: %+v", timeline)
	}
	if timeline[0].SourceTitle == nil || *timeline[0].SourceTitle != "Original title" {
		t.Fatalf("timeline did not preserve source title: %+v", timeline[0])
	}

	ownerGetRequest := httptest.NewRequest(http.MethodGet, "/api/shared-events/"+share.ID.String(), nil)
	ownerGetRequest.SetPathValue("shareID", share.ID.String())
	ownerGetResponse := httptest.NewRecorder()
	ownerHandler.GetSharedEvent(ownerGetResponse, ownerGetRequest)
	if ownerGetResponse.Code != http.StatusNotFound {
		t.Fatalf("owner should not use recipient shared-event view: expected status %d, got %d", http.StatusNotFound, ownerGetResponse.Code)
	}
}
