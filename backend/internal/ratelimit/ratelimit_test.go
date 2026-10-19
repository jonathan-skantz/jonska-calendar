package ratelimit

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestAllowLimitsEachUserPerWindow(t *testing.T) {
	limiter := New(2, time.Hour)
	now := time.Date(2026, 10, 17, 8, 0, 0, 0, time.UTC)
	limiter.Now = func() time.Time { return now }
	userID := uuid.New()

	if !limiter.Allow(userID) || !limiter.Allow(userID) {
		t.Fatal("expected the first two requests to be allowed")
	}
	if limiter.Allow(userID) {
		t.Fatal("expected the third request to be refused")
	}
	if !limiter.Allow(uuid.New()) {
		t.Fatal("expected another user to have their own budget")
	}
	now = now.Add(time.Hour)
	if !limiter.Allow(userID) {
		t.Fatal("expected the budget to reset after the window")
	}
}

func TestWindowsOfIdleUsersAreForgotten(t *testing.T) {
	limiter := New(5, time.Hour)
	now := time.Date(2026, 10, 17, 8, 0, 0, 0, time.UTC)
	limiter.Now = func() time.Time { return now }
	for range 3 {
		limiter.Allow(uuid.New())
	}
	if got := limiter.tracked(); got != 3 {
		t.Fatalf("expected 3 windows, got %d", got)
	}

	now = now.Add(time.Hour)
	limiter.Allow(uuid.New())
	if got := limiter.tracked(); got != 1 {
		t.Fatalf("expected only the active user's window, got %d", got)
	}
}
