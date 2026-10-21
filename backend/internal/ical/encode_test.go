package ical

import (
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/jonathan-skantz/calendar/backend/db"
)

func exportEvents() []db.ExportEvent {
	startsAt := time.Date(2026, 10, 24, 16, 0, 0, 0, time.UTC)
	return []db.ExportEvent{
		{
			Event: db.Event{
				ID: uuid.New(), Title: "Football; semi-final, home",
				Description: new("Bring\nboots"), Location: new("Arena"), URL: new("https://example.com/match"),
				StartsAt: startsAt, EndsAt: startsAt.Add(2 * time.Hour),
				Kind: db.EventKindAttendable, StartIsApproximate: true, EndIsApproximate: true,
				CommuteBeforeSeconds: 1800, CommuteAfterSeconds: 2700,
				GatheringSeconds: 900, GatheringIsOptional: true, GatheringLabel: new("Warm-up"),
				IsSkipped: true,
			},
			UID:             "match@jonska-calendar",
			TagNames:        []string{"sport", "family"},
			ReminderOffsets: []int{0, 900, 86400},
		},
		{
			// Daylight saving ends on 25 October, inside this event.
			Event: db.Event{
				ID: uuid.New(), Title: "Weekend away" + strings.Repeat(" with a long title", 6),
				StartsAt: time.Date(2026, 10, 24, 0, 0, 0, 0, stockholm),
				EndsAt:   time.Date(2026, 10, 26, 0, 0, 0, 0, stockholm),
				IsAllDay: true, Kind: db.EventKindInformational,
			},
			UID: "weekend@example.com",
		},
	}
}

func TestEncodeWritesValidICalendar(t *testing.T) {
	var file strings.Builder
	stamp := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	if err := Encode(&file, exportEvents(), stockholm, stamp); err != nil {
		t.Fatal(err)
	}
	text := file.String()
	for _, want := range []string{
		"BEGIN:VCALENDAR\r\n", "VERSION:2.0\r\n", "PRODID:" + productID + "\r\n",
		"DTSTAMP:20261021T120000Z\r\n",
		"SUMMARY:Football\\; semi-final\\, home\r\n",
		"DESCRIPTION:Bring\\nboots\r\n",
		"DTSTART:20261024T160000Z\r\n",
		"DTSTART;VALUE=DATE:20261024\r\n", "DTEND;VALUE=DATE:20261026\r\n",
		"TRANSP:TRANSPARENT\r\n",
		"TRIGGER:-PT15M\r\n", "TRIGGER:-PT24H\r\n", "TRIGGER:PT0S\r\n",
		"X-JONSKA-COMMUTE-BEFORE:PT30M\r\n",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("expected %q in:\n%s", want, text)
		}
	}
	for line := range strings.SplitSeq(strings.TrimSuffix(text, "\r\n"), "\r\n") {
		if len(line) > 75 {
			t.Errorf("line longer than 75 octets: %q", line)
		}
	}
	if strings.Contains(text, "COMMENT") {
		t.Errorf("unexpected COMMENT in:\n%s", text)
	}
}

func TestEncodeThenParseKeepsEvents(t *testing.T) {
	events := exportEvents()
	var file strings.Builder
	if err := Encode(&file, events, stockholm, time.Now()); err != nil {
		t.Fatal(err)
	}
	result, err := Parse(strings.NewReader(file.String()), stockholm)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Events) != len(events) || len(result.Skipped) != 0 || len(result.Warnings) != 0 {
		t.Fatalf("unexpected result %+v", result)
	}
	for index, parsed := range result.Events {
		want := events[index]
		// IDs and main tags belong to the account, not the file.
		want.ID, want.MainTagID = uuid.Nil, uuid.Nil
		if !parsed.StartsAt.Equal(want.StartsAt) || !parsed.EndsAt.Equal(want.EndsAt) {
			t.Fatalf("event %d: expected %v to %v, got %v to %v", index, want.StartsAt, want.EndsAt, parsed.StartsAt, parsed.EndsAt)
		}
		parsed.StartsAt, parsed.EndsAt = want.StartsAt, want.EndsAt
		if !reflect.DeepEqual(parsed.Event, want.Event) {
			t.Fatalf("event %d: expected %+v, got %+v", index, want.Event, parsed.Event)
		}
		if parsed.UID != want.UID || !slices.Equal(parsed.Categories, want.TagNames) ||
			!slices.Equal(parsed.ReminderOffsets, want.ReminderOffsets) {
			t.Fatalf("event %d: expected %+v, got %+v", index, want, parsed)
		}
	}
}
