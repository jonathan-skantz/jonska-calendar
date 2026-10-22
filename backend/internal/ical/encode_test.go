package ical

import (
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/jonathan-skantz/calendar/backend/db"
	"github.com/jonathan-skantz/calendar/backend/internal/recurrence"
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
		// The football's 15 minutes of gathering start it earlier, and its
		// reminders are relative to that.
		"DTSTART:20261024T154500Z\r\n", "DTEND:20261024T180000Z\r\n",
		"X-JONSKA-GATHERING:PT15M\r\n", "X-JONSKA-STARTS-AT-GATHERING:TRUE\r\n",
		"DTSTART;VALUE=DATE:20261024\r\n", "DTEND;VALUE=DATE:20261026\r\n",
		"TRANSP:TRANSPARENT\r\n",
		"TRIGGER:PT15M\r\n", "TRIGGER:PT0S\r\n", "TRIGGER:-PT23H45M\r\n",
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

func TestEncodeFeedLeavesOutCommuteAndSkips(t *testing.T) {
	events := exportEvents()
	start := events[0].StartsAt
	rule, moved := "FREQ=DAILY", start.Add(time.Hour)
	skipped := true
	events[0].RecurrenceRule, events[0].RecurrenceTimeZone = &rule, new("UTC")
	events[0].Exceptions = []db.OccurrenceException{
		{OriginalStart: start.AddDate(0, 0, 1), IsSkipped: &skipped},
		{OriginalStart: start.AddDate(0, 0, 2), StartsAt: &moved, EndsAt: new(moved.Add(time.Hour)), IsSkipped: &skipped},
	}
	var file strings.Builder
	if err := EncodeFeed(&file, "Family", events, stockholm, time.Now()); err != nil {
		t.Fatal(err)
	}
	text := file.String()
	for _, unwanted := range []string{"COMMUTE", "SKIPPED", "RECURRENCE-ID;TZID=UTC:20261025"} {
		if strings.Contains(text, unwanted) {
			t.Errorf("expected no %q in:\n%s", unwanted, text)
		}
	}
	for _, want := range []string{"X-JONSKA-GATHERING:PT15M\r\n", "RECURRENCE-ID;TZID=UTC:20261026T154500\r\n"} {
		if !strings.Contains(text, want) {
			t.Errorf("expected %q in:\n%s", want, text)
		}
	}
	if events[0].CommuteBeforeSeconds == 0 || events[0].Exceptions[0].IsSkipped == nil {
		t.Fatal("expected the caller's events unchanged")
	}
}

func TestGatheringStartsRepeatingEventsOnTheSameDayOnly(t *testing.T) {
	encodeSeries := func(start time.Time, gatheringSeconds int) (string, Event) {
		t.Helper()
		rule := recurrence.Rule{Frequency: recurrence.Weekly, Interval: 1, Weekdays: []string{"TU"}}.RRULE(start, false)
		moved := start.AddDate(0, 0, 7).Add(time.Hour)
		series := db.ExportEvent{
			Event: db.Event{
				Title: "Practice", StartsAt: start, EndsAt: start.Add(time.Hour), Kind: db.EventKindAttendable,
				GatheringSeconds: gatheringSeconds, RecurrenceRule: &rule, RecurrenceTimeZone: new("Europe/Stockholm"),
			},
			UID:             "practice@jonska-calendar",
			ReminderOffsets: []int{600},
			Exceptions: []db.OccurrenceException{
				{OriginalStart: start.AddDate(0, 0, 14), IsCancelled: true},
				{OriginalStart: start.AddDate(0, 0, 7), StartsAt: &moved, EndsAt: new(moved.Add(time.Hour))},
			},
		}
		var file strings.Builder
		if err := Encode(&file, []db.ExportEvent{series}, stockholm, time.Now()); err != nil {
			t.Fatal(err)
		}
		result, err := Parse(strings.NewReader(file.String()), stockholm)
		if err != nil || len(result.Events) != 1 || len(result.Warnings) != 0 {
			t.Fatalf("unexpected result %+v, %v", result, err)
		}
		got := result.Events[0]
		if !got.StartsAt.Equal(start) || !slices.Equal(got.ReminderOffsets, []int{600}) || len(got.Exceptions) != 2 ||
			!got.Exceptions[0].OriginalStart.Equal(start.AddDate(0, 0, 14)) ||
			!got.Exceptions[1].OriginalStart.Equal(start.AddDate(0, 0, 7)) || !got.Exceptions[1].StartsAt.Equal(moved) {
			t.Fatalf("expected the series read back as written, got %+v", got)
		}
		return file.String(), got
	}

	text, _ := encodeSeries(time.Date(2026, 10, 20, 18, 0, 0, 0, stockholm), 1800)
	for _, want := range []string{
		"DTSTART;TZID=Europe/Stockholm:20261020T173000\r\n",
		"EXDATE;TZID=Europe/Stockholm:20261103T173000\r\n",
		"RECURRENCE-ID;TZID=Europe/Stockholm:20261027T173000\r\n",
		"DTSTART;TZID=Europe/Stockholm:20261027T183000\r\n",
		"TRIGGER:PT20M\r\n",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("expected %q in:\n%s", want, text)
		}
	}
	// Gathering from 23:30 on Monday would make the rule's Tuesday a Monday.
	text, _ = encodeSeries(time.Date(2026, 10, 20, 0, 15, 0, 0, stockholm), 2700)
	if !strings.Contains(text, "DTSTART;TZID=Europe/Stockholm:20261020T001500\r\n") || strings.Contains(text, "STARTS-AT-GATHERING") {
		t.Errorf("expected the series' own start, got:\n%s", text)
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

func TestEncodeThenParseKeepsRepeatingEvents(t *testing.T) {
	start := time.Date(2026, 10, 20, 10, 0, 0, 0, stockholm)
	rule := recurrence.Rule{Frequency: recurrence.Weekly, Interval: 1, Weekdays: []string{"TU", "TH"}, Count: 10}
	value := rule.RRULE(start, false)
	movedStart, movedEnd := time.Date(2026, 10, 29, 14, 0, 0, 0, stockholm), time.Date(2026, 10, 29, 16, 0, 0, 0, stockholm)
	skipped, attended := true, false
	series := db.ExportEvent{
		Event: db.Event{
			Title: "Lecture", StartsAt: start, EndsAt: start.Add(2 * time.Hour), Kind: db.EventKindAttendable,
			IsSkipped: true, RecurrenceRule: &value, RecurrenceTimeZone: new("Europe/Stockholm"),
		},
		UID: "lecture@jonska-calendar",
		Exceptions: []db.OccurrenceException{
			{OriginalStart: time.Date(2026, 10, 22, 10, 0, 0, 0, stockholm), IsCancelled: true},
			{OriginalStart: time.Date(2026, 10, 27, 10, 0, 0, 0, stockholm), IsSkipped: &attended},
			{OriginalStart: time.Date(2026, 10, 29, 10, 0, 0, 0, stockholm), StartsAt: &movedStart, EndsAt: &movedEnd, IsSkipped: &skipped},
		},
	}
	allDayStart := time.Date(2026, 10, 30, 0, 0, 0, 0, stockholm)
	yearly := recurrence.Rule{Frequency: recurrence.Yearly, Interval: 1, Until: new("2030-10-30")}.RRULE(allDayStart, true)
	birthday := db.ExportEvent{
		Event: db.Event{
			Title: "Birthday", StartsAt: allDayStart, EndsAt: allDayStart.AddDate(0, 0, 1), IsAllDay: true,
			Kind: db.EventKindInformational, RecurrenceRule: &yearly, RecurrenceTimeZone: new("Europe/Stockholm"),
		},
		UID:        "birthday@jonska-calendar",
		Exceptions: []db.OccurrenceException{{OriginalStart: allDayStart.AddDate(1, 0, 0), IsCancelled: true}},
	}

	var file strings.Builder
	if err := Encode(&file, []db.ExportEvent{series, birthday}, time.UTC, time.Now()); err != nil {
		t.Fatal(err)
	}
	text := file.String()
	for _, want := range []string{
		"DTSTART;TZID=Europe/Stockholm:20261020T100000\r\n",
		"RRULE:" + value + "\r\n",
		"EXDATE;TZID=Europe/Stockholm:20261022T100000\r\n",
		"RECURRENCE-ID;TZID=Europe/Stockholm:20261029T100000\r\n",
		"DTSTART;VALUE=DATE:20261030\r\n",
		"EXDATE;VALUE=DATE:20271030\r\n",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("expected %q in:\n%s", want, text)
		}
	}

	// The user's zone differs from the series', which must not matter.
	result, err := Parse(strings.NewReader(text), time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Events) != 2 || len(result.Skipped) != 0 || len(result.Warnings) != 0 {
		t.Fatalf("unexpected result %+v", result)
	}
	// Times with a TZID keep their zone, whatever the user's.
	got, want := result.Events[0], series
	wantRule, err := recurrence.FromRRULE(*want.RecurrenceRule, want.StartsAt.In(stockholm))
	if err != nil {
		t.Fatal(err)
	}
	if got.Recurrence == nil || !reflect.DeepEqual(*got.Recurrence, wantRule) || got.TimeZone != "Europe/Stockholm" ||
		!got.StartsAt.Equal(want.StartsAt) || !got.EndsAt.Equal(want.EndsAt) {
		t.Fatalf("expected %+v, got %+v", want, got)
	}
	if len(got.Exceptions) != len(want.Exceptions) {
		t.Fatalf("expected exceptions %+v, got %+v", want.Exceptions, got.Exceptions)
	}
	for i, exception := range want.Exceptions {
		parsed := got.Exceptions[i]
		if !parsed.OriginalStart.Equal(exception.OriginalStart) || parsed.IsCancelled != exception.IsCancelled ||
			(exception.StartsAt == nil) != (parsed.StartsAt == nil) ||
			(exception.StartsAt != nil && !parsed.StartsAt.Equal(*exception.StartsAt)) ||
			!reflect.DeepEqual(parsed.IsSkipped, exception.IsSkipped) {
			t.Fatalf("exception %d: expected %+v, got %+v", i, exception, parsed)
		}
	}

	// Dates have no zone: an all-day series keeps its dates in the user's.
	day := result.Events[1]
	if day.Recurrence == nil || day.Recurrence.Frequency != recurrence.Yearly || *day.Recurrence.Until != "2030-10-30" ||
		day.TimeZone != "UTC" || !day.StartsAt.Equal(time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC)) ||
		len(day.Exceptions) != 1 || !day.Exceptions[0].OriginalStart.Equal(time.Date(2027, 10, 30, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected all-day series %+v", day)
	}
}
