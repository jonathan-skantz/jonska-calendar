package assistant

import (
	"encoding/json"
	"maps"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jonathan-skantz/calendar/backend/db"
)

func TestToolSchemasComeFromInputTypes(t *testing.T) {
	tools := newTurn(nil, uuid.New(), time.UTC, time.Now(), nil).tools()
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

	want := []string{"create_event", "delete_event", "find_free_slots", "list_events", "move_event", "skip_event"}
	if got := slices.Sorted(maps.Keys(schemas)); !slices.Equal(got, want) {
		t.Fatalf("unexpected tools %v", got)
	}
	propose := schemas["create_event"]
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

func TestNewEventTimes(t *testing.T) {
	now := time.Date(2026, 10, 19, 9, 0, 0, 0, time.UTC)
	current := newTurn(nil, uuid.New(), time.UTC, now, nil)
	at := func(day, hour int) time.Time { return time.Date(2026, 10, day, hour, 0, 0, 0, time.UTC) }
	tests := []struct {
		name       string
		input      createEventInput
		start, end time.Time
		err        string
	}{
		{"a timed event without an end lasts an hour", createEventInput{Date: "2026-10-20", StartTime: "12:00"}, at(20, 12), at(20, 13), ""},
		{"an all-day event today ends at midnight", createEventInput{Date: "2026-10-19", AllDay: true}, at(19, 0), at(20, 0), ""},
		{"a start earlier today has passed", createEventInput{Date: "2026-10-19", StartTime: "08:00"}, time.Time{}, time.Time{}, "start_time has already passed"},
		{"an all-day event yesterday has passed", createEventInput{Date: "2026-10-18", AllDay: true}, time.Time{}, time.Time{}, "date has already passed"},
		{"an end before the start is rejected", createEventInput{Date: "2026-10-20", StartTime: "12:00", EndTime: "11:00"}, time.Time{}, time.Time{}, "end_time must be after"},
		{"an all-day event takes no times", createEventInput{Date: "2026-10-20", AllDay: true, StartTime: "12:00"}, time.Time{}, time.Time{}, "no start_time"},
		{"a timed event needs a start", createEventInput{Date: "2026-10-20"}, time.Time{}, time.Time{}, "start_time is required"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.input.Title = "Lunch"
			event, _, _, err := current.newEvent(test.input)
			if test.err != "" {
				if err == nil || !strings.Contains(err.Error(), test.err) {
					t.Fatalf("expected an error containing %q, got %v", test.err, err)
				}
				return
			}
			if err != nil || !event.StartsAt.Equal(test.start) || !event.EndsAt.Equal(test.end) || event.Kind != db.EventKindAttendable {
				t.Fatalf("expected an attended event %s–%s, got %+v, %v", test.start, test.end, event, err)
			}
		})
	}
}

func TestMovedTimes(t *testing.T) {
	stockholm, err := time.LoadLocation("Europe/Stockholm")
	if err != nil {
		t.Fatal(err)
	}
	current := newTurn(nil, uuid.New(), stockholm, time.Date(2026, 10, 19, 9, 0, 0, 0, stockholm), nil)
	at := func(day, hour, minute int) time.Time { return time.Date(2026, 10, day, hour, minute, 0, 0, stockholm) }
	timed := db.Event{StartsAt: at(22, 10, 30), EndsAt: at(22, 12, 0)}
	allDay := db.Event{IsAllDay: true, StartsAt: at(23, 0, 0), EndsAt: at(25, 0, 0)}
	tests := []struct {
		name       string
		event      db.Event
		input      moveEventInput
		start, end time.Time
		err        string
	}{
		{"a new date keeps the time and duration", timed, moveEventInput{Date: "2026-10-27"}, at(27, 10, 30), at(27, 12, 0), ""},
		{"a new start keeps the duration", timed, moveEventInput{Date: "2026-10-22", StartTime: "14:00"}, at(22, 14, 0), at(22, 15, 30), ""},
		{"a new end changes the duration", timed, moveEventInput{Date: "2026-10-22", EndTime: "11:00"}, at(22, 10, 30), at(22, 11, 0), ""},
		{"an all-day event keeps its days across the time change", allDay, moveEventInput{Date: "2026-10-24"}, at(24, 0, 0), at(26, 0, 0), ""},
		{"an all-day event moves by date only", allDay, moveEventInput{Date: "2026-10-24", StartTime: "10:00"}, time.Time{}, time.Time{}, "by date only"},
		{"a move into the past is rejected", timed, moveEventInput{Date: "2026-10-18"}, time.Time{}, time.Time{}, "already passed"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			start, end, err := current.movedTimes(test.event, test.input)
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
