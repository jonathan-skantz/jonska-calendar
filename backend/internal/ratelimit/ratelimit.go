package ratelimit

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// Limiter counts requests per user in fixed windows. It is in memory, which
// is enough for the single server this app runs on.
type Limiter struct {
	limit  int
	window time.Duration
	// Now is replaceable so tests can move time.
	Now func() time.Time

	mu        sync.Mutex
	windows   map[uuid.UUID]count
	lastSweep time.Time
}

type count struct {
	start time.Time
	count int
}

func New(limit int, window time.Duration) *Limiter {
	return &Limiter{limit: limit, window: window, Now: time.Now, windows: map[uuid.UUID]count{}}
}

// Allow reports whether userID may make another request, and counts it if so.
func (l *Limiter) Allow(userID uuid.UUID) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.Now()
	// Expired windows are forgotten at most once per window, so the map holds
	// only users who made requests recently.
	if now.Sub(l.lastSweep) >= l.window {
		for id, window := range l.windows {
			if now.Sub(window.start) >= l.window {
				delete(l.windows, id)
			}
		}
		l.lastSweep = now
	}
	window := l.windows[userID]
	if now.Sub(window.start) >= l.window {
		window = count{start: now}
	}
	if window.count >= l.limit {
		return false
	}
	window.count++
	l.windows[userID] = window
	return true
}

// tracked is the number of users with a live window, for tests.
func (l *Limiter) tracked() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.windows)
}
