package ratelimit

import (
	"errors"
	"sync"
	"time"
)

// ErrDailyBudget is the message for a request refused by a spent Budget.
var ErrDailyBudget = errors.New("the AI features have reached today's limit, try again tomorrow")

// Budget counts requests across all users per UTC day, for spending that is
// shared, such as calls on the host's API key.
type Budget struct {
	limit int
	// Now is replaceable so tests can move time.
	Now func() time.Time

	mu   sync.Mutex
	day  string
	used int
}

func NewBudget(limit int) *Budget {
	return &Budget{limit: limit, Now: time.Now}
}

// Allow reports whether today's budget has room for another request, and
// counts it if so. A nil budget allows everything.
func (b *Budget) Allow() bool {
	if b == nil {
		return true
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if day := b.Now().UTC().Format(time.DateOnly); day != b.day {
		b.day, b.used = day, 0
	}
	if b.used >= b.limit {
		return false
	}
	b.used++
	return true
}
