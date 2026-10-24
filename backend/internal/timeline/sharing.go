package timeline

import (
	"net/http"

	"github.com/jonathan-skantz/calendar/backend/db"
	"github.com/jonathan-skantz/calendar/backend/internal/auth"
)

// Only the event owner manages shares. Recipients are read-only and see a
// shared event through their timeline, never through these routes.

var (
	eventNotFound = mappedErrorResponse{
		cause:    db.ErrEventNotFound,
		response: httpErrorResponse{status: http.StatusNotFound, message: "event not found"},
	}
	shareNotFound = mappedErrorResponse{
		cause:    db.ErrSharedEventNotFound,
		response: httpErrorResponse{status: http.StatusNotFound, message: "share not found"},
	}
)

func (h *Handler) ListEventShares(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	eventID, ok := pathID(w, r, "eventID")
	if !ok {
		return
	}
	shares, err := h.Store.ListEventShares(r.Context(), ownerID, eventID)
	if err != nil {
		writeMappedError(w, err,
			httpErrorResponse{status: http.StatusInternalServerError, message: "failed to list shares"},
			eventNotFound,
		)
		return
	}
	writeJSON(w, http.StatusOK, shares)
}

func (h *Handler) ShareEvent(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := h.requireUser(w, r)
	if !ok || auth.RejectDemo(w, r, h.Store, ownerID) {
		return
	}
	eventID, ok := pathID(w, r, "eventID")
	if !ok {
		return
	}
	var request shareRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	fields, err := normalizeShareFields(request.shareFieldsRequest)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	recipient, ok := h.findUserByEmail(w, r, request.RecipientEmail, "recipient not found")
	if !ok {
		return
	}
	if recipient.ID == ownerID {
		http.Error(w, "you cannot share an event with yourself", http.StatusBadRequest)
		return
	}
	shareID, err := h.Store.ShareEvent(r.Context(), ownerID, eventID, recipient.ID, fields)
	if err != nil {
		writeMappedError(w, err,
			httpErrorResponse{status: http.StatusInternalServerError, message: "failed to share event"},
			eventNotFound,
			mappedErrorResponse{
				cause:    db.ErrAlreadyShared,
				response: httpErrorResponse{status: http.StatusConflict, message: db.ErrAlreadyShared.Error()},
			},
		)
		return
	}
	writeCreated(w, shareID)
}

func (h *Handler) UpdateEventShare(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	eventID, ok := pathID(w, r, "eventID")
	if !ok {
		return
	}
	shareID, ok := pathID(w, r, "shareID")
	if !ok {
		return
	}
	var request shareFieldsRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	fields, err := normalizeShareFields(request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.Store.UpdateEventShare(r.Context(), ownerID, eventID, shareID, fields); err != nil {
		writeMappedError(w, err,
			httpErrorResponse{status: http.StatusInternalServerError, message: "failed to update share"},
			shareNotFound,
		)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) DeleteEventShare(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	eventID, ok := pathID(w, r, "eventID")
	if !ok {
		return
	}
	shareID, ok := pathID(w, r, "shareID")
	if !ok {
		return
	}
	if err := h.Store.DeleteEventShare(r.Context(), ownerID, eventID, shareID); err != nil {
		writeMappedError(w, err,
			httpErrorResponse{status: http.StatusInternalServerError, message: "failed to remove share"},
			shareNotFound,
		)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
