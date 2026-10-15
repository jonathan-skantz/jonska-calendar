package timeline

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/jonathan-skantz/calendar/backend/db"
)

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
