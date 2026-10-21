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
	scope, err := parseScope(r.URL.Query())
	if err != nil {
		writeEventError(w, err, "failed to delete timeline item")
		return
	}
	// Deleting all occurrences deletes the series.
	if scope.scope == "" || scope.scope == scopeAll {
		err = h.Store.DeleteTimelineItem(r.Context(), userID, itemID)
	} else {
		err = h.deleteOccurrences(r, userID, itemID, scope)
	}
	if err != nil {
		writeEventError(w, err, "failed to delete timeline item")
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
	from, to, err := parseTimelineRange(query.Get("from"), query.Get("to"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	entries, err := h.Store.ListTimeline(r.Context(), userID, from, to)
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
