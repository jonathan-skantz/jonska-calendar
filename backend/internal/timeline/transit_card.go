package timeline

import (
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
		writeServerError(w, "failed to list transit card days", err)
		return
	}
	household, err := h.Store.GetHousehold(r.Context(), userID)
	if err != nil {
		writeServerError(w, "failed to list transit card days", err)
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
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) SetTransitCardDays(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	var request transitCardDaysRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	submission, err := validateTransitCardDaysRequest(request, time.Now().UTC().Truncate(24*time.Hour))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.Store.SetTransitCardDays(r.Context(), userID, submission.days.from, submission.days.to, submission.needed, submission.maybe); err != nil {
		writeServerError(w, "failed to save transit card days", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
