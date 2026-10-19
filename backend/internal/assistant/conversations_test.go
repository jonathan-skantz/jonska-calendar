package assistant

import (
	"errors"
	"testing"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/google/uuid"
)

func TestConversations(t *testing.T) {
	now := time.Date(2026, 10, 19, 9, 0, 0, 0, time.UTC)
	store := newConversations(func() time.Time { return now })
	userID := uuid.New()
	history := []anthropic.BetaMessageParam{anthropic.NewBetaUserMessage(anthropic.NewBetaTextBlock("hi"))}

	id, _, err := store.claim(userID, uuid.Nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.claim(userID, id); !errors.Is(err, errConversationBusy) {
		t.Fatalf("expected a busy conversation, got %v", err)
	}
	store.save(id, history)
	if _, _, err := store.claim(uuid.New(), id); !errors.Is(err, errConversationNotFound) {
		t.Fatalf("expected another user's conversation to be hidden, got %v", err)
	}

	// A failed message keeps the history and does not count as a turn.
	for range maxTurns * 2 {
		if _, got, err := store.claim(userID, id); err != nil || len(got) != 1 {
			t.Fatalf("expected the saved history, got %v, %v", got, err)
		}
		store.unlock(id)
	}
	for range maxTurns - 1 {
		if _, _, err := store.claim(userID, id); err != nil {
			t.Fatal(err)
		}
		store.save(id, history)
	}
	if _, _, err := store.claim(userID, id); !errors.Is(err, errConversationFull) {
		t.Fatalf("expected a full conversation after %d turns, got %v", maxTurns, err)
	}

	now = now.Add(conversationTTL)
	if _, _, err := store.claim(userID, id); !errors.Is(err, errConversationNotFound) {
		t.Fatalf("expected the idle conversation to expire, got %v", err)
	}
}

func TestConversationsEndOrDropEmpty(t *testing.T) {
	store := newConversations(time.Now)
	userID := uuid.New()

	failed, _, _ := store.claim(userID, uuid.Nil)
	store.unlock(failed)
	if _, _, err := store.claim(userID, failed); !errors.Is(err, errConversationNotFound) {
		t.Fatalf("expected a conversation whose first message failed to be dropped, got %v", err)
	}

	ended, _, _ := store.claim(userID, uuid.Nil)
	store.save(ended, nil)
	if _, _, err := store.claim(userID, ended); !errors.Is(err, errConversationNotFound) {
		t.Fatalf("expected an ended conversation to be gone, got %v", err)
	}
}
