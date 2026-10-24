package ical

import (
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jonathan-skantz/calendar/backend/db"
	"github.com/jonathan-skantz/calendar/backend/internal/recurrence"
)

var stockholm = mustLoad("Europe/Stockholm")

func mustLoad(name string) *time.Location {
	zone, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return zone
}

// calendar wraps VEVENT bodies in a calendar, with CRLF line endings as in
// real files.
func calendar(events ...string) string {
	lines := []string{"BEGIN:VCALENDAR", "VERSION:2.0", "PRODID:-//Test//EN"}
	for _, event := range events {
		lines = append(lines, "BEGIN:VEVENT")
		lines = append(lines, strings.Split(strings.TrimSpace(event), "\n")...)
		lines = append(lines, "END:VEVENT")
	}
	lines = append(lines, "END:VCALENDAR")
	return strings.Join(lines, "\r\n") + "\r\n"
}

func parseOne(t *testing.T, event string) Event {
	t.Helper()
	result, err := Parse(strings.NewReader(calendar(event)), stockholm)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Events) != 1 {
		t.Fatalf("expected one event, got %+v", result)
	}
	return result.Events[0]
}

func TestParseTimes(t *testing.T) {
	tests := []struct {
		name       string
		event      string
		wantStart  time.Time
		wantEnd    time.Time
		wantAllDay bool
	}{
		{
			name:      "UTC",
			event:     "DTSTART:20261024T080000Z\nDTEND:20261024T093000Z",
			wantStart: time.Date(2026, 10, 24, 8, 0, 0, 0, time.UTC),
			wantEnd:   time.Date(2026, 10, 24, 9, 30, 0, 0, time.UTC),
		},
		{
			name:      "IANA TZID",
			event:     "DTSTART;TZID=America/New_York:20261024T090000\nDTEND;TZID=America/New_York:20261024T100000",
			wantStart: time.Date(2026, 10, 24, 13, 0, 0, 0, time.UTC),
			wantEnd:   time.Date(2026, 10, 24, 14, 0, 0, 0, time.UTC),
		},
		{
			name:      "Windows TZID from Outlook",
			event:     "DTSTART;TZID=W. Europe Standard Time:20261024T090000\nDTEND;TZID=W. Europe Standard Time:20261024T100000",
			wantStart: time.Date(2026, 10, 24, 7, 0, 0, 0, time.UTC),
			wantEnd:   time.Date(2026, 10, 24, 8, 0, 0, 0, time.UTC),
		},
		{
			name:      "floating time is in the user's zone",
			event:     "DTSTART:20261024T090000\nDTEND:20261024T100000",
			wantStart: time.Date(2026, 10, 24, 9, 0, 0, 0, stockholm),
			wantEnd:   time.Date(2026, 10, 24, 10, 0, 0, 0, stockholm),
		},
		{
			name:      "duration instead of end",
			event:     "DTSTART:20261024T080000Z\nDURATION:PT1H15M",
			wantStart: time.Date(2026, 10, 24, 8, 0, 0, 0, time.UTC),
			wantEnd:   time.Date(2026, 10, 24, 9, 15, 0, 0, time.UTC),
		},
		{
			name:      "no end",
			event:     "DTSTART:20261024T080000Z",
			wantStart: time.Date(2026, 10, 24, 8, 0, 0, 0, time.UTC),
			wantEnd:   time.Date(2026, 10, 24, 8, 30, 0, 0, time.UTC),
		},
		{
			name:      "end at the start",
			event:     "DTSTART:20261024T080000Z\nDTEND:20261024T080000Z",
			wantStart: time.Date(2026, 10, 24, 8, 0, 0, 0, time.UTC),
			wantEnd:   time.Date(2026, 10, 24, 8, 30, 0, 0, time.UTC),
		},
		{
			name:       "all-day with exclusive end",
			event:      "DTSTART;VALUE=DATE:20261024\nDTEND;VALUE=DATE:20261026",
			wantStart:  time.Date(2026, 10, 24, 0, 0, 0, 0, stockholm),
			wantEnd:    time.Date(2026, 10, 26, 0, 0, 0, 0, stockholm),
			wantAllDay: true,
		},
		{
			name:       "all-day without end lasts one day",
			event:      "DTSTART;VALUE=DATE:20261024",
			wantStart:  time.Date(2026, 10, 24, 0, 0, 0, 0, stockholm),
			wantEnd:    time.Date(2026, 10, 25, 0, 0, 0, 0, stockholm),
			wantAllDay: true,
		},
		{
			name:       "all-day duration keeps midnight across daylight saving",
			event:      "DTSTART;VALUE=DATE:20261024\nDURATION:P2D",
			wantStart:  time.Date(2026, 10, 24, 0, 0, 0, 0, stockholm),
			wantEnd:    time.Date(2026, 10, 26, 0, 0, 0, 0, stockholm),
			wantAllDay: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			event := parseOne(t, "UID:1\nSUMMARY:Test\n"+test.event)
			if !event.StartsAt.Equal(test.wantStart) || !event.EndsAt.Equal(test.wantEnd) || event.IsAllDay != test.wantAllDay {
				t.Fatalf("expected %v to %v (all-day %v), got %v to %v (all-day %v)",
					test.wantStart, test.wantEnd, test.wantAllDay, event.StartsAt, event.EndsAt, event.IsAllDay)
			}
		})
	}
}

func TestParseFields(t *testing.T) {
	event := parseOne(t, `
UID:abc@example.com
SUMMARY:Dinner\, with friends
DESCRIPTION:Bring wine\nand cheese
LOCATION:Kungsgatan 1
URL:https://example.com/dinner
TRANSP:TRANSPARENT
CATEGORIES:Family, Food
CATEGORIES:Friends
DTSTART:20261024T160000Z
DTEND:20261024T190000Z`)
	if event.UID != "abc@example.com" || event.Title != "Dinner, with friends" ||
		*event.Description != "Bring wine\nand cheese" || *event.Location != "Kungsgatan 1" ||
		*event.URL != "https://example.com/dinner" || event.Kind != db.EventKindInformational {
		t.Fatalf("unexpected fields %+v", event)
	}
	if !slices.Equal(event.Categories, []string{"Family", "Food", "Friends"}) {
		t.Fatalf("unexpected categories %v", event.Categories)
	}

	plain := parseOne(t, "DTSTART:20261024T160000Z\nSUMMARY:  ")
	if plain.Title != "Untitled" || plain.Kind != db.EventKindAttendable ||
		plain.Description != nil || plain.UID != "" {
		t.Fatalf("unexpected defaults %+v", plain)
	}
}

func TestParseAppFields(t *testing.T) {
	event := parseOne(t, `
SUMMARY:Football
DTSTART:20261024T160000Z
DTEND:20261024T180000Z
X-JONSKA-START-APPROX:TRUE
X-JONSKA-END-APPROX:TRUE
X-JONSKA-COMMUTE-BEFORE:PT30M
X-JONSKA-COMMUTE-AFTER:PT45M
X-JONSKA-GATHERING:PT15M
X-JONSKA-GATHERING-OPTIONAL:TRUE
X-JONSKA-GATHERING-LABEL:Warm-up
X-JONSKA-SKIPPED:TRUE`)
	if !event.StartIsApproximate || !event.EndIsApproximate ||
		event.CommuteBeforeSeconds != 1800 || event.CommuteAfterSeconds != 2700 ||
		event.GatheringSeconds != 900 || !event.GatheringIsOptional ||
		event.GatheringLabel == nil || *event.GatheringLabel != "Warm-up" || !event.IsSkipped {
		t.Fatalf("unexpected app fields %+v", event.Event)
	}
}

func TestParseKeepsTheStartOfFilesWithoutTheGatheringStart(t *testing.T) {
	// Files written before DTSTART moved to the gathering.
	event := parseOne(t, `
SUMMARY:Match
DTSTART:20261024T160000Z
DTEND:20261024T180000Z
X-JONSKA-GATHERING:PT15M`)
	if !event.StartsAt.Equal(time.Date(2026, 10, 24, 16, 0, 0, 0, time.UTC)) || event.GatheringSeconds != 900 {
		t.Fatalf("expected the start as written, got %+v", event.Event)
	}
}

func TestParseReminders(t *testing.T) {
	event := parseOne(t, `
SUMMARY:Meeting
DTSTART:20261024T100000Z
DTEND:20261024T110000Z
BEGIN:VALARM
ACTION:DISPLAY
TRIGGER:-PT15M
END:VALARM
BEGIN:VALARM
ACTION:DISPLAY
TRIGGER;VALUE=DATE-TIME:20261023T100000Z
END:VALARM
BEGIN:VALARM
ACTION:DISPLAY
TRIGGER;RELATED=END:-PT1H
END:VALARM
BEGIN:VALARM
ACTION:DISPLAY
TRIGGER:-PT90S
END:VALARM
BEGIN:VALARM
ACTION:DISPLAY
TRIGGER:PT5M
END:VALARM`)
	if want := []int{900, 86400, 0, 60}; !slices.Equal(event.ReminderOffsets, want) {
		t.Fatalf("expected offsets %v, got %v", want, event.ReminderOffsets)
	}

	allDay := parseOne(t, `
SUMMARY:Birthday
DTSTART;VALUE=DATE:20261024
BEGIN:VALARM
ACTION:DISPLAY
TRIGGER:-PT15H
END:VALARM`)
	if want := []int{15 * 3600}; !slices.Equal(allDay.ReminderOffsets, want) {
		t.Fatalf("all-day: expected offsets %v, got %v", want, allDay.ReminderOffsets)
	}
}

func TestParseSkipsAndWarns(t *testing.T) {
	result, err := Parse(strings.NewReader(calendar(
		"SUMMARY:Hourly\nDTSTART:20261024T100000Z\nRRULE:FREQ=HOURLY",
		"SUMMARY:Extra dates\nDTSTART:20261024T100000Z\nRDATE:20261025T100000Z",
		"UID:elsewhere\nSUMMARY:Moved occurrence\nDTSTART:20261031T110000Z\nRECURRENCE-ID:20261031T100000Z",
		"SUMMARY:Called off\nDTSTART:20261024T100000Z\nSTATUS:CANCELLED",
		"SUMMARY:No start",
		"SUMMARY:Mixed\nDTSTART;VALUE=DATE:20261024\nDTEND:20261024T100000Z",
		"SUMMARY:Odd zone\nDTSTART;TZID=Somewhere/Else:20261024T100000\nBEGIN:VALARM\nTRIGGER:PT5M\nEND:VALARM",
	)), stockholm)
	if err != nil {
		t.Fatal(err)
	}
	wantSkipped := []Skipped{
		{Title: "Hourly", Reason: ReasonUnsupportedRule},
		{Title: "Extra dates", Reason: ReasonUnsupportedRule},
		{Title: "Called off", Reason: ReasonCancelled},
		{Title: "No start", Reason: ReasonNoStart},
		{Title: "Mixed", Reason: ReasonBadTime},
		{Title: "Moved occurrence", Reason: ReasonOrphanOccurrence},
	}
	if !slices.Equal(result.Skipped, wantSkipped) {
		t.Fatalf("expected skipped %+v, got %+v", wantSkipped, result.Skipped)
	}
	if len(result.Events) != 1 || !result.Events[0].StartsAt.Equal(time.Date(2026, 10, 24, 10, 0, 0, 0, stockholm)) {
		t.Fatalf("expected the odd zone event read in the user's zone, got %+v", result.Events)
	}
	if len(result.Warnings) != 2 || !strings.HasPrefix(result.Warnings[0], "Odd zone: unknown time zone") ||
		!strings.HasPrefix(result.Warnings[1], "Odd zone: a reminder") {
		t.Fatalf("unexpected warnings %q", result.Warnings)
	}
}

func TestParseToleratesMalformedLinesAndOtherComponents(t *testing.T) {
	file := strings.Join([]string{
		"BEGIN:VCALENDAR",
		"VERSION:2.0",
		"X-WR-CALNAME:Work",
		"BEGIN:VTIMEZONE",
		"TZID:Europe/Stockholm",
		"END:VTIMEZONE",
		"BEGIN:VTODO",
		"SUMMARY:A task",
		"END:VTODO",
		"BEGIN:VEVENT",
		"SUMMARY:Long title that is folded",
		"  onto a second line",
		"this line has no colon",
		"DTSTART:20261024T100000Z",
		"END:VEVENT",
		"END:VCALENDAR",
	}, "\r\n")
	result, err := Parse(strings.NewReader(file), stockholm)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Events) != 1 || result.Events[0].Title != "Long title that is folded onto a second line" {
		t.Fatalf("unexpected result %+v", result)
	}
}

func TestParseRejectsInvalidFiles(t *testing.T) {
	if _, err := Parse(strings.NewReader("not a calendar"), stockholm); !errors.Is(err, ErrInvalidFile) {
		t.Fatalf("expected ErrInvalidFile, got %v", err)
	}
	events := make([]string, MaxEvents+1)
	for index := range events {
		events[index] = "DTSTART:20261024T100000Z"
	}
	if _, err := Parse(strings.NewReader(calendar(events...)), stockholm); !errors.Is(err, ErrTooManyEvents) {
		t.Fatalf("expected ErrTooManyEvents, got %v", err)
	}
}

func TestParseDuration(t *testing.T) {
	start := time.Date(2026, 10, 24, 12, 0, 0, 0, stockholm)
	for value, want := range map[string]time.Time{
		"PT15M":     start.Add(15 * time.Minute),
		"-PT1H30M":  start.Add(-90 * time.Minute),
		"+P1D":      start.AddDate(0, 0, 1),
		"P1W":       start.AddDate(0, 0, 7),
		"-P1DT2H":   start.AddDate(0, 0, -1).Add(-2 * time.Hour),
		"PT0S":      start,
		"P1DT0H30M": start.AddDate(0, 0, 1).Add(30 * time.Minute),
	} {
		length, err := parseDuration(value)
		if err != nil {
			t.Fatalf("%s: %v", value, err)
		}
		if got := length.addTo(start); !got.Equal(want) {
			t.Fatalf("%s: expected %v, got %v", value, want, got)
		}
	}
	for _, value := range []string{"", "P", "PT", "15M", "P1H", "PT1D", "P-1D", "PT15"} {
		if _, err := parseDuration(value); err == nil {
			t.Fatalf("%q: expected an error", value)
		}
	}
}

func TestParseRecurringEvents(t *testing.T) {
	result, err := Parse(strings.NewReader(calendar(
		`UID:lecture@example.com
SUMMARY:Lecture
DTSTART;TZID=Europe/Stockholm:20261020T100000
DTEND;TZID=Europe/Stockholm:20261020T120000
RRULE:FREQ=WEEKLY;BYDAY=TU,TH;UNTIL=20261231T225959Z
EXDATE;TZID=Europe/Stockholm:20261022T100000,20261027T100000`,
		`UID:lecture@example.com
SUMMARY:Lecture
RECURRENCE-ID;TZID=Europe/Stockholm:20261029T100000
DTSTART;TZID=Europe/Stockholm:20261029T140000
DTEND;TZID=Europe/Stockholm:20261029T160000`,
		`UID:lecture@example.com
SUMMARY:Lecture
RECURRENCE-ID;TZID=Europe/Stockholm:20261103T100000
STATUS:CANCELLED`,
		`UID:lecture@example.com
SUMMARY:Guest lecture
RECURRENCE-ID;TZID=Europe/Stockholm:20261105T100000
DTSTART;TZID=Europe/Stockholm:20261105T100000
DTEND;TZID=Europe/Stockholm:20261105T120000
X-JONSKA-SKIPPED:TRUE`,
		`UID:birthday@example.com
SUMMARY:Birthday
DTSTART;VALUE=DATE:20261030
RRULE:FREQ=YEARLY
EXDATE;VALUE=DATE:20271030`,
	)), stockholm)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Events) != 2 || len(result.Skipped) != 0 {
		t.Fatalf("expected two series, got %+v", result)
	}
	if len(result.Warnings) != 1 || !strings.HasPrefix(result.Warnings[0], "Guest lecture: changes to one occurrence") {
		t.Fatalf("unexpected warnings %q", result.Warnings)
	}

	lecture := result.Events[0]
	until := "2026-12-31"
	wantRule := recurrence.Rule{Frequency: recurrence.Weekly, Interval: 1, Weekdays: []string{"TU", "TH"}, Until: &until}
	if lecture.Recurrence == nil || !reflect.DeepEqual(*lecture.Recurrence, wantRule) || lecture.TimeZone != "Europe/Stockholm" {
		t.Fatalf("unexpected rule %+v in %q", lecture.Recurrence, lecture.TimeZone)
	}
	at := func(day, hour int) time.Time { return time.Date(2026, 10, day, hour, 0, 0, 0, stockholm) }
	november := func(day, hour int) time.Time { return time.Date(2026, 11, day, hour, 0, 0, 0, stockholm) }
	exceptions := lecture.Exceptions
	if len(exceptions) != 5 {
		t.Fatalf("expected five exceptions, got %+v", exceptions)
	}
	for index, want := range []time.Time{at(22, 10), at(27, 10), at(29, 10), november(3, 10), november(5, 10)} {
		if !exceptions[index].OriginalStart.Equal(want) {
			t.Fatalf("exception %d: expected %v, got %+v", index, want, exceptions[index])
		}
	}
	if !exceptions[0].IsCancelled || !exceptions[1].IsCancelled || !exceptions[3].IsCancelled {
		t.Fatalf("expected excluded and cancelled occurrences, got %+v", exceptions)
	}
	if moved := exceptions[2]; moved.StartsAt == nil || !moved.StartsAt.Equal(at(29, 14)) || !moved.EndsAt.Equal(at(29, 16)) {
		t.Fatalf("expected the 29th moved to 14:00, got %+v", moved)
	}
	if retitled := exceptions[4]; retitled.StartsAt != nil || retitled.IsSkipped == nil || !*retitled.IsSkipped {
		t.Fatalf("expected the 5th only skipped, got %+v", retitled)
	}

	birthday := result.Events[1]
	if birthday.Recurrence == nil || birthday.Recurrence.Frequency != recurrence.Yearly || birthday.TimeZone != "Europe/Stockholm" ||
		len(birthday.Exceptions) != 1 || !birthday.Exceptions[0].OriginalStart.Equal(time.Date(2027, 10, 30, 0, 0, 0, 0, stockholm)) {
		t.Fatalf("unexpected birthday %+v", birthday)
	}
}

func TestParseReadsCalendarName(t *testing.T) {
	for name, test := range map[string]struct {
		properties []string
		want       string
	}{
		"none":            {nil, ""},
		"x-wr-calname":    {[]string{"X-WR-CALNAME: School "}, "School"},
		"name wins":       {[]string{"X-WR-CALNAME:Old", "NAME:New"}, "New"},
		"name either way": {[]string{"NAME:New", "X-WR-CALNAME:Old"}, "New"},
	} {
		t.Run(name, func(t *testing.T) {
			lines := append([]string{"BEGIN:VCALENDAR", "VERSION:2.0", "PRODID:-//Test//EN"}, test.properties...)
			file := strings.Join(append(lines, "END:VCALENDAR"), "\r\n") + "\r\n"
			result, err := Parse(strings.NewReader(file), stockholm)
			if err != nil {
				t.Fatal(err)
			}
			if result.Name != test.want {
				t.Fatalf("expected %q, got %q", test.want, result.Name)
			}
		})
	}
}

func TestParseKeepsAtMostMaxExceptions(t *testing.T) {
	var excluded []string
	for i := range MaxExceptions + 500 {
		excluded = append(excluded, time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC).AddDate(0, 0, i).Format("20060102T150405Z"))
	}
	result, err := Parse(strings.NewReader(calendar(`
UID:daily
SUMMARY:Daily
DTSTART:20260101T090000Z
DTEND:20260101T100000Z
RRULE:FREQ=DAILY
EXDATE:`+strings.Join(excluded, ","))), stockholm)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Events) != 1 || len(result.Events[0].Exceptions) != MaxExceptions {
		t.Fatalf("expected one series with %d exceptions, got %+v", MaxExceptions, len(result.Events))
	}
	if len(result.Warnings) != 1 || !strings.Contains(result.Warnings[0], "only the first 1000") {
		t.Fatalf("expected a warning about the cap, got %v", result.Warnings)
	}
}
