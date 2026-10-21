package assistant

import (
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/google/uuid"
)

const (
	conversationTTL = 30 * time.Minute
	maxTurns        = 10
)

var (
	errConversationNotFound = errors.New("conversation not found")
	errConversationBusy     = errors.New("conversation is busy")
	errConversationFull     = errors.New("conversation is full")
)

// conversations keeps each conversation on the server: its Messages API
// history, so a client can never send forged tool results, and its actions,
// so a client can only decide what the agent actually asked. It is in
// memory, which is enough for the single server this app runs on; a restart
// starts over.
type conversations struct {
	now func() time.Time

	mu   sync.Mutex
	byID map[uuid.UUID]*conversation
}

type conversation struct {
	userID  uuid.UUID
	history []anthropic.BetaMessageParam
	actions []Action
	// notes tell the agent, with the next message, what the user decided.
	notes   []string
	turns   int
	updated time.Time
	busy    bool
	// ended conversations take no more messages, but their actions can
	// still be decided.
	ended bool
}

// claimed is what a message starts from.
type claimed struct {
	id      uuid.UUID
	history []anthropic.BetaMessageParam
	notes   []string
	pending []Action
}

// answered is what a message leaves behind.
type answered struct {
	history []anthropic.BetaMessageParam
	actions []Action
	// notesSeen is how many notes the message passed to the agent.
	notesSeen int
	// ended is set when the history cannot take another message.
	ended bool
}

func newConversations(now func() time.Time) *conversations {
	return &conversations{now: now, byID: map[uuid.UUID]*conversation{}}
}

// claim starts a message in the user's conversation id, or in a new
// conversation when id is nil, and marks it busy until save or unlock.
func (c *conversations) claim(userID, id uuid.UUID) (claimed, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := c.now()
	c.forgetExpired(now)

	if id == uuid.Nil {
		id = uuid.New()
		c.byID[id] = &conversation{userID: userID, updated: now, busy: true}
		return claimed{id: id}, nil
	}
	existing, ok := c.find(userID, id)
	switch {
	case !ok || existing.ended:
		return claimed{}, errConversationNotFound
	case existing.busy:
		return claimed{}, errConversationBusy
	case existing.turns >= maxTurns:
		return claimed{}, errConversationFull
	}
	existing.busy = true
	pending := slices.DeleteFunc(slices.Clone(existing.actions), func(action Action) bool {
		return action.Status != statusPending
	})
	return claimed{id: id, history: existing.history, notes: slices.Clone(existing.notes), pending: pending}, nil
}

// save stores what an answered message left behind.
func (c *conversations) save(id uuid.UUID, result answered) {
	c.mu.Lock()
	defer c.mu.Unlock()
	existing, ok := c.byID[id]
	if !ok {
		return
	}
	existing.actions = append(existing.actions, result.actions...)
	existing.notes = existing.notes[result.notesSeen:]
	existing.updated, existing.busy = c.now(), false
	if result.ended {
		existing.history, existing.ended = nil, true
	} else {
		existing.history = result.history
		existing.turns++
	}
	if existing.ended && len(existing.actions) == 0 {
		delete(c.byID, id)
	}
}

// unlock frees the conversation after a failed message, keeping its history
// so the message can be retried. A new conversation is dropped.
func (c *conversations) unlock(id uuid.UUID) {
	c.mu.Lock()
	defer c.mu.Unlock()
	existing, ok := c.byID[id]
	if !ok {
		return
	}
	if existing.history == nil && len(existing.actions) == 0 {
		delete(c.byID, id)
		return
	}
	existing.updated, existing.busy = c.now(), false
}

// decide marks the user's decision on an action, so it is carried out once,
// and returns the action as it was. settle must follow.
func (c *conversations) decide(userID, id, actionID uuid.UUID, decision string) (Action, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.forgetExpired(c.now())
	action, err := c.action(userID, id, actionID)
	if err != nil {
		return Action{}, err
	}
	from, to, ok := transition(decision)
	if !ok || action.Status != from {
		return Action{}, errInvalidDecision
	}
	decided := *action
	action.Status = to
	c.byID[id].updated = c.now()
	return decided, nil
}

// settle records how a decision went: on failure the action can be decided
// again. Either way the agent hears about it with the next message.
func (c *conversations) settle(userID, id, actionID uuid.UUID, decision string, failure error) Action {
	c.mu.Lock()
	defer c.mu.Unlock()
	action, err := c.action(userID, id, actionID)
	if err != nil {
		return Action{}
	}
	from, _, _ := transition(decision)
	note := fmt.Sprintf("The user's %s of %s %q: %s.", decision, action.Kind, action.Title, action.Status)
	if failure != nil {
		action.Status = from
		note = fmt.Sprintf("The user's %s of %s %q failed: %v.", decision, action.Kind, action.Title, failure)
	}
	conversation := c.byID[id]
	conversation.notes = append(conversation.notes, note)
	conversation.updated = c.now()
	return *action
}

func (c *conversations) find(userID, id uuid.UUID) (*conversation, bool) {
	existing, ok := c.byID[id]
	return existing, ok && existing.userID == userID
}

func (c *conversations) action(userID, id, actionID uuid.UUID) (*Action, error) {
	existing, ok := c.find(userID, id)
	if !ok {
		return nil, errConversationNotFound
	}
	i := slices.IndexFunc(existing.actions, func(action Action) bool { return action.ID == actionID })
	if i < 0 {
		return nil, errActionNotFound
	}
	return &existing.actions[i], nil
}

// forgetExpired drops conversations idle for the TTL. Busy ones are kept so
// save and unlock still find them.
func (c *conversations) forgetExpired(now time.Time) {
	for id, existing := range c.byID {
		if !existing.busy && now.Sub(existing.updated) >= conversationTTL {
			delete(c.byID, id)
		}
	}
}
