package recurrence

import (
	"slices"
	"time"

	"github.com/teambition/rrule-go"
)

// Series is a repeating event: its first occurrence, its rule, and the time
// zone its wall-clock times repeat in, so a 10:00 event stays at 10:00 across
// daylight saving changes.
type Series struct {
	StartsAt time.Time
	EndsAt   time.Time
	IsAllDay bool
	Rule     Rule
	Zone     *time.Location
}

// Exception is an occurrence that differs from its series, identified by the
// start the rule gives it.
type Exception struct {
	OriginalStart time.Time
	IsCancelled   bool
	// StartsAt and EndsAt are set together when the occurrence was moved.
	StartsAt  *time.Time
	EndsAt    *time.Time
	IsSkipped *bool
}

// Occurrence is one occurrence of a series as it happens. IsSkipped is nil
// when it follows the series.
type Occurrence struct {
	OriginalStart time.Time
	StartsAt      time.Time
	EndsAt        time.Time
	IsSkipped     *bool
}

func (series Series) rrule() (*rrule.RRule, error) {
	return rrule.NewRRule(series.Rule.options(series.StartsAt.In(series.Zone)))
}

// end is when an occurrence that starts at start ends. All-day occurrences
// last the same number of days; timed ones the same length of time.
func (series Series) end(start time.Time) time.Time {
	if series.IsAllDay {
		first := series.StartsAt.In(series.Zone)
		last := series.EndsAt.In(series.Zone)
		days := int(dateOf(last).Sub(dateOf(first)).Hours() / 24)
		return start.AddDate(0, 0, days)
	}
	return start.Add(series.EndsAt.Sub(series.StartsAt))
}

// Occurrences returns the occurrences that overlap [from, to), in order of
// their start. Cancelled occurrences are left out, and moved ones are
// returned at their new time.
func (series Series) Occurrences(exceptions []Exception, from, to time.Time) ([]Occurrence, error) {
	rule, err := series.rrule()
	if err != nil {
		return nil, err
	}
	byStart := make(map[time.Time]Exception, len(exceptions))
	for _, exception := range exceptions {
		byStart[exception.OriginalStart.UTC()] = exception
	}
	var occurrences []Occurrence
	add := func(originalStart time.Time) {
		occurrence := Occurrence{OriginalStart: originalStart, StartsAt: originalStart, EndsAt: series.end(originalStart)}
		if exception, ok := byStart[originalStart.UTC()]; ok {
			if exception.IsCancelled {
				return
			}
			if exception.StartsAt != nil && exception.EndsAt != nil {
				occurrence.StartsAt, occurrence.EndsAt = *exception.StartsAt, *exception.EndsAt
			}
			occurrence.IsSkipped = exception.IsSkipped
		}
		if occurrence.StartsAt.Before(to) && occurrence.EndsAt.After(from) {
			occurrences = append(occurrences, occurrence)
		}
	}

	// An occurrence that starts before from can still overlap it. The extra
	// day covers all-day occurrences that grow by an hour across a daylight
	// saving change.
	length := series.end(series.StartsAt).Sub(series.StartsAt) + 24*time.Hour
	seen := map[time.Time]bool{}
	for _, start := range rule.Between(from.Add(-length), to, true) {
		seen[start.UTC()] = true
		add(start)
	}
	// A moved occurrence can land in the window from outside it.
	var moved []time.Time
	for _, exception := range exceptions {
		if !seen[exception.OriginalStart.UTC()] && exception.StartsAt != nil {
			moved = append(moved, exception.OriginalStart)
		}
	}
	occurs := occursAmong(rule, moved)
	for _, start := range moved {
		if occurs[start.UTC()] {
			add(start.In(series.Zone))
		}
	}
	slices.SortFunc(occurrences, func(a, b Occurrence) int { return a.StartsAt.Compare(b.StartsAt) })
	return occurrences, nil
}

// IsOccurrence reports whether the rule gives an occurrence that starts at
// originalStart.
func (series Series) IsOccurrence(originalStart time.Time) (bool, error) {
	rule, err := series.rrule()
	if err != nil {
		return false, err
	}
	return series.isOccurrence(rule, originalStart), nil
}

func (series Series) isOccurrence(rule *rrule.RRule, originalStart time.Time) bool {
	found := rule.Between(originalStart, originalStart, true)
	return len(found) == 1 && found[0].Equal(originalStart)
}

// OccursAmong returns which of starts, keyed in UTC, the rule gives an
// occurrence at.
func (series Series) OccursAmong(starts []time.Time) (map[time.Time]bool, error) {
	rule, err := series.rrule()
	if err != nil {
		return nil, err
	}
	return occursAmong(rule, starts), nil
}

// occursAmong checks all starts in one walk of the rule. The rule is walked
// from its first occurrence each time, so checking starts one by one would
// repeat that walk for each.
func occursAmong(rule *rrule.RRule, starts []time.Time) map[time.Time]bool {
	found := map[time.Time]bool{}
	if len(starts) == 0 {
		return found
	}
	wanted := make(map[time.Time]bool, len(starts))
	earliest, latest := starts[0], starts[0]
	for _, start := range starts {
		wanted[start.UTC()] = true
		earliest, latest = minTime(earliest, start), maxTime(latest, start)
	}
	for _, start := range rule.Between(earliest, latest, true) {
		if wanted[start.UTC()] {
			found[start.UTC()] = true
		}
	}
	return found
}

func minTime(a, b time.Time) time.Time {
	if b.Before(a) {
		return b
	}
	return a
}

func maxTime(a, b time.Time) time.Time {
	if b.After(a) {
		return b
	}
	return a
}

// LastEnd is when the last occurrence ends, counting moved occurrences, or
// false when the series never ends.
func (series Series) LastEnd(exceptions []Exception) (time.Time, bool, error) {
	if series.Rule.Count == 0 && series.Rule.Until == nil {
		return time.Time{}, false, nil
	}
	rule, err := series.rrule()
	if err != nil {
		return time.Time{}, false, err
	}
	starts := rule.All()
	if len(starts) == 0 {
		return series.EndsAt, true, nil
	}
	last := series.end(starts[len(starts)-1])
	for _, exception := range exceptions {
		if exception.EndsAt != nil && exception.EndsAt.After(last) {
			last = *exception.EndsAt
		}
	}
	return last, true, nil
}

func dateOf(at time.Time) time.Time {
	return time.Date(at.Year(), at.Month(), at.Day(), 0, 0, 0, 0, time.UTC)
}
