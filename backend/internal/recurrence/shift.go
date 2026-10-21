package recurrence

import (
	"slices"
	"time"
)

// Shift moves at by the calendar days and wall-clock time between from and
// to, in zone. Moving a series by how one of its occurrences moved keeps the
// others at the same local times across daylight saving changes.
func Shift(at, from, to time.Time, zone *time.Location) time.Time {
	at, from, to = at.In(zone), from.In(zone), to.In(zone)
	days := DaysBetween(from, to)
	seconds := clockSeconds(to) - clockSeconds(from)
	return time.Date(at.Year(), at.Month(), at.Day()+days, at.Hour(), at.Minute(), at.Second()+seconds, 0, zone)
}

// DaysBetween is the number of calendar days from a's date to b's, in the
// zone they are given in.
func DaysBetween(a, b time.Time) int {
	return int(dateOf(b).Sub(dateOf(a)).Hours() / 24)
}

func clockSeconds(at time.Time) int {
	return at.Hour()*3600 + at.Minute()*60 + at.Second()
}

// ShiftWeekdays moves a weekly rule's weekdays by days, so a series moved
// from Tuesdays to Wednesdays keeps repeating on the day it moved to.
func (rule Rule) ShiftWeekdays(days int) Rule {
	if rule.Frequency != Weekly || days%7 == 0 {
		return rule
	}
	moved := make([]string, len(rule.Weekdays))
	for index, day := range rule.Weekdays {
		moved[index] = weekdayCodes[((slices.Index(weekdayCodes, day)+days)%7+7)%7]
	}
	// Keep them in week order, as Validate does.
	shifted := slices.DeleteFunc(slices.Clone(weekdayCodes), func(code string) bool {
		return !slices.Contains(moved, code)
	})
	rule.Weekdays = shifted
	return rule
}

// EndingBefore is the rule cut off before the occurrence that starts at
// at: it ends on the day before, which holds because an occurrence repeats
// at most once a day.
func (rule Rule) EndingBefore(at time.Time) Rule {
	lastDay := at.AddDate(0, 0, -1).Format(dateLayout)
	rule.Until, rule.Count = &lastDay, 0
	return rule
}

// CountBefore is how many occurrences the rule gives before at.
func (series Series) CountBefore(at time.Time) (int, error) {
	rule, err := series.rrule()
	if err != nil {
		return 0, err
	}
	return len(rule.Between(series.StartsAt, at.Add(-time.Second), true)), nil
}

// OccurrenceEnd is when an occurrence that starts at start ends.
func (series Series) OccurrenceEnd(start time.Time) time.Time {
	return series.end(start)
}
