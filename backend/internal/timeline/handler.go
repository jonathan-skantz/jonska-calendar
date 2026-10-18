package timeline

import (
	"net/http"
	"strings"

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

// findUserByEmail finds another user by the email they signed up with. On
// failure it answers 400 for an empty email or 404 with notFoundMessage, and
// returns false.
func (h *Handler) findUserByEmail(w http.ResponseWriter, r *http.Request, rawEmail, notFoundMessage string) (db.User, bool) {
	email := strings.ToLower(strings.TrimSpace(rawEmail))
	if email == "" {
		http.Error(w, ErrInvalidRequest.Error(), http.StatusBadRequest)
		return db.User{}, false
	}
	user, err := h.Store.FindUserByEmail(r.Context(), email)
	if err != nil {
		http.Error(w, notFoundMessage, http.StatusNotFound)
		return db.User{}, false
	}
	return user, true
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
	itemID, ok := pathID(w, r, "itemID")
	if !ok {
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
	writeJSON(w, http.StatusOK, entries)
}
