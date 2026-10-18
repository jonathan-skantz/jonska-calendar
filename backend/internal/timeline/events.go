package timeline

import (
	"net/http"

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
	if err := validateEventRequest(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id, err := h.Store.CreateEvent(r.Context(), userID, request.toEvent())
	if err != nil {
		writeMappedError(w, err,
			httpErrorResponse{status: http.StatusInternalServerError, message: "failed to create event"},
			mainTagNotFound,
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
	eventID, ok := pathID(w, r, "eventID")
	if !ok {
		return
	}
	var request eventRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	if err := validateEventRequest(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.Store.UpdateEvent(r.Context(), userID, eventID, request.toEvent()); err != nil {
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
