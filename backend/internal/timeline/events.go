package timeline

import (
	"encoding/json"
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
	if err := validateEventRequest(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id, err := h.Store.CreateEvent(r.Context(), userID, db.Event{
		Title:                request.Title,
		Description:          request.Description,
		URL:                  request.URL,
		StartsAt:             request.StartsAt,
		EndsAt:               request.EndsAt,
		IsAllDay:             request.IsAllDay,
		StartIsApproximate:   request.StartIsApproximate,
		EndIsApproximate:     request.EndIsApproximate,
		Kind:                 request.Kind,
		IsSkipped:            request.IsSkipped,
		Location:             request.Location,
		GatheringSeconds:     request.GatheringSeconds,
		GatheringIsOptional:  request.GatheringIsOptional,
		GatheringLabel:       request.GatheringLabel,
		CommuteBeforeSeconds: request.CommuteBeforeSeconds,
		CommuteAfterSeconds:  request.CommuteAfterSeconds,
		MainTagID:            request.MainTagID,
	})
	if err != nil {
		writeMappedError(w, err,
			httpErrorResponse{status: http.StatusInternalServerError, message: "failed to create event"},
			mappedErrorResponse{
				cause:    db.ErrTagNotFound,
				response: httpErrorResponse{status: http.StatusNotFound, message: "main tag not found"},
			},
		)
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
	if err := validateEventRequest(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.Store.UpdateEvent(r.Context(), userID, eventID, db.Event{
		Title: request.Title, Description: request.Description, URL: request.URL,
		StartsAt: request.StartsAt, EndsAt: request.EndsAt, Kind: request.Kind,
		IsSkipped:          request.IsSkipped,
		IsAllDay:           request.IsAllDay,
		StartIsApproximate: request.StartIsApproximate, EndIsApproximate: request.EndIsApproximate,
		Location: request.Location, GatheringSeconds: request.GatheringSeconds,
		GatheringIsOptional: request.GatheringIsOptional, GatheringLabel: request.GatheringLabel,
		CommuteBeforeSeconds: request.CommuteBeforeSeconds,
		CommuteAfterSeconds:  request.CommuteAfterSeconds,
		MainTagID:            request.MainTagID,
	}); err != nil {
		writeMappedError(w, err,
			httpErrorResponse{status: http.StatusInternalServerError, message: "failed to update event"},
			mappedErrorResponse{
				cause:    db.ErrTagNotFound,
				response: httpErrorResponse{status: http.StatusNotFound, message: "main tag not found"},
			},
			mappedErrorResponse{
				cause:    pgx.ErrNoRows,
				response: httpErrorResponse{status: http.StatusNotFound, message: "event not found"},
			},
		)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
