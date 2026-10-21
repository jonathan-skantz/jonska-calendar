package timeline

import (
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jonathan-skantz/calendar/backend/db"
	"github.com/jonathan-skantz/calendar/backend/internal/auth"
)

type Handler struct {
	Store       *db.Store
	CurrentUser auth.CurrentUser
}

func New(store *db.Store, currentUser auth.CurrentUser) *Handler {
	return &Handler{Store: store, CurrentUser: currentUser}
}

// findUserByEmail finds another user by the email they signed up with. On
// failure it answers 400 for an empty email, 404 with notFoundMessage for an
// unknown one, or 500, and returns false.
func (h *Handler) findUserByEmail(w http.ResponseWriter, r *http.Request, rawEmail, notFoundMessage string) (db.User, bool) {
	email := strings.ToLower(strings.TrimSpace(rawEmail))
	if email == "" {
		http.Error(w, ErrInvalidRequest.Error(), http.StatusBadRequest)
		return db.User{}, false
	}
	user, err := h.Store.FindUserByEmail(r.Context(), email)
	if errors.Is(err, db.ErrUserNotFound) {
		http.Error(w, notFoundMessage, http.StatusNotFound)
		return db.User{}, false
	}
	if err != nil {
		writeServerError(w, "failed to find user", err)
		return db.User{}, false
	}
	return user, true
}

func (h *Handler) requireUser(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	return auth.RequireUser(w, r, h.CurrentUser)
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

	query := r.URL.Query()
	// Until the frontend loads by range, a request without one gets every
	// item, with repeating events as their first occurrence.
	if !query.Has("from") && !query.Has("to") {
		entries, err := h.Store.ListTimeline(r.Context(), userID)
		if err != nil {
			writeServerError(w, "failed to list timeline", err)
			return
		}
		writeJSON(w, http.StatusOK, entries)
		return
	}
	from, to, err := parseTimelineRange(query.Get("from"), query.Get("to"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	entries, err := h.Store.ListTimelineRange(r.Context(), userID, from, to)
	if err != nil {
		writeServerError(w, "failed to list timeline", err)
		return
	}
	items, err := ExpandTimeline(entries, from, to)
	if err != nil {
		writeServerError(w, "failed to list timeline", err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}
