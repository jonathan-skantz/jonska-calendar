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

// Store is the read access the tools need. Every call is scoped to the
// signed-in user, like the rest of the API.
type Store interface {
	ListTimeline(ctx context.Context, userID uuid.UUID) ([]db.TimelineEntry, error)
	ListTags(ctx context.Context, userID uuid.UUID) ([]db.Tag, error)
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

	mu        sync.Mutex
	calls     int
	entries   []db.TimelineEntry
	loaded    bool
	proposals []Proposal
	steps     []Step
}

func newTurn(store Store, userID uuid.UUID, location *time.Location, now time.Time, tags []db.Tag) *turn {
	return &turn{
		store: store, userID: userID, location: location, now: now, tags: tags,
		proposals: []Proposal{}, steps: []Step{},
	}
}

// tools lists what the agent can do. A new tool is an input struct whose
// jsonschema tags describe it, a method on turn, and a line here.
func (t *turn) tools() []anthropic.BetaTool {
	return []anthropic.BetaTool{
		newTool(t, "list_events", listEventsDescription, t.listEvents),
		newTool(t, "find_free_slots", findFreeSlotsDescription, t.findFreeSlots),
		newTool(t, "propose_event", proposeEventDescription, t.proposeEvent),
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

// timeline reads the user's timeline once per message. The caller holds mu.
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

// today is the start of the current day in the user's zone.
func (t *turn) today() time.Time {
	local := t.now.In(t.location)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, t.location)
}

func (t *turn) format(value time.Time) string {
	return value.In(t.location).Format(timeFormat)
}
