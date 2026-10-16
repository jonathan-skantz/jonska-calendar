package timeline

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/jonathan-skantz/calendar/backend/db"
)

const (
	transitCardResolved = "resolved"
	// transitCardConflict is reserved for when several users need the shared
	// card on the same day. Only the requesting user's days are listed today,
	// so it is never returned yet.
	transitCardConflict = "conflict"
)

func transitCardStatus(db.TransitCardDay) string {
	return transitCardResolved
}

func (h *Handler) ListTransitCardDays(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	days, err := parseDateRange(r.URL.Query().Get("from"), r.URL.Query().Get("to"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	reported, err := h.Store.ListTransitCardDays(r.Context(), userID, days.from, days.to)
	if err != nil {
		http.Error(w, "failed to list transit card days", http.StatusInternalServerError)
		return
	}
	response := make([]transitCardDayResponse, 0, len(reported))
	for _, day := range reported {
		response = append(response, transitCardDayResponse{
			Date:        day.Day.Format(dateLayout),
			UserID:      day.UserID,
			DisplayName: day.DisplayName,
			Need:        day.Need,
			Status:      transitCardStatus(day),
		})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

func (h *Handler) SetTransitCardDays(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	var request transitCardDaysRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, ErrInvalidRequest.Error(), http.StatusBadRequest)
		return
	}
	submission, err := validateTransitCardDaysRequest(request, time.Now().UTC().Truncate(24*time.Hour))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.Store.SetTransitCardDays(r.Context(), userID, submission.days.from, submission.days.to, submission.needed, submission.maybe); err != nil {
		http.Error(w, "failed to save transit card days", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
