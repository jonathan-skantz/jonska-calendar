package recurrence

import (
	"slices"
	"testing"
	"time"
)

func TestShiftKeepsLocalTimesAcrossDaylightSaving(t *testing.T) {
	// The series starts in summer time; one occurrence after the change is
	// moved from 10:00 to 11:30 the next day.
	seriesStart := day(2026, 10, 13, 10)
	from := day(2026, 10, 27, 10)
	to := time.Date(2026, 10, 28, 11, 30, 0, 0, stockholm)
	got := Shift(seriesStart, from, to, stockholm)
	if want := time.Date(2026, 10, 14, 11, 30, 0, 0, stockholm); !got.Equal(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	if got := Shift(seriesStart, from, from, stockholm); !got.Equal(seriesStart) {
		t.Fatalf("no move must keep the start, got %v", got)
	}
	// Moving across midnight backwards.
	got = Shift(seriesStart, from, day(2026, 10, 26, 23), stockholm)
	if want := day(2026, 10, 12, 23); !got.Equal(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestShiftWeekdays(t *testing.T) {
	rule := Rule{Frequency: Weekly, Interval: 1, Weekdays: []string{"TU", "SU"}}
	if got := rule.ShiftWeekdays(1).Weekdays; !slices.Equal(got, []string{"MO", "WE"}) {
		t.Fatalf("one day later: expected MO, WE, got %v", got)
	}
	if got := rule.ShiftWeekdays(-2).Weekdays; !slices.Equal(got, []string{"FR", "SU"}) {
		t.Fatalf("two days earlier: expected FR, SU, got %v", got)
	}
	if got := rule.ShiftWeekdays(14).Weekdays; !slices.Equal(got, rule.Weekdays) {
		t.Fatalf("whole weeks keep the days, got %v", got)
	}
	daily := Rule{Frequency: Daily, Interval: 1}
	if got := daily.ShiftWeekdays(3); got.Weekdays != nil {
		t.Fatalf("only weekly rules have weekdays, got %v", got)
	}
}

func TestEndingBeforeAndCountBefore(t *testing.T) {
	series := Series{
		StartsAt: tuesday, EndsAt: tuesday.Add(time.Hour),
		Rule: Rule{Frequency: Daily, Interval: 1, Count: 10}, Zone: stockholm,
	}
	count, err := series.CountBefore(day(2026, 10, 16, 10))
	if err != nil || count != 3 {
		t.Fatalf("expected 3 occurrences before the 16th, got %d %v", count, err)
	}
	series.Rule = series.Rule.EndingBefore(day(2026, 10, 16, 10))
	if series.Rule.Count != 0 || *series.Rule.Until != "2026-10-15" {
		t.Fatalf("expected the rule to end on the 15th, got %+v", series.Rule)
	}
	expectTimes(t, starts(t, series, nil, day(2026, 10, 1, 0), day(2026, 11, 1, 0)),
		day(2026, 10, 13, 10), day(2026, 10, 14, 10), day(2026, 10, 15, 10))
	if end := series.OccurrenceEnd(day(2026, 10, 14, 10)); !end.Equal(day(2026, 10, 14, 11)) {
		t.Fatalf("expected the occurrence to end at 11:00, got %v", end)
	}
}
