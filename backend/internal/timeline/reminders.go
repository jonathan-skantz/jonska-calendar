package timeline

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"slices"
	"time"

	"github.com/jonathan-skantz/calendar/backend/db"
)

// reminderHorizon is how far ahead the reminders of repeating events are
// listed: occurrences that start within it. Later ones are listed when the
// client loads the reminders again.
const reminderHorizon = 31 * 24 * time.Hour

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
		writeServerError(w, "failed to list reminders", err)
		return
	}
	expanded, err := expandReminders(reminders, time.Now())
	if err != nil {
		writeServerError(w, "failed to list reminders", err)
		return
	}
	writeJSON(w, http.StatusOK, expanded)
}

// expandReminders lists a repeating event's reminder once for each of its
// occurrences that has not ended and starts within reminderHorizon, in the
// order the reminders are due.
func expandReminders(reminders []db.Reminder, now time.Time) ([]db.Reminder, error) {
	expanded := make([]db.Reminder, 0, len(reminders))
	for _, reminder := range reminders {
		if reminder.RecurrenceRule == nil {
			expanded = append(expanded, reminder)
			continue
		}
		series, err := seriesOf(db.TimelineEntry{
			StartsAt: &reminder.EventStartsAt, EndsAt: &reminder.EventEndsAt, IsAllDay: reminder.EventIsAllDay,
			RecurrenceRule: reminder.RecurrenceRule, RecurrenceTimeZone: reminder.RecurrenceTimeZone,
		})
		if err != nil {
			return nil, err
		}
		occurrences, err := series.Occurrences(exceptionsOf(reminder.Exceptions), now, now.Add(reminderHorizon))
		if err != nil {
			return nil, err
		}
		for _, occurrence := range occurrences {
			listed := reminder
			listed.EventStartsAt, listed.EventEndsAt = occurrence.StartsAt, occurrence.EndsAt
			listed.OccurrenceStart = occurrence.StartsAt
			listed.RemindAt = occurrence.StartsAt.Add(-time.Duration(reminder.OffsetSeconds) * time.Second)
			listed.NotifiedAt = nil
			if notifiedAt, ok := reminder.Deliveries[occurrence.StartsAt.UTC()]; ok {
				listed.NotifiedAt = &notifiedAt
			}
			// Exceptions carry no skips for recipients.
			if occurrence.IsSkipped != nil {
				listed.EventIsSkipped = *occurrence.IsSkipped
			}
			expanded = append(expanded, listed)
		}
	}
	slices.SortStableFunc(expanded, func(a, b db.Reminder) int { return a.RemindAt.Compare(b.RemindAt) })
	return expanded, nil
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
	// The occurrence is optional: without it, the event's own start.
	var request reminderNotifiedRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil && !errors.Is(err, io.EOF) {
		http.Error(w, ErrInvalidRequest.Error(), http.StatusBadRequest)
		return
	}
	if err := h.Store.MarkReminderNotified(r.Context(), userID, reminderID, request.OccurrenceStart, time.Now()); err != nil {
		writeMappedError(w, err,
			httpErrorResponse{status: http.StatusInternalServerError, message: "failed to mark reminder"},
			reminderNotFound,
		)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
