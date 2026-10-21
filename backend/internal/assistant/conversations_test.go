package assistant

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/google/uuid"
)

var history = []anthropic.BetaMessageParam{anthropic.NewBetaUserMessage(anthropic.NewBetaTextBlock("hi"))}

func TestConversations(t *testing.T) {
	now := time.Date(2026, 10, 19, 9, 0, 0, 0, time.UTC)
	store := newConversations(func() time.Time { return now })
	userID := uuid.New()

	start, err := store.claim(userID, uuid.Nil)
	if err != nil {
		t.Fatal(err)
	}
	id := start.id
	if _, err := store.claim(userID, id); !errors.Is(err, errConversationBusy) {
		t.Fatalf("expected a busy conversation, got %v", err)
	}
	store.save(id, answered{history: history})
	if _, err := store.claim(uuid.New(), id); !errors.Is(err, errConversationNotFound) {
		t.Fatalf("expected another user's conversation to be hidden, got %v", err)
	}

	// A failed message keeps the history and does not count as a turn.
	for range maxTurns * 2 {
		if start, err := store.claim(userID, id); err != nil || len(start.history) != 1 {
			t.Fatalf("expected the saved history, got %+v, %v", start, err)
		}
		store.unlock(id)
	}
	for range maxTurns - 1 {
		if _, err := store.claim(userID, id); err != nil {
			t.Fatal(err)
		}
		store.save(id, answered{history: history})
	}
	if _, err := store.claim(userID, id); !errors.Is(err, errConversationFull) {
		t.Fatalf("expected a full conversation after %d turns, got %v", maxTurns, err)
	}

	now = now.Add(conversationTTL)
	if _, err := store.claim(userID, id); !errors.Is(err, errConversationNotFound) {
		t.Fatalf("expected the idle conversation to expire, got %v", err)
	}
}

func TestConversationsEndOrDropEmpty(t *testing.T) {
	store := newConversations(time.Now)
	userID := uuid.New()

	failed, _ := store.claim(userID, uuid.Nil)
	store.unlock(failed.id)
	if _, err := store.claim(userID, failed.id); !errors.Is(err, errConversationNotFound) {
		t.Fatalf("expected a conversation whose first message failed to be dropped, got %v", err)
	}

	ended, _ := store.claim(userID, uuid.Nil)
	store.save(ended.id, answered{ended: true})
	if _, err := store.claim(userID, ended.id); !errors.Is(err, errConversationNotFound) {
		t.Fatalf("expected an ended conversation to take no messages, got %v", err)
	}
}

func TestDecisionsAreMadeOnceAndReachTheAgent(t *testing.T) {
	store := newConversations(time.Now)
	userID := uuid.New()
	move := Action{ID: uuid.New(), Kind: kindMove, Status: statusPending, Title: "Dentist"}
	skip := Action{ID: uuid.New(), Kind: kindSkip, Status: statusPending, Title: "Lecture"}

	start, _ := store.claim(userID, uuid.Nil)
	store.save(start.id, answered{history: history, actions: []Action{move, skip}, ended: true})

	if _, err := store.decide(uuid.New(), start.id, move.ID, decisionApply); !errors.Is(err, errConversationNotFound) {
		t.Fatalf("expected another user to be refused, got %v", err)
	}
	if _, err := store.decide(userID, start.id, uuid.New(), decisionApply); !errors.Is(err, errActionNotFound) {
		t.Fatalf("expected an unknown action to be refused, got %v", err)
	}
	if _, err := store.decide(userID, start.id, move.ID, decisionUndo); !errors.Is(err, errInvalidDecision) {
		t.Fatalf("expected undo of a pending action to be refused, got %v", err)
	}

	decided, err := store.decide(userID, start.id, move.ID, decisionApply)
	if err != nil || decided.Status != statusPending {
		t.Fatalf("expected the action as it was, got %+v, %v", decided, err)
	}
	if _, err := store.decide(userID, start.id, move.ID, decisionDecline); !errors.Is(err, errInvalidDecision) {
		t.Fatalf("expected a second decision while the first runs to be refused, got %v", err)
	}
	if settled := store.settle(userID, start.id, move.ID, decisionApply, nil); settled.Status != statusApplied {
		t.Fatalf("expected the move applied, got %+v", settled)
	}

	// A failed decision leaves the action open.
	store.decide(userID, start.id, skip.ID, decisionApply)
	if settled := store.settle(userID, start.id, skip.ID, decisionApply, errors.New("boom")); settled.Status != statusPending {
		t.Fatalf("expected the failed skip pending again, got %+v", settled)
	}

	notes := store.byID[start.id].notes
	if len(notes) != 2 || !strings.Contains(notes[0], `apply of move "Dentist": applied`) || !strings.Contains(notes[1], "failed: boom") {
		t.Fatalf("unexpected notes %q", notes)
	}
}

func TestClaimPassesNotesAndPendingActions(t *testing.T) {
	store := newConversations(time.Now)
	userID := uuid.New()
	pending := Action{ID: uuid.New(), Kind: kindDelete, Status: statusPending}
	done := Action{ID: uuid.New(), Kind: kindCreate, Status: statusDone}

	first, _ := store.claim(userID, uuid.Nil)
	store.save(first.id, answered{history: history, actions: []Action{pending, done}})
	store.decide(userID, first.id, done.ID, decisionUndo)
	store.settle(userID, first.id, done.ID, decisionUndo, nil)

	second, _ := store.claim(userID, first.id)
	if len(second.notes) != 1 || len(second.pending) != 1 || second.pending[0].ID != pending.ID {
		t.Fatalf("expected one note and the pending action, got %+v", second)
	}
	// A note made while the message runs waits for the next one.
	store.decide(userID, first.id, pending.ID, decisionDecline)
	store.settle(userID, first.id, pending.ID, decisionDecline, nil)
	store.save(first.id, answered{history: history, notesSeen: len(second.notes)})
	if third, _ := store.claim(userID, first.id); len(third.notes) != 1 || !strings.Contains(third.notes[0], "decline") {
		t.Fatalf("expected only the new note, got %q", third.notes)
	}
}
