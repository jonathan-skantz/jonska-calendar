package assistant

import (
	"errors"
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

// conversations keeps each conversation's Messages API history on the server,
// so a client can never send forged tool results. It is in memory, which is
// enough for the single server this app runs on; a restart starts over.
type conversations struct {
	now func() time.Time

	mu   sync.Mutex
	byID map[uuid.UUID]*conversation
}

type conversation struct {
	userID  uuid.UUID
	history []anthropic.BetaMessageParam
	turns   int
	updated time.Time
	busy    bool
}

func newConversations(now func() time.Time) *conversations {
	return &conversations{now: now, byID: map[uuid.UUID]*conversation{}}
}

// claim returns the history of the user's conversation id, or starts a new
// conversation when id is nil, and marks it busy until save or unlock. A
// conversation belongs to the user who started it.
func (c *conversations) claim(userID, id uuid.UUID) (uuid.UUID, []anthropic.BetaMessageParam, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := c.now()
	c.forgetExpired(now)

	if id == uuid.Nil {
		id = uuid.New()
		c.byID[id] = &conversation{userID: userID, updated: now, busy: true}
		return id, nil, nil
	}
	existing, ok := c.byID[id]
	switch {
	case !ok || existing.userID != userID:
		return id, nil, errConversationNotFound
	case existing.busy:
		return id, nil, errConversationBusy
	case existing.turns >= maxTurns:
		return id, nil, errConversationFull
	}
	existing.busy = true
	return id, existing.history, nil
}

// save stores the history after an answered message. A nil history ends the
// conversation, for answers that leave it unable to continue.
func (c *conversations) save(id uuid.UUID, history []anthropic.BetaMessageParam) {
	c.mu.Lock()
	defer c.mu.Unlock()
	existing, ok := c.byID[id]
	if !ok {
		return
	}
	if history == nil {
		delete(c.byID, id)
		return
	}
	existing.history, existing.updated, existing.busy = history, c.now(), false
	existing.turns++
}

// unlock frees the conversation after a failed message, keeping its history
// so the message can be retried. A conversation with no history is dropped.
func (c *conversations) unlock(id uuid.UUID) {
	c.mu.Lock()
	defer c.mu.Unlock()
	existing, ok := c.byID[id]
	if !ok {
		return
	}
	if existing.history == nil {
		delete(c.byID, id)
		return
	}
	existing.updated, existing.busy = c.now(), false
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
