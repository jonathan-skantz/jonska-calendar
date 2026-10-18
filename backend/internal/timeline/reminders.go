package timeline

import (
	"net/http"
	"time"

	"github.com/jonathan-skantz/calendar/backend/db"
)

var reminderNotFound = mappedErrorResponse{
	cause:    db.ErrReminderNotFound,
	response: httpErrorResponse{status: http.StatusNotFound, message: "reminder not found"},
}

func (h *Handler) ListReminders(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	reminders, err := h.Store.ListReminders(r.Context(), userID)
	if err != nil {
		http.Error(w, "failed to list reminders", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, reminders)
}

func (h *Handler) CreateReminder(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	eventID, ok := pathID(w, r, "eventID")
	if !ok {
		return
	}
	var request reminderRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	if err := validateReminderRequest(request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	reminderID, err := h.Store.CreateReminder(r.Context(), userID, eventID, request.OffsetSeconds)
	if err != nil {
		writeMappedError(w, err,
			httpErrorResponse{status: http.StatusInternalServerError, message: "failed to create reminder"},
			eventNotFound,
			mappedErrorResponse{
				cause:    db.ErrReminderExists,
				response: httpErrorResponse{status: http.StatusConflict, message: db.ErrReminderExists.Error()},
			},
		)
		return
	}
	writeCreated(w, reminderID)
}

func (h *Handler) DeleteReminder(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	reminderID, ok := pathID(w, r, "reminderID")
	if !ok {
		return
	}
	if err := h.Store.DeleteReminder(r.Context(), userID, reminderID); err != nil {
		writeMappedError(w, err,
			httpErrorResponse{status: http.StatusInternalServerError, message: "failed to delete reminder"},
			reminderNotFound,
		)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) MarkReminderNotified(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	reminderID, ok := pathID(w, r, "reminderID")
	if !ok {
		return
	}
	if err := h.Store.MarkReminderNotified(r.Context(), userID, reminderID, time.Now()); err != nil {
		writeMappedError(w, err,
			httpErrorResponse{status: http.StatusInternalServerError, message: "failed to mark reminder"},
			reminderNotFound,
		)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
