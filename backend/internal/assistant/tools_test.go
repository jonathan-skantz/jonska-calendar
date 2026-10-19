package assistant

import (
	"encoding/json"
	"maps"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestToolSchemasComeFromInputTypes(t *testing.T) {
	tools := newTurn(fakeStore{}, uuid.New(), time.UTC, time.Now(), nil).tools()
	schemas := map[string]struct {
		Required             []string                  `json:"required"`
		AdditionalProperties bool                      `json:"additionalProperties"`
		Properties           map[string]map[string]any `json:"properties"`
	}{}
	for _, tool := range tools {
		data, err := json.Marshal(tool.InputSchema())
		if err != nil {
			t.Fatal(err)
		}
		schema := schemas[tool.Name()]
		if err := json.Unmarshal(data, &schema); err != nil {
			t.Fatal(err)
		}
		schemas[tool.Name()] = schema
	}

	if got := slices.Sorted(maps.Keys(schemas)); !slices.Equal(got, []string{"find_free_slots", "list_events", "propose_event"}) {
		t.Fatalf("unexpected tools %v", got)
	}
	propose := schemas["propose_event"]
	if !slices.Equal(propose.Required, []string{"title", "date"}) || propose.AdditionalProperties {
		t.Fatalf("expected only title and date required and no extra properties, got %+v", propose)
	}
	if kinds := propose.Properties["kind"]["enum"]; len(kinds.([]any)) != 2 {
		t.Fatalf("expected the two event kinds as an enum, got %v", kinds)
	}
	if description := propose.Properties["date"]["description"]; description != "YYYY-MM-DD" {
		t.Fatalf("expected the date format in the description, got %v", description)
	}
}

func TestFreeGaps(t *testing.T) {
	at := func(hour, minute int) time.Time { return time.Date(2026, 10, 23, hour, minute, 0, 0, time.UTC) }
	window := interval{start: at(11, 0), end: at(14, 0)}
	tests := []struct {
		name    string
		busy    []interval
		minimum time.Duration
		want    []interval
	}{
		{"an empty window is all free", nil, time.Hour, []interval{window}},
		{
			"overlapping busy spans merge",
			[]interval{{start: at(11, 30), end: at(12, 30)}, {start: at(12, 0), end: at(13, 0)}},
			30 * time.Minute,
			[]interval{{start: at(11, 0), end: at(11, 30)}, {start: at(13, 0), end: at(14, 0)}},
		},
		{
			"gaps shorter than the minimum are left out",
			[]interval{{start: at(11, 30), end: at(13, 30)}},
			time.Hour,
			nil,
		},
		{
			"busy time outside the window is clipped",
			[]interval{{start: at(9, 0), end: at(12, 0)}},
			time.Hour,
			[]interval{{start: at(12, 0), end: at(14, 0)}},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := freeGaps(window, test.busy, test.minimum); !slices.Equal(got, test.want) {
				t.Fatalf("expected %v, got %v", test.want, got)
			}
		})
	}
}

func TestReminderMinutes(t *testing.T) {
	got, err := reminderMinutes([]int{60, 15, 60, 0})
	if err != nil || !slices.Equal(got, []int{0, 15, 60}) {
		t.Fatalf("expected sorted unique offsets, got %v, %v", got, err)
	}
	many := make([]int, maxReminders+5)
	for i := range many {
		many[i] = i
	}
	if got, _ := reminderMinutes(many); len(got) != maxReminders {
		t.Fatalf("expected at most %d reminders, got %d", maxReminders, len(got))
	}
	for _, invalid := range []int{-1, 31 * 24 * 60} {
		if _, err := reminderMinutes([]int{invalid}); err == nil {
			t.Fatalf("expected %d minutes to be rejected", invalid)
		}
	}
}

func TestProposalTimes(t *testing.T) {
	now := time.Date(2026, 10, 19, 9, 0, 0, 0, time.UTC)
	current := newTurn(fakeStore{}, uuid.New(), time.UTC, now, nil)
	at := func(day, hour int) time.Time { return time.Date(2026, 10, day, hour, 0, 0, 0, time.UTC) }
	tests := []struct {
		name       string
		input      proposeEventInput
		start, end time.Time
		err        string
	}{
		{"a timed event without an end lasts an hour", proposeEventInput{Date: "2026-10-20", StartTime: "12:00"}, at(20, 12), at(20, 13), ""},
		{"an all-day event today ends at midnight", proposeEventInput{Date: "2026-10-19", AllDay: true}, at(19, 0), at(20, 0), ""},
		{"a start earlier today has passed", proposeEventInput{Date: "2026-10-19", StartTime: "08:00"}, time.Time{}, time.Time{}, "start_time has already passed"},
		{"an all-day event yesterday has passed", proposeEventInput{Date: "2026-10-18", AllDay: true}, time.Time{}, time.Time{}, "date has already passed"},
		{"an end before the start is rejected", proposeEventInput{Date: "2026-10-20", StartTime: "12:00", EndTime: "11:00"}, time.Time{}, time.Time{}, "end_time must be after"},
		{"an all-day event takes no times", proposeEventInput{Date: "2026-10-20", AllDay: true, StartTime: "12:00"}, time.Time{}, time.Time{}, "no start_time"},
		{"a timed event needs a start", proposeEventInput{Date: "2026-10-20"}, time.Time{}, time.Time{}, "start_time is required"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			start, end, err := current.proposalTimes(test.input)
			if test.err != "" {
				if err == nil || !strings.Contains(err.Error(), test.err) {
					t.Fatalf("expected an error containing %q, got %v", test.err, err)
				}
				return
			}
			if err != nil || !start.Equal(test.start) || !end.Equal(test.end) {
				t.Fatalf("expected %s–%s, got %s–%s, %v", test.start, test.end, start, end, err)
			}
		})
	}
}
