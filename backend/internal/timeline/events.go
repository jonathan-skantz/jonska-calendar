package timeline

import (
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/jonathan-skantz/calendar/backend/db"
)

var mainTagNotFound = mappedErrorResponse{
	cause:    db.ErrTagNotFound,
	response: httpErrorResponse{status: http.StatusNotFound, message: "main tag not found"},
}

func (h *Handler) CreateEvent(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUser(w, r)
	if !ok {
		return
	}

	var request eventRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	event, err := ValidateEvent(request.toEvent())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	event, err = applyRecurrence(event, request.Recurrence, request.TimeZone, nil)
	if err != nil {
		writeEventError(w, err, "failed to create event")
		return
	}

	id, err := h.Store.CreateEvent(r.Context(), userID, event)
	if err != nil {
		writeMappedError(w, err,
			httpErrorResponse{status: http.StatusInternalServerError, message: "failed to create event"},
			mainTagNotFound,
		)
		return
	}
	writeCreated(w, id)
}

// writeEventError answers an error from saving or deleting an event or some
// of its occurrences.
func writeEventError(w http.ResponseWriter, err error, message string) {
	switch {
	case isRecurrenceError(err), errors.Is(err, ErrInvalidScope), errors.Is(err, ErrNotRepeating),
		errors.Is(err, ErrOccurrenceFields):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, ErrOccurrenceNotFound):
		http.Error(w, err.Error(), http.StatusNotFound)
	default:
		writeMappedError(w, err,
			httpErrorResponse{status: http.StatusInternalServerError, message: message},
			mainTagNotFound,
			mappedErrorResponse{
				cause:    db.ErrEventNotFound,
				response: httpErrorResponse{status: http.StatusNotFound, message: "event not found"},
			},
			mappedErrorResponse{
				cause:    db.ErrTimelineItemNotFound,
				response: httpErrorResponse{status: http.StatusNotFound, message: "timeline item not found"},
			},
		)
	}
}

func (h *Handler) UpdateEvent(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	eventID, ok := pathID(w, r, "eventID")
	if !ok {
		return
	}
	var request eventRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	event, err := ValidateEvent(request.toEvent())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	scope, err := parseScope(r.URL.Query())
	if err != nil {
		writeEventError(w, err, "failed to update event")
		return
	}
	existing, err := h.Store.GetOwnEvent(r.Context(), userID, eventID)
	if err != nil {
		writeEventError(w, err, "failed to update event")
		return
	}
	if scope.scope != "" {
		newID, err := h.updateOccurrences(r, userID, existing, event, request, scope)
		if err != nil {
			writeEventError(w, err, "failed to update event")
			return
		}
		if newID != uuid.Nil {
			// Changing the following occurrences starts a new series.
			writeJSON(w, http.StatusOK, createResponse{ID: newID})
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if existing.RecurrenceRule == nil && request.Recurrence == nil {
		err = h.Store.UpdateEvent(r.Context(), userID, eventID, event)
	} else {
		var kept []db.OccurrenceException
		event, kept, err = WithSeries(event, request.Recurrence, request.TimeZone, existing.Exceptions)
		if err == nil {
			err = h.Store.UpdateSeries(r.Context(), userID, eventID, event, kept)
		}
	}
	if err != nil {
		writeEventError(w, err, "failed to update event")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
