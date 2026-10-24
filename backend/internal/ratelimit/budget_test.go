package ratelimit

import (
	"testing"
	"time"
)

func TestBudgetResetsEachUTCDay(t *testing.T) {
	budget := NewBudget(2)
	now := time.Date(2026, 10, 17, 23, 0, 0, 0, time.UTC)
	budget.Now = func() time.Time { return now }

	if !budget.Allow() || !budget.Allow() {
		t.Fatal("expected the first two requests to be allowed")
	}
	if budget.Allow() {
		t.Fatal("expected the third request to be refused")
	}
	now = now.Add(time.Hour)
	if !budget.Allow() {
		t.Fatal("expected the budget to reset at midnight UTC")
	}
}

func TestNilBudgetAllowsEverything(t *testing.T) {
	var budget *Budget
	if !budget.Allow() {
		t.Fatal("expected a nil budget to allow")
	}
}
