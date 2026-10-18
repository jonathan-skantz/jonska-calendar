package timeline

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
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
		writeMappedError(w, err,
			httpErrorResponse{status: http.StatusInternalServerError, message: "failed to delete timeline item"},
			mappedErrorResponse{
				cause:    db.ErrTimelineItemNotFound,
				response: httpErrorResponse{status: http.StatusNotFound, message: "timeline item not found"},
			},
		)
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

func writeCreated(w http.ResponseWriter, id uuid.UUID) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(createResponse{ID: id})
}
