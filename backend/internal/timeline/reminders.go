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
// order the reminders are due. A recipient's customization moves the
// reminder with the event; the shifted start identifies its delivery, so
// changing the shift arms it again, as moving the event does.
func expandReminders(reminders []db.Reminder, now time.Time) ([]db.Reminder, error) {
	// A shift may move an occurrence that has just ended back into the
	// listing, so series are expanded that much earlier.
	margin := db.MaxShiftSeconds * time.Second
	expanded := make([]db.Reminder, 0, len(reminders))
	for _, reminder := range reminders {
		listed := []db.Reminder{reminder}
		if reminder.RecurrenceRule != nil {
			var err error
			if listed, err = reminderOccurrences(reminder, now.Add(-margin), now.Add(reminderHorizon)); err != nil {
				return nil, err
			}
		}
		for _, item := range listed {
			customizeReminder(&item)
			if !item.EventEndsAt.After(now) ||
				(item.RecurrenceRule != nil && !item.EventStartsAt.Before(now.Add(reminderHorizon))) {
				continue
			}
			expanded = append(expanded, item)
		}
	}
	slices.SortStableFunc(expanded, func(a, b db.Reminder) int { return a.RemindAt.Compare(b.RemindAt) })
	return expanded, nil
}

func reminderOccurrences(reminder db.Reminder, from, to time.Time) ([]db.Reminder, error) {
	series, err := seriesOf(db.TimelineEntry{
		StartsAt: &reminder.EventStartsAt, EndsAt: &reminder.EventEndsAt, IsAllDay: reminder.EventIsAllDay,
		RecurrenceRule: reminder.RecurrenceRule, RecurrenceTimeZone: reminder.RecurrenceTimeZone,
	})
	if err != nil {
		return nil, err
	}
	occurrences, err := series.Occurrences(exceptionsOf(reminder.Exceptions), from, to)
	if err != nil {
		return nil, err
	}
	listed := make([]db.Reminder, 0, len(occurrences))
	for _, occurrence := range occurrences {
		item := reminder
		item.EventStartsAt, item.EventEndsAt = occurrence.StartsAt, occurrence.EndsAt
		setReminderOccurrence(&item)
		// Exceptions carry no skips for recipients.
		if occurrence.IsSkipped != nil {
			item.EventIsSkipped = *occurrence.IsSkipped
		}
		listed = append(listed, item)
	}
	return listed, nil
}

// customizeReminder applies the recipient's customization to the event of
// a reminder.
func customizeReminder(reminder *db.Reminder) {
	if reminder.Customization == nil {
		return
	}
	values := ReceivedValues{
		Title: reminder.EventTitle, Location: reminder.EventLocation,
		StartsAt: reminder.EventStartsAt, EndsAt: reminder.EventEndsAt,
	}.customized(*reminder.Customization, reminder.EventIsAllDay)
	reminder.EventTitle, reminder.EventLocation = values.Title, values.Location
	reminder.EventStartsAt, reminder.EventEndsAt = values.StartsAt, values.EndsAt
	setReminderOccurrence(reminder)
}

// setReminderOccurrence identifies the reminder by its event's start and
// finds whether it was delivered for it.
func setReminderOccurrence(reminder *db.Reminder) {
	reminder.OccurrenceStart = reminder.EventStartsAt
	reminder.RemindAt = reminder.EventStartsAt.Add(-time.Duration(reminder.OffsetSeconds) * time.Second)
	reminder.NotifiedAt = nil
	if notifiedAt, ok := reminder.Deliveries[reminder.EventStartsAt.UTC()]; ok {
		reminder.NotifiedAt = &notifiedAt
	}
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
