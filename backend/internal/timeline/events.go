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
		writeRecurrenceError(w, err)
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

// updateSeries saves an event that is or becomes a series, or stops being
// one. Exceptions that are no longer occurrences of the new rule are dropped.
func (h *Handler) updateSeries(r *http.Request, userID, eventID uuid.UUID, event db.Event,
	request eventRequest, exceptions []db.OccurrenceException) error {
	withRule, err := applyRecurrence(event, request.Recurrence, request.TimeZone, nil)
	if err != nil {
		return err
	}
	kept, err := matchingExceptions(withRule, exceptions)
	if err != nil {
		return err
	}
	// The series' end depends on the exceptions it keeps.
	withRule, err = applyRecurrence(event, request.Recurrence, request.TimeZone, kept)
	if err != nil {
		return err
	}
	return h.Store.UpdateSeries(r.Context(), userID, eventID, withRule, kept)
}

func writeRecurrenceError(w http.ResponseWriter, err error) {
	if isRecurrenceError(err) {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeServerError(w, "failed to save the repeat", err)
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
	existing, err := h.Store.GetOwnEvent(r.Context(), userID, eventID)
	if errors.Is(err, db.ErrEventNotFound) {
		http.Error(w, "event not found", http.StatusNotFound)
		return
	}
	if err != nil {
		writeServerError(w, "failed to update event", err)
		return
	}
	if existing.RecurrenceRule == nil && request.Recurrence == nil {
		err = h.Store.UpdateEvent(r.Context(), userID, eventID, event)
	} else {
		err = h.updateSeries(r, userID, eventID, event, request, existing.Exceptions)
		if isRecurrenceError(err) {
			writeRecurrenceError(w, err)
			return
		}
	}
	if err != nil {
		writeMappedError(w, err,
			httpErrorResponse{status: http.StatusInternalServerError, message: "failed to update event"},
			mainTagNotFound,
			mappedErrorResponse{
				cause:    db.ErrEventNotFound,
				response: httpErrorResponse{status: http.StatusNotFound, message: "event not found"},
			},
		)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
