package recurrence

import (
	"testing"
	"time"
)

func starts(t *testing.T, series Series, exceptions []Exception, from, to time.Time) []time.Time {
	t.Helper()
	occurrences, err := series.Occurrences(exceptions, from, to)
	if err != nil {
		t.Fatal(err)
	}
	var result []time.Time
	for _, occurrence := range occurrences {
		result = append(result, occurrence.StartsAt)
		if !occurrence.EndsAt.After(occurrence.StartsAt) {
			t.Fatalf("occurrence %+v does not end after it starts", occurrence)
		}
	}
	return result
}

func expectTimes(t *testing.T, got []time.Time, want ...time.Time) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	for index := range want {
		if !got[index].Equal(want[index]) {
			t.Fatalf("expected %v, got %v", want, got)
		}
	}
}

func day(year int, month time.Month, dayOfMonth, hour int) time.Time {
	return time.Date(year, month, dayOfMonth, hour, 0, 0, 0, stockholm)
}

func TestOccurrencesKeepWallClockAcrossDaylightSaving(t *testing.T) {
	series := Series{
		StartsAt: tuesday, EndsAt: tuesday.Add(90 * time.Minute),
		Rule: Rule{Frequency: Weekly, Interval: 1, Weekdays: []string{"TU"}}, Zone: stockholm,
	}
	got := starts(t, series, nil, day(2026, 10, 19, 0), day(2026, 11, 2, 0))
	// Summer time ends on 25 October: both are at 10:00 in Stockholm.
	expectTimes(t, got, day(2026, 10, 20, 10), day(2026, 10, 27, 10))
	if got[0].UTC().Hour() != 8 || got[1].UTC().Hour() != 9 {
		t.Fatalf("expected 08:00 and 09:00 UTC, got %v", got)
	}
}

func TestAllDayOccurrencesStayAtMidnight(t *testing.T) {
	series := Series{
		StartsAt: day(2026, 10, 23, 0), EndsAt: day(2026, 10, 25, 0), IsAllDay: true,
		Rule: Rule{Frequency: Daily, Interval: 1}, Zone: stockholm,
	}
	occurrences, err := series.Occurrences(nil, day(2026, 10, 25, 0), day(2026, 10, 26, 0))
	if err != nil {
		t.Fatal(err)
	}
	// The two-day occurrences starting on the 24th and 25th overlap the 25th;
	// the one starting on the 24th ends at midnight after the DST change.
	if len(occurrences) != 2 ||
		!occurrences[0].StartsAt.Equal(day(2026, 10, 24, 0)) || !occurrences[0].EndsAt.Equal(day(2026, 10, 26, 0)) ||
		!occurrences[1].StartsAt.Equal(day(2026, 10, 25, 0)) || !occurrences[1].EndsAt.Equal(day(2026, 10, 27, 0)) {
		t.Fatalf("unexpected occurrences %+v", occurrences)
	}
}

func TestOccurrencesFollowTheRule(t *testing.T) {
	tests := []struct {
		name     string
		start    time.Time
		isAllDay bool
		rule     Rule
		from, to time.Time
		want     []time.Time
	}{
		{
			name: "every other week on two days", start: tuesday,
			rule: Rule{Frequency: Weekly, Interval: 2, Weekdays: []string{"TU", "TH"}},
			from: day(2026, 10, 12, 0), to: day(2026, 11, 2, 0),
			want: []time.Time{day(2026, 10, 13, 10), day(2026, 10, 15, 10), day(2026, 10, 27, 10), day(2026, 10, 29, 10)},
		},
		{
			name: "the 2nd Tuesday", start: tuesday,
			rule: Rule{Frequency: Monthly, Interval: 1, MonthlyBy: Weekday},
			from: day(2026, 11, 1, 0), to: day(2027, 1, 1, 0),
			want: []time.Time{day(2026, 11, 10, 10), day(2026, 12, 8, 10)},
		},
		{
			name: "the last Friday", start: day(2026, 10, 30, 18),
			rule: Rule{Frequency: Monthly, Interval: 1, MonthlyBy: LastWeekday},
			from: day(2026, 11, 1, 0), to: day(2027, 1, 1, 0),
			want: []time.Time{day(2026, 11, 27, 18), day(2026, 12, 25, 18)},
		},
		{
			name: "the 31st skips shorter months", start: day(2026, 10, 31, 9),
			rule: Rule{Frequency: Monthly, Interval: 1, MonthlyBy: DayOfMonth},
			from: day(2026, 10, 1, 0), to: day(2027, 2, 1, 0),
			want: []time.Time{day(2026, 10, 31, 9), day(2026, 12, 31, 9), day(2027, 1, 31, 9)},
		},
		{
			name: "29 February only in leap years", start: day(2028, 2, 29, 0), isAllDay: true,
			rule: Rule{Frequency: Yearly, Interval: 1},
			from: day(2028, 1, 1, 0), to: day(2033, 1, 1, 0),
			want: []time.Time{day(2028, 2, 29, 0), day(2032, 2, 29, 0)},
		},
		{
			name: "count", start: tuesday,
			rule: Rule{Frequency: Daily, Interval: 1, Count: 3},
			from: day(2026, 10, 1, 0), to: day(2026, 11, 1, 0),
			want: []time.Time{day(2026, 10, 13, 10), day(2026, 10, 14, 10), day(2026, 10, 15, 10)},
		},
		{
			name: "until includes its date", start: tuesday,
			rule: Rule{Frequency: Daily, Interval: 1, Until: new("2026-10-14")},
			from: day(2026, 10, 1, 0), to: day(2026, 11, 1, 0),
			want: []time.Time{day(2026, 10, 13, 10), day(2026, 10, 14, 10)},
		},
		{
			name: "an occurrence that started before the window", start: tuesday,
			rule: Rule{Frequency: Daily, Interval: 1},
			from: day(2026, 10, 14, 11), to: day(2026, 10, 14, 12),
			want: []time.Time{day(2026, 10, 14, 10)},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			end := test.start.Add(2 * time.Hour)
			if test.isAllDay {
				end = test.start.AddDate(0, 0, 1)
			}
			series := Series{StartsAt: test.start, EndsAt: end, IsAllDay: test.isAllDay, Rule: test.rule, Zone: stockholm}
			expectTimes(t, starts(t, series, nil, test.from, test.to), test.want...)
		})
	}
}

func TestOccurrencesApplyExceptions(t *testing.T) {
	series := Series{
		StartsAt: tuesday, EndsAt: tuesday.Add(time.Hour),
		Rule: Rule{Frequency: Daily, Interval: 1}, Zone: stockholm,
	}
	skipped := true
	movedIn, movedInEnd := day(2026, 10, 16, 15), day(2026, 10, 16, 16)
	movedOut, movedOutEnd := day(2026, 10, 30, 10), day(2026, 10, 30, 11)
	exceptions := []Exception{
		{OriginalStart: day(2026, 10, 14, 10), IsCancelled: true},
		{OriginalStart: day(2026, 10, 15, 10), IsSkipped: &skipped},
		// Moved from outside the window into it.
		{OriginalStart: day(2026, 10, 20, 10), StartsAt: &movedIn, EndsAt: &movedInEnd},
		// Moved from inside the window out of it.
		{OriginalStart: day(2026, 10, 16, 10), StartsAt: &movedOut, EndsAt: &movedOutEnd},
		// Not an occurrence of the rule, so ignored.
		{OriginalStart: day(2026, 10, 16, 12), StartsAt: &movedIn, EndsAt: &movedInEnd},
	}
	occurrences, err := series.Occurrences(exceptions, day(2026, 10, 14, 0), day(2026, 10, 17, 0))
	if err != nil {
		t.Fatal(err)
	}
	if len(occurrences) != 2 {
		t.Fatalf("expected two occurrences, got %+v", occurrences)
	}
	if first := occurrences[0]; !first.StartsAt.Equal(day(2026, 10, 15, 10)) || first.IsSkipped == nil || !*first.IsSkipped {
		t.Fatalf("expected the skipped 15th first, got %+v", first)
	}
	if moved := occurrences[1]; !moved.StartsAt.Equal(movedIn) || !moved.OriginalStart.Equal(day(2026, 10, 20, 10)) {
		t.Fatalf("expected the occurrence moved in, got %+v", moved)
	}
}

func TestIsOccurrenceAndLastEnd(t *testing.T) {
	series := Series{
		StartsAt: tuesday, EndsAt: tuesday.Add(time.Hour),
		Rule: Rule{Frequency: Weekly, Interval: 1, Weekdays: []string{"TU"}, Count: 3}, Zone: stockholm,
	}
	for at, want := range map[time.Time]bool{
		day(2026, 10, 20, 10): true,
		day(2026, 10, 20, 11): false,
		day(2026, 10, 21, 10): false,
		day(2026, 11, 3, 10):  false, // after the count
	} {
		got, err := series.IsOccurrence(at)
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Errorf("%v: expected %v, got %v", at, want, got)
		}
	}

	last, ends, err := series.LastEnd(nil)
	if err != nil || !ends || !last.Equal(day(2026, 10, 27, 11)) {
		t.Fatalf("expected the last end on the 27th at 11:00, got %v %v %v", last, ends, err)
	}
	movedEnd := day(2026, 12, 1, 12)
	movedStart := movedEnd.Add(-time.Hour)
	last, _, _ = series.LastEnd([]Exception{{OriginalStart: day(2026, 10, 27, 10), StartsAt: &movedStart, EndsAt: &movedEnd}})
	if !last.Equal(movedEnd) {
		t.Fatalf("expected a moved occurrence to extend the end, got %v", last)
	}

	series.Rule.Count = 0
	if _, ends, _ := series.LastEnd(nil); ends {
		t.Fatal("a rule without count or until never ends")
	}
}
