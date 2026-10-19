package assistant

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/jonathan-skantz/calendar/backend/db"
)

// The cases are shared with frontend/src/lib/timeline.test.ts, which keeps
// them because Vite only reads files inside the frontend.
func TestOccupiedSpanMatchesSharedCases(t *testing.T) {
	data, err := os.ReadFile("../../../frontend/src/lib/testdata/occupied-spans.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Name  string           `json:"name"`
		Entry db.TimelineEntry `json:"entry"`
		Busy  bool             `json:"busy"`
		Start time.Time        `json:"start"`
		End   time.Time        `json:"end"`
	}
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			start, end, busy := occupiedSpan(c.Entry)
			if busy != c.Busy {
				t.Fatalf("expected busy %v, got %v", c.Busy, busy)
			}
			if busy && (!start.Equal(c.Start) || !end.Equal(c.End)) {
				t.Fatalf("expected %s–%s, got %s–%s", c.Start, c.End, start, end)
			}
		})
	}
}

func TestTasksNeverBlockTime(t *testing.T) {
	deadline := time.Date(2026, 10, 23, 10, 0, 0, 0, time.UTC)
	if _, _, busy := occupiedSpan(db.TimelineEntry{Type: "task", DeadlineAt: &deadline}); busy {
		t.Fatal("expected a task to leave the time free")
	}
}
