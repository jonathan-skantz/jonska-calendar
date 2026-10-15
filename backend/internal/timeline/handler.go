package timeline

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/jonathan-skantz/calendar/backend/db"
)

type CurrentUser func(*http.Request) (uuid.UUID, error)

type Handler struct {
	Store       *db.Store
	CurrentUser CurrentUser
}

func New(store *db.Store, currentUser CurrentUser) *Handler {
	return &Handler{Store: store, CurrentUser: currentUser}
}

func (h *Handler) requireUser(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	userID, err := h.CurrentUser(r)
	if err != nil {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return uuid.Nil, false
	}
	return userID, true
}

func (h *Handler) CreateEvent(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUser(w, r)
	if !ok {
		return
	}

	var request eventRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, ErrInvalidRequest.Error(), http.StatusBadRequest)
		return
	}
	if err := validateEventRequest(request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id, err := h.Store.CreateEvent(r.Context(), userID, db.Event{
		Title:                request.Title,
		Description:          request.Description,
		URL:                  request.URL,
		StartsAt:             request.StartsAt,
		EndsAt:               request.EndsAt,
		Kind:                 request.Kind,
		Location:             request.Location,
		CommuteBeforeSeconds: request.CommuteBeforeSeconds,
		CommuteAfterSeconds:  request.CommuteAfterSeconds,
		MainTagID:            request.MainTagID,
	})
	if err != nil {
		if errors.Is(err, db.ErrTagNotFound) {
			http.Error(w, "main tag not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to create event", http.StatusInternalServerError)
		return
	}
	writeCreated(w, id)
}

func (h *Handler) UpdateEvent(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	eventID, err := uuid.Parse(r.PathValue("eventID"))
	if err != nil {
		http.Error(w, "invalid event id", http.StatusBadRequest)
		return
	}
	var request eventRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, ErrInvalidRequest.Error(), http.StatusBadRequest)
		return
	}
	if err := validateEventRequest(request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.Store.UpdateEvent(r.Context(), userID, eventID, db.Event{
		Title: request.Title, Description: request.Description, URL: request.URL,
		StartsAt: request.StartsAt, EndsAt: request.EndsAt, Kind: request.Kind,
		Location: request.Location, CommuteBeforeSeconds: request.CommuteBeforeSeconds,
		CommuteAfterSeconds: request.CommuteAfterSeconds,
		MainTagID:           request.MainTagID,
	}); err != nil {
		if errors.Is(err, db.ErrTagNotFound) {
			http.Error(w, "main tag not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "event not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to update event", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) CreateTask(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUser(w, r)
	if !ok {
		return
	}

	var request taskRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, ErrInvalidRequest.Error(), http.StatusBadRequest)
		return
	}
	if err := validateTaskRequest(request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id, err := h.Store.CreateTask(r.Context(), userID, db.Task{
		Title:       request.Title,
		Description: request.Description,
		URL:         request.URL,
		DeadlineAt:  request.DeadlineAt,
	})
	if err != nil {
		http.Error(w, "failed to create task", http.StatusInternalServerError)
		return
	}
	writeCreated(w, id)
}

func (h *Handler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	taskID, err := uuid.Parse(r.PathValue("taskID"))
	if err != nil {
		http.Error(w, "invalid task id", http.StatusBadRequest)
		return
	}
	var request taskRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, ErrInvalidRequest.Error(), http.StatusBadRequest)
		return
	}
	if err := validateTaskRequest(request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.Store.UpdateTask(r.Context(), userID, taskID, db.Task{
		Title: request.Title, Description: request.Description, URL: request.URL,
		DeadlineAt: request.DeadlineAt,
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "task not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to update task", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) SetTaskCompleted(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	taskID, err := uuid.Parse(r.PathValue("taskID"))
	if err != nil {
		http.Error(w, "invalid task id", http.StatusBadRequest)
		return
	}
	var request taskCompletionRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, ErrInvalidRequest.Error(), http.StatusBadRequest)
		return
	}
	if err := h.Store.SetTaskCompleted(r.Context(), userID, taskID, request.IsCompleted); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "task not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to update task", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) DeleteTimelineItem(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	itemID, err := uuid.Parse(r.PathValue("itemID"))
	if err != nil {
		http.Error(w, "invalid item id", http.StatusBadRequest)
		return
	}
	if err := h.Store.DeleteTimelineItem(r.Context(), userID, itemID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "timeline item not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to delete timeline item", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ListTimeline(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUser(w, r)
	if !ok {
		return
	}

	entries, err := h.Store.ListTimeline(r.Context(), userID)
	if err != nil {
		http.Error(w, "failed to list timeline", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(entries)
}

func (h *Handler) CreateTag(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	var request tagRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, ErrInvalidRequest.Error(), http.StatusBadRequest)
		return
	}
	if err := validateTagName(request.Name); err != nil {
		http.Error(w, ErrInvalidRequest.Error(), http.StatusBadRequest)
		return
	}
	tag, err := h.Store.CreateTag(r.Context(), userID, strings.TrimSpace(request.Name))
	if errors.Is(err, db.ErrTagAlreadyExists) {
		http.Error(w, "tag already exists", http.StatusConflict)
		return
	}
	if err != nil {
		http.Error(w, "failed to create tag", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(tag)
}

func (h *Handler) ListTags(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	tags, err := h.Store.ListTags(r.Context(), userID)
	if err != nil {
		http.Error(w, "failed to list tags", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(tags)
}

func (h *Handler) SetDefaultTag(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	var request tagSelectionRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, ErrInvalidRequest.Error(), http.StatusBadRequest)
		return
	}
	if err := validateTagSelectionRequest(request); err != nil {
		http.Error(w, ErrInvalidRequest.Error(), http.StatusBadRequest)
		return
	}
	if err := h.Store.SetDefaultTag(r.Context(), userID, request.TagID); err != nil {
		http.Error(w, "tag not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) RenameTag(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	tagID, err := uuid.Parse(r.PathValue("tagID"))
	if err != nil {
		http.Error(w, "invalid tag id", http.StatusBadRequest)
		return
	}
	var request tagUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, ErrInvalidRequest.Error(), http.StatusBadRequest)
		return
	}
	if err := validateTagUpdateRequest(request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.Store.RenameTag(r.Context(), userID, tagID, strings.TrimSpace(request.Name), request.Color); err != nil {
		if errors.Is(err, db.ErrTagAlreadyExists) {
			http.Error(w, "tag already exists", http.StatusConflict)
			return
		}
		http.Error(w, "tag not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) DeleteTag(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	tagID, err := uuid.Parse(r.PathValue("tagID"))
	if err != nil {
		http.Error(w, "invalid tag id", http.StatusBadRequest)
		return
	}
	if err := h.Store.DeleteTag(r.Context(), userID, tagID); err != nil {
		if errors.Is(err, db.ErrLastTag) {
			http.Error(w, "at least one tag is required", http.StatusConflict)
			return
		}
		http.Error(w, "tag not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) AddTagToItem(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	itemID, err := uuid.Parse(r.PathValue("itemID"))
	if err != nil {
		http.Error(w, "invalid item id", http.StatusBadRequest)
		return
	}
	var request tagSelectionRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, ErrInvalidRequest.Error(), http.StatusBadRequest)
		return
	}
	if err := validateTagSelectionRequest(request); err != nil {
		http.Error(w, ErrInvalidRequest.Error(), http.StatusBadRequest)
		return
	}
	if err := h.Store.AddTagToItem(r.Context(), userID, itemID, request.TagID); err != nil {
		http.Error(w, "item or tag not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) RemoveTagFromItem(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	itemID, err := uuid.Parse(r.PathValue("itemID"))
	if err != nil {
		http.Error(w, "invalid item id", http.StatusBadRequest)
		return
	}
	tagID, err := uuid.Parse(r.PathValue("tagID"))
	if err != nil {
		http.Error(w, "invalid tag id", http.StatusBadRequest)
		return
	}
	if err := h.Store.RemoveTagFromItem(r.Context(), userID, itemID, tagID); err != nil {
		if errors.Is(err, db.ErrLastTag) {
			http.Error(w, "events must have at least one tag", http.StatusConflict)
			return
		}
		http.Error(w, "item tag not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ShareEvent(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	eventID, err := uuid.Parse(r.PathValue("eventID"))
	if err != nil {
		http.Error(w, "invalid event id", http.StatusBadRequest)
		return
	}
	var request shareRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, ErrInvalidRequest.Error(), http.StatusBadRequest)
		return
	}
	request.RecipientEmail = strings.ToLower(strings.TrimSpace(request.RecipientEmail))
	if request.RecipientEmail == "" {
		http.Error(w, ErrInvalidRequest.Error(), http.StatusBadRequest)
		return
	}
	recipient, err := h.Store.FindUserByEmail(r.Context(), request.RecipientEmail)
	if err != nil {
		http.Error(w, "recipient not found", http.StatusNotFound)
		return
	}
	shareID, err := h.Store.ShareEvent(r.Context(), ownerID, eventID, recipient.ID)
	if err != nil {
		http.Error(w, "event cannot be shared", http.StatusBadRequest)
		return
	}
	writeCreated(w, shareID)
}

func (h *Handler) GetSharedEvent(w http.ResponseWriter, r *http.Request) {
	recipientID, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	shareID, err := uuid.Parse(r.PathValue("shareID"))
	if err != nil {
		http.Error(w, "invalid share id", http.StatusBadRequest)
		return
	}
	event, err := h.Store.FindSharedEvent(r.Context(), recipientID, shareID)
	if err != nil {
		http.Error(w, "shared event not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(toSharedEventResponse(event))
}

func (h *Handler) UpdateSharedEvent(w http.ResponseWriter, r *http.Request) {
	recipientID, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	shareID, err := uuid.Parse(r.PathValue("shareID"))
	if err != nil {
		http.Error(w, "invalid share id", http.StatusBadRequest)
		return
	}
	var request sharedEventOverrideRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, ErrInvalidRequest.Error(), http.StatusBadRequest)
		return
	}
	existing, err := h.Store.FindSharedEvent(r.Context(), recipientID, shareID)
	if err != nil {
		http.Error(w, "shared event not found", http.StatusNotFound)
		return
	}

	// Merge only fields present in the PATCH body so editing one local field
	// does not erase the recipient's other overrides or private note.
	if request.Title != nil {
		existing.LocalTitle = request.Title
	}
	if request.Description != nil {
		existing.LocalDescription = request.Description
	}
	if request.URL != nil {
		existing.LocalURL = request.URL
	}
	if request.Location != nil {
		existing.LocalLocation = request.Location
	}
	if request.CustomInfo != nil {
		existing.PrivateCustomInfo = request.CustomInfo
	}

	if err := h.Store.UpdateSharedEventOverride(r.Context(), recipientID, shareID, existing); err != nil {
		http.Error(w, "shared event not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func toSharedEventResponse(event db.SharedEvent) sharedEventResponse {
	return sharedEventResponse{
		ShareID:             event.ShareID,
		EventID:             event.EventID,
		OriginalTitle:       event.OriginalTitle,
		OriginalDescription: event.OriginalDescription,
		OriginalURL:         event.OriginalURL,
		OriginalLocation:    event.OriginalLocation,
		StartsAt:            event.StartsAt,
		EndsAt:              event.EndsAt,
		Kind:                event.Kind,
		LocalTitle:          event.LocalTitle,
		LocalDescription:    event.LocalDescription,
		LocalURL:            event.LocalURL,
		LocalLocation:       event.LocalLocation,
		PrivateCustomInfo:   event.PrivateCustomInfo,
	}
}

func writeCreated(w http.ResponseWriter, id uuid.UUID) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(createResponse{ID: id})
}
