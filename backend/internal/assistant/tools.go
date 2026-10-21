package assistant

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/toolrunner"
	"github.com/google/uuid"
	"github.com/jonathan-skantz/calendar/backend/db"
)

const (
	dateLayout  = "2006-01-02"
	clockLayout = "15:04"
	// Tool output formats, so the model sees the weekday with every date.
	dayFormat  = "Mon 2006-01-02"
	timeFormat = "Mon 2006-01-02 15:04"

	maxToolCalls = 16
)

var errToolLimit = errors.New("tool call limit reached for this message; answer with what you have")

// Store is what the agent may read and write. Every call is scoped to the
// signed-in user, like the rest of the API, so it only reaches the user's own
// data.
type Store interface {
	ListTimeline(ctx context.Context, userID uuid.UUID) ([]db.TimelineEntry, error)
	ListTags(ctx context.Context, userID uuid.UUID) ([]db.Tag, error)
	CreateEvent(ctx context.Context, userID uuid.UUID, event db.Event) (uuid.UUID, error)
	UpdateEvent(ctx context.Context, userID, eventID uuid.UUID, event db.Event) error
	DeleteTimelineItem(ctx context.Context, userID, itemID uuid.UUID) error
	AddTagToItem(ctx context.Context, userID, itemID, tagID uuid.UUID) error
	CreateReminder(ctx context.Context, userID, eventID uuid.UUID, offsetSeconds int) (uuid.UUID, error)
}

// Step is one tool call, summarized for the user to see what the agent did.
type Step struct {
	Tool    string `json:"tool"`
	Summary string `json:"summary"`
}

// turn holds what the tools of one user message share. The runner may run
// several tool calls at once, so each tool holds mu for its whole run.
type turn struct {
	store    Store
	userID   uuid.UUID
	location *time.Location
	now      time.Time
	tags     []db.Tag

	mu      sync.Mutex
	calls   int
	entries []db.TimelineEntry
	loaded  bool
	actions []Action
	steps   []Step
}

func newTurn(store Store, userID uuid.UUID, location *time.Location, now time.Time, tags []db.Tag) *turn {
	return &turn{
		store: store, userID: userID, location: location, now: now, tags: tags,
		actions: []Action{}, steps: []Step{},
	}
}

// tools lists what the agent can do. A new tool is an input struct whose
// jsonschema tags describe it, a method on turn, and a line here.
func (t *turn) tools() []anthropic.BetaTool {
	return []anthropic.BetaTool{
		newTool(t, "list_events", listEventsDescription, t.listEvents),
		newTool(t, "find_free_slots", findFreeSlotsDescription, t.findFreeSlots),
		newTool(t, "create_event", createEventDescription, t.createEvent),
		newTool(t, "move_event", moveEventDescription, t.moveEvent),
		newTool(t, "skip_event", skipEventDescription, t.skipEvent),
		newTool(t, "delete_event", deleteEventDescription, t.deleteEvent),
	}
}

// toolRun is a tool's body. It returns what the model sees and a one-line
// summary for the user.
type toolRun[T any] func(ctx context.Context, input T) (result any, summary string, err error)

// newTool derives the input schema from T and wraps run so that tool calls
// are serialized, counted against the per-message limit, recorded as steps,
// and returned as JSON. An error goes back to the model as the tool result.
func newTool[T any](t *turn, name, description string, run toolRun[T]) anthropic.BetaTool {
	tool, err := toolrunner.NewBetaToolFromJSONSchema(name, description,
		func(ctx context.Context, input T) (anthropic.BetaToolResultBlockParamContentUnion, error) {
			t.mu.Lock()
			defer t.mu.Unlock()
			if t.calls >= maxToolCalls {
				return anthropic.BetaToolResultBlockParamContentUnion{}, errToolLimit
			}
			t.calls++
			result, summary, err := run(ctx, input)
			if err != nil {
				return anthropic.BetaToolResultBlockParamContentUnion{}, err
			}
			t.steps = append(t.steps, Step{Tool: name, Summary: summary})
			text, err := json.Marshal(result)
			if err != nil {
				return anthropic.BetaToolResultBlockParamContentUnion{}, err
			}
			return anthropic.BetaToolResultBlockParamContentUnion{OfText: &anthropic.BetaTextBlockParam{Text: string(text)}}, nil
		})
	if err != nil {
		// The schema comes from a Go type, so this is a programming error.
		panic(fmt.Sprintf("assistant: tool %s: %v", name, err))
	}
	return tool
}

// timeline reads the user's timeline once per message, and again after the
// agent created an event, so later tools see it. The caller holds mu.
func (t *turn) timeline(ctx context.Context) ([]db.TimelineEntry, error) {
	if !t.loaded {
		entries, err := t.store.ListTimeline(ctx, t.userID)
		if err != nil {
			return nil, fmt.Errorf("loading the calendar failed: %w", err)
		}
		t.entries, t.loaded = entries, true
	}
	return t.entries, nil
}

// canAct refuses another change once the message has made its share. Tools
// check it before writing anything.
func (t *turn) canAct() error {
	if len(t.actions) >= maxActions {
		return fmt.Errorf("at most %d changes per message", maxActions)
	}
	return nil
}

// date reads a YYYY-MM-DD date as local midnight in the user's zone.
func (t *turn) date(value, field string) (time.Time, error) {
	day, err := time.ParseInLocation(dateLayout, value, t.location)
	if err != nil {
		return time.Time{}, fmt.Errorf("%s must be a YYYY-MM-DD date", field)
	}
	return day, nil
}

// at is the given clock time on day, in the user's zone.
func (t *turn) at(day time.Time, value, field string) (time.Time, error) {
	clock, err := time.Parse(clockLayout, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("%s must be HH:MM, 24-hour", field)
	}
	return time.Date(day.Year(), day.Month(), day.Day(), clock.Hour(), clock.Minute(), 0, 0, t.location), nil
}

// timedRange is the span from startClock on day to endClock, or lasting
// duration without an end. It must not have started yet.
func (t *turn) timedRange(day time.Time, startClock, endClock string, duration time.Duration) (time.Time, time.Time, error) {
	start, err := t.at(day, startClock, "start_time")
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	end := start.Add(duration)
	if endClock != "" {
		if end, err = t.at(day, endClock, "end_time"); err != nil {
			return time.Time{}, time.Time{}, err
		}
	}
	switch {
	case !end.After(start):
		return time.Time{}, time.Time{}, errors.New("end_time must be after start_time on the same date")
	case start.Before(t.now):
		return time.Time{}, time.Time{}, errors.New("start_time has already passed")
	}
	return start, end, nil
}

// allDayRange is days whole days from day, ending at midnight after the last
// one. It must not be in the past.
func (t *turn) allDayRange(day time.Time, days int) (time.Time, time.Time, error) {
	local := t.now.In(t.location)
	if day.Before(time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, t.location)) {
		return time.Time{}, time.Time{}, errors.New("date has already passed")
	}
	return day, day.AddDate(0, 0, days), nil
}

// eventID reads an event id the model took from list_events.
func eventID(value string) (uuid.UUID, error) {
	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, errors.New("event_id must be an id from list_events")
	}
	return id, nil
}

func (t *turn) format(value time.Time) string {
	return value.In(t.location).Format(timeFormat)
}
