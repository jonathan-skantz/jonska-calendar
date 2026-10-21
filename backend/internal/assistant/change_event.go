package assistant

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jonathan-skantz/calendar/backend/db"
)

const confirmNote = "Waiting for the user to apply or decline it; nothing has changed yet."

const moveEventDescription = "Ask to move one of the user's own events to a new date or time. " +
	"The user confirms it before anything changes. Events with shared_by belong to someone else, and repeating events are changed in the app, so neither can be moved here."

type moveEventInput struct {
	EventID   string `json:"event_id" jsonschema:"required" jsonschema_description:"id from list_events"`
	Date      string `json:"date" jsonschema:"required" jsonschema_description:"New first day, YYYY-MM-DD"`
	StartTime string `json:"start_time,omitempty" jsonschema_description:"New start, HH:MM, 24-hour; leave out to keep the time of day. Not for all-day events"`
	EndTime   string `json:"end_time,omitempty" jsonschema_description:"New end, HH:MM, 24-hour; leave out to keep the duration. Not for all-day events"`
}

const skipEventDescription = "Ask to mark one of the user's own attended events as skipped, or as attended again. " +
	"A skipped event stays in the calendar but no longer takes up time. The user confirms it before anything changes."

type skipEventInput struct {
	EventID string `json:"event_id" jsonschema:"required" jsonschema_description:"id from list_events"`
	Skipped bool   `json:"skipped" jsonschema:"required" jsonschema_description:"true to skip, false to attend again"`
}

const deleteEventDescription = "Ask to delete one of the user's own events. The user confirms it before anything changes."

type deleteEventInput struct {
	EventID string `json:"event_id" jsonschema:"required" jsonschema_description:"id from list_events"`
}

func (t *turn) moveEvent(ctx context.Context, input moveEventInput) (any, string, error) {
	return t.queue(ctx, input.EventID, kindMove, func(event db.Event) (time.Time, time.Time, error) {
		return t.movedTimes(event, input)
	})
}

func (t *turn) skipEvent(ctx context.Context, input skipEventInput) (any, string, error) {
	kind := kindUnskip
	if input.Skipped {
		kind = kindSkip
	}
	return t.queue(ctx, input.EventID, kind, nil)
}

func (t *turn) deleteEvent(ctx context.Context, input deleteEventInput) (any, string, error) {
	return t.queue(ctx, input.EventID, kindDelete, nil)
}

// queue adds a pending action on one of the user's own events. times gives
// the event's new times; without it they stay. The change is checked now so
// the model hears about problems, and again when the user applies it.
func (t *turn) queue(ctx context.Context, rawID, kind string, times func(db.Event) (time.Time, time.Time, error)) (any, string, error) {
	if err := t.canAct(); err != nil {
		return nil, "", err
	}
	id, err := eventID(rawID)
	if err != nil {
		return nil, "", err
	}
	entry, err := t.ownEvent(ctx, id)
	if err != nil {
		return nil, "", err
	}

	event := entry.Event()
	action := Action{
		ID: uuid.New(), Kind: kind, Status: statusPending, EventID: id, Title: event.Title,
		StartsAt: event.StartsAt, EndsAt: event.EndsAt, IsAllDay: event.IsAllDay,
	}
	if times != nil {
		if action.StartsAt, action.EndsAt, err = times(event); err != nil {
			return nil, "", err
		}
	}
	if kind != kindDelete {
		if _, err := change(event, action, t.now); err != nil {
			return nil, "", err
		}
	}

	t.actions = append(t.actions, action)
	return map[string]any{"queued": true, "note": confirmNote}, describe(action, t.location), nil
}

// movedTimes is where input moves event to. An all-day event keeps its
// number of days; a timed one keeps its time of day and duration unless
// input changes them.
func (t *turn) movedTimes(event db.Event, input moveEventInput) (time.Time, time.Time, error) {
	day, err := t.date(input.Date, "date")
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	if event.IsAllDay {
		if input.StartTime != "" || input.EndTime != "" {
			return time.Time{}, time.Time{}, errors.New("all-day events move by date only")
		}
		return t.allDayRange(day, calendarDays(event.StartsAt.In(t.location), event.EndsAt.In(t.location)))
	}
	startClock := input.StartTime
	if startClock == "" {
		startClock = event.StartsAt.In(t.location).Format(clockLayout)
	}
	return t.timedRange(day, startClock, input.EndTime, event.EndsAt.Sub(event.StartsAt))
}

// calendarDays counts the days from start to end, both local midnights.
// Rounding absorbs a daylight saving change in between.
func calendarDays(start, end time.Time) int {
	return int((end.Sub(start) + 12*time.Hour) / (24 * time.Hour))
}

// describe summarizes an action, for the user and in the agent's context.
func describe(action Action, location *time.Location) string {
	switch action.Kind {
	case kindMove:
		return fmt.Sprintf("Asked to move %q to %s", action.Title, when(action, location))
	case kindSkip:
		return fmt.Sprintf("Asked to skip %q", action.Title)
	case kindUnskip:
		return fmt.Sprintf("Asked to attend %q again", action.Title)
	case kindDelete:
		return fmt.Sprintf("Asked to delete %q", action.Title)
	}
	return fmt.Sprintf("Created %q, %s", action.Title, when(action, location))
}

func when(action Action, location *time.Location) string {
	start := action.StartsAt.In(location)
	if action.IsAllDay {
		return start.Format(dayFormat)
	}
	return start.Format(timeFormat) + "–" + action.EndsAt.In(location).Format(clockLayout)
}
