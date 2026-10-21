package assistant

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jonathan-skantz/calendar/backend/db"
	"github.com/jonathan-skantz/calendar/backend/internal/quickadd"
	"github.com/jonathan-skantz/calendar/backend/internal/timeline"
)

const (
	maxActions        = 5
	maxLocationLength = 200
	maxReminders      = 10
)

const createEventDescription = "Create a new event in the user's calendar right away. The user can undo it or open it to edit. " +
	"Use it once you know the date and time; tags must be the user's existing tag names."

type createEventInput struct {
	Title           string   `json:"title" jsonschema:"required" jsonschema_description:"Short title in the user's language, without the date or time"`
	Date            string   `json:"date" jsonschema:"required" jsonschema_description:"YYYY-MM-DD"`
	StartTime       string   `json:"start_time,omitempty" jsonschema_description:"HH:MM, 24-hour; leave out for all-day events"`
	EndTime         string   `json:"end_time,omitempty" jsonschema_description:"HH:MM, 24-hour, after start_time; leave out for one hour"`
	AllDay          bool     `json:"all_day,omitempty"`
	Location        string   `json:"location,omitempty"`
	Kind            string   `json:"kind,omitempty" jsonschema:"enum=attendable,enum=informational" jsonschema_description:"attendable (the default) if the user takes part, informational for a heads-up"`
	Tags            []string `json:"tags,omitempty" jsonschema_description:"Existing tag names only; the first is the main tag"`
	ReminderMinutes []int    `json:"reminder_minutes,omitempty" jsonschema_description:"Minutes before the start"`
}

func (t *turn) createEvent(ctx context.Context, input createEventInput) (any, string, error) {
	if err := t.canAct(); err != nil {
		return nil, "", err
	}
	event, tagIDs, warnings, err := t.newEvent(input)
	if err != nil {
		return nil, "", err
	}
	reminders, err := reminderMinutes(input.ReminderMinutes)
	if err != nil {
		return nil, "", err
	}
	id, err := t.store.CreateEvent(ctx, t.userID, event)
	if err != nil {
		return nil, "", fmt.Errorf("saving the event failed: %w", err)
	}
	t.loaded = false
	// The event exists now; the extras are best effort and reported.
	warnings = append(warnings, t.addExtras(ctx, id, tagIDs, reminders)...)

	action := Action{
		ID: uuid.New(), Kind: kindCreate, Status: statusDone, EventID: id, Title: event.Title,
		StartsAt: event.StartsAt, EndsAt: event.EndsAt, IsAllDay: event.IsAllDay, Warnings: warnings,
	}
	t.actions = append(t.actions, action)
	result := map[string]any{"created": true, "event_id": id}
	if len(warnings) > 0 {
		result["warnings"] = warnings
	}
	return result, describe(action, t.location), nil
}

// newEvent validates input into an event, with the IDs of its tags, the first
// of which is the main tag. Unknown tag names become warnings.
func (t *turn) newEvent(input createEventInput) (db.Event, []uuid.UUID, []string, error) {
	day, err := t.date(input.Date, "date")
	if err != nil {
		return db.Event{}, nil, nil, err
	}
	var start, end time.Time
	switch {
	case input.AllDay && (input.StartTime != "" || input.EndTime != ""):
		err = errors.New("all-day events have no start_time or end_time")
	case input.AllDay:
		start, end, err = t.allDayRange(day, 1)
	case input.StartTime == "":
		err = errors.New("start_time is required unless all_day is true")
	default:
		start, end, err = t.timedRange(day, input.StartTime, input.EndTime, time.Hour)
	}
	if err != nil {
		return db.Event{}, nil, nil, err
	}

	location := strings.TrimSpace(input.Location)
	if utf8.RuneCountInString(location) > maxLocationLength {
		return db.Event{}, nil, nil, fmt.Errorf("location must be at most %d characters", maxLocationLength)
	}
	event := db.Event{Title: input.Title, StartsAt: start, EndsAt: end, IsAllDay: input.AllDay, Kind: input.Kind}
	if event.Kind == "" {
		event.Kind = db.EventKindAttendable
	}
	if location != "" {
		event.Location = &location
	}
	tagIDs, unknown := quickadd.MatchTags(input.Tags, t.tags)
	if len(tagIDs) > 0 {
		event.MainTagID = tagIDs[0]
	}
	if event, err = timeline.ValidateEvent(event); err != nil {
		return db.Event{}, nil, nil, err
	}

	var warnings []string
	for _, name := range unknown {
		warnings = append(warnings, fmt.Sprintf("No tag named %q", name))
	}
	return event, tagIDs, warnings, nil
}

// addExtras attaches the tags after the main one and the reminders to a new
// event, and returns a warning for each that failed.
func (t *turn) addExtras(ctx context.Context, eventID uuid.UUID, tagIDs []uuid.UUID, reminders []int) []string {
	var warnings []string
	for _, tagID := range tagIDs[min(1, len(tagIDs)):] {
		if err := t.store.AddTagToItem(ctx, t.userID, eventID, tagID); err != nil {
			warnings = append(warnings, "Couldn't add a tag")
		}
	}
	for _, minutes := range reminders {
		if _, err := t.store.CreateReminder(ctx, t.userID, eventID, minutes*60); err != nil {
			warnings = append(warnings, fmt.Sprintf("Couldn't add the reminder %d minutes before", minutes))
		}
	}
	return warnings
}

// reminderMinutes checks the offsets and returns them sorted, without
// duplicates.
func reminderMinutes(minutes []int) ([]int, error) {
	for _, value := range minutes {
		if value < 0 || value*60 > db.MaxReminderOffsetSeconds {
			return nil, errors.New("reminder_minutes must be from 0 up to 30 days")
		}
	}
	unique := slices.Compact(slices.Sorted(slices.Values(minutes)))
	return unique[:min(len(unique), maxReminders)], nil
}
