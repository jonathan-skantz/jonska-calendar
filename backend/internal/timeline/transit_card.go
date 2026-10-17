package timeline

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/jonathan-skantz/calendar/backend/db"
)

const (
	transitCardResolved = "resolved"
	transitCardConflict = "conflict"
)

// transitCardStatuses marks every "yes" on a date as a conflict when more
// household members said yes than the household has cards. Without a
// household cardCount is 0 and nothing is ever a conflict.
func transitCardStatuses(days []db.TransitCardDay, cardCount int) []string {
	yesByDate := map[time.Time]int{}
	for _, day := range days {
		if day.Need == "yes" {
			yesByDate[day.Day]++
		}
	}
	statuses := make([]string, len(days))
	for i, day := range days {
		statuses[i] = transitCardResolved
		if cardCount > 0 && day.Need == "yes" && yesByDate[day.Day] > cardCount {
			statuses[i] = transitCardConflict
		}
	}
	return statuses
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
	household, err := h.Store.GetHousehold(r.Context(), userID)
	if err != nil {
		http.Error(w, "failed to list transit card days", http.StatusInternalServerError)
		return
	}
	cardCount := 0
	if household != nil {
		cardCount = household.TransitCardCount
	}
	statuses := transitCardStatuses(reported, cardCount)
	response := make([]transitCardDayResponse, 0, len(reported))
	for i, day := range reported {
		response = append(response, transitCardDayResponse{
			Date:        day.Day.Format(dateLayout),
			UserID:      day.UserID,
			DisplayName: day.DisplayName,
			Need:        day.Need,
			Status:      statuses[i],
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
