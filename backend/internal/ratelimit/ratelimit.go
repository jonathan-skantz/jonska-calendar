package ratelimit

import (
	"sync"
	"time"
)

// Limiter counts requests per key, such as a user ID or a client IP, in fixed
// windows. It is in memory, which is enough for the single server this app
// runs on.
type Limiter[K comparable] struct {
	limit  int
	window time.Duration
	// Now is replaceable so tests can move time.
	Now func() time.Time

	mu        sync.Mutex
	windows   map[K]count
	lastSweep time.Time
}

type count struct {
	start time.Time
	count int
}

func New[K comparable](limit int, window time.Duration) *Limiter[K] {
	return &Limiter[K]{limit: limit, window: window, Now: time.Now, windows: map[K]count{}}
}

// Allow reports whether key may make another request, and counts it if so.
func (l *Limiter[K]) Allow(key K) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.Now()
	// Expired windows are forgotten at most once per window, so the map holds
	// only keys that made requests recently.
	if now.Sub(l.lastSweep) >= l.window {
		for id, window := range l.windows {
			if now.Sub(window.start) >= l.window {
				delete(l.windows, id)
			}
		}
		l.lastSweep = now
	}
	window := l.windows[key]
	if now.Sub(window.start) >= l.window {
		window = count{start: now}
	}
	if window.count >= l.limit {
		return false
	}
	window.count++
	l.windows[key] = window
	return true
}

// tracked is the number of keys with a live window, for tests.
func (l *Limiter[K]) tracked() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.windows)
}
