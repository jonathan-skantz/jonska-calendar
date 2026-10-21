package assistant

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jonathan-skantz/calendar/backend/db"
	"github.com/jonathan-skantz/calendar/backend/internal/timeline"
)

// Action kinds. Creating an event runs at once and can be undone; the others
// change what already exists, so they wait for the user to apply them.
const (
	kindCreate = "create"
	kindMove   = "move"
	kindSkip   = "skip"
	kindUnskip = "unskip"
	kindDelete = "delete"
)

// Action statuses.
const (
	statusDone     = "done"    // ran at once; can be undone
	statusPending  = "pending" // waits for the user to apply or decline it
	statusApplied  = "applied"
	statusDeclined = "declined"
	statusUndone   = "undone"
)

// Decisions the user can take on an action.
const (
	decisionApply   = "apply"
	decisionDecline = "decline"
	decisionUndo    = "undo"
)

var (
	errActionNotFound  = errors.New("action not found")
	errInvalidDecision = errors.New("that decision does not fit the action")
	// errInvalidChange marks a change the event rules reject.
	errInvalidChange = errors.New("invalid change")
)

// Action is something the agent did or wants to do to one of the user's own
// events. StartsAt and EndsAt are the event's times after the action.
type Action struct {
	ID       uuid.UUID `json:"id"`
	Kind     string    `json:"kind"`
	Status   string    `json:"status"`
	EventID  uuid.UUID `json:"event_id"`
	Title    string    `json:"title"`
	StartsAt time.Time `json:"starts_at"`
	EndsAt   time.Time `json:"ends_at"`
	IsAllDay bool      `json:"is_all_day"`
	Warnings []string  `json:"warnings,omitempty"`
}

// transition is the status a decision leads to, and the status it starts
// from. ok is false for decisions that do not exist.
func transition(decision string) (from, to string, ok bool) {
	switch decision {
	case decisionApply:
		return statusPending, statusApplied, true
	case decisionDecline:
		return statusPending, statusDeclined, true
	case decisionUndo:
		return statusDone, statusUndone, true
	}
	return "", "", false
}

// change returns event with a move, skip or unskip applied. It is used both
// when the agent queues the action, to report problems early, and when the
// user applies it, on the event as it is then. A move must still lie ahead
// at that point.
func change(event db.Event, action Action, now time.Time) (db.Event, error) {
	switch action.Kind {
	case kindMove:
		if action.StartsAt.Before(now) {
			return db.Event{}, fmt.Errorf("%w: the new time has already passed", errInvalidChange)
		}
		event.StartsAt, event.EndsAt = action.StartsAt, action.EndsAt
	case kindSkip:
		event.IsSkipped = true
	case kindUnskip:
		event.IsSkipped = false
	}
	validated, err := timeline.ValidateEvent(event)
	if err != nil {
		return db.Event{}, fmt.Errorf("%w: %w", errInvalidChange, err)
	}
	return validated, nil
}

// carryOut makes the change a decision stands for. Declining changes
// nothing; undoing removes the created event.
func carryOut(ctx context.Context, store Store, userID uuid.UUID, action Action, decision string, now time.Time) error {
	switch {
	case decision == decisionDecline:
		return nil
	case decision == decisionUndo, action.Kind == kindDelete:
		return store.DeleteTimelineItem(ctx, userID, action.EventID)
	}
	entry, err := ownEvent(ctx, store, userID, action.EventID)
	if err != nil {
		return err
	}
	event, err := change(entry.Event(), action, now)
	if err != nil {
		return err
	}
	return store.UpdateEvent(ctx, userID, action.EventID, event)
}
