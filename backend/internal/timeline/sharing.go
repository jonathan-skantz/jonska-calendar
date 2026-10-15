package timeline

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/jonathan-skantz/calendar/backend/db"
)

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
