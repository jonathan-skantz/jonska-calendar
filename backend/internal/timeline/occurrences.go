package timeline

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/jonathan-skantz/calendar/backend/db"
	"github.com/jonathan-skantz/calendar/backend/internal/recurrence"
)

var ErrInvalidTimeZone = errors.New("time_zone must be an IANA time zone")

// TimelineItem is a timeline entry as clients see it: an item, or one
// occurrence of a repeating event. An occurrence carries the start its rule
// gives it, which identifies it, and the series' rule. A received event the
// recipient customized carries what they received as Original.
type TimelineItem struct {
	db.TimelineEntry
	OccurrenceStart *time.Time       `json:"occurrence_start,omitempty"`
	Recurrence      *recurrence.Rule `json:"recurrence,omitempty"`
	Original        *ReceivedValues  `json:"original,omitempty"`
}

// ExpandTimeline turns the series among entries into their occurrences in
// [from, to), applies the recipient's customizations to received events,
// and keeps every other entry. The result is ordered by time, like the
// store's.
func ExpandTimeline(entries []db.TimelineEntry, from, to time.Time) ([]TimelineItem, error) {
	// A customization may shift an occurrence into the range from just
	// outside it, so series expand that much around it.
	margin := db.MaxShiftSeconds * time.Second
	items := make([]TimelineItem, 0, len(entries))
	for _, entry := range entries {
		expanded := []TimelineItem{{TimelineEntry: entry}}
		if entry.RecurrenceRule != nil {
			var err error
			if expanded, err = expandSeries(entry, from.Add(-margin), to.Add(margin)); err != nil {
				return nil, fmt.Errorf("event %s: %w", entry.ID, err)
			}
		}
		for _, item := range expanded {
			customize(&item)
			if overlaps(item, from, to) {
				items = append(items, item)
			}
		}
	}
	slices.SortStableFunc(items, func(a, b TimelineItem) int {
		return itemTime(a).Compare(itemTime(b))
	})
	return items, nil
}

func expandSeries(entry db.TimelineEntry, from, to time.Time) ([]TimelineItem, error) {
	series, err := seriesOf(entry)
	if err != nil {
		return nil, err
	}
	occurrences, err := series.Occurrences(exceptionsOf(entry.Exceptions), from, to)
	if err != nil {
		return nil, err
	}
	items := make([]TimelineItem, 0, len(occurrences))
	for _, occurrence := range occurrences {
		item := TimelineItem{
			TimelineEntry:   entry,
			OccurrenceStart: &occurrence.OriginalStart,
			Recurrence:      &series.Rule,
		}
		item.StartsAt, item.EndsAt = &occurrence.StartsAt, &occurrence.EndsAt
		// Recipients never see the owner's skips; the store leaves them
		// out of a shared series' exceptions.
		if occurrence.IsSkipped != nil {
			item.IsSkipped = *occurrence.IsSkipped
		}
		items = append(items, item)
	}
	return items, nil
}

// overlaps reports whether an event overlaps [from, to). Tasks always do.
func overlaps(item TimelineItem, from, to time.Time) bool {
	if item.StartsAt == nil || item.EndsAt == nil {
		return true
	}
	return item.StartsAt.Before(to) && item.EndsAt.After(from)
}

func itemTime(item TimelineItem) time.Time {
	if item.StartsAt != nil {
		return *item.StartsAt
	}
	if item.DeadlineAt != nil {
		return *item.DeadlineAt
	}
	return time.Time{}
}

// seriesOf reads a stored series back. Stored rules were validated when they
// were saved.
func seriesOf(entry db.TimelineEntry) (recurrence.Series, error) {
	zone, err := time.LoadLocation(*entry.RecurrenceTimeZone)
	if err != nil {
		return recurrence.Series{}, err
	}
	start := entry.StartsAt.In(zone)
	rule, err := recurrence.FromRRULE(*entry.RecurrenceRule, start)
	if err != nil {
		return recurrence.Series{}, err
	}
	return recurrence.Series{StartsAt: start, EndsAt: *entry.EndsAt, IsAllDay: entry.IsAllDay, Rule: rule, Zone: zone}, nil
}

func exceptionsOf(stored []db.OccurrenceException) []recurrence.Exception {
	exceptions := make([]recurrence.Exception, len(stored))
	for index, exception := range stored {
		exceptions[index] = recurrence.Exception{
			OriginalStart: exception.OriginalStart, IsCancelled: exception.IsCancelled,
			StartsAt: exception.StartsAt, EndsAt: exception.EndsAt, IsSkipped: exception.IsSkipped,
		}
	}
	return exceptions
}

// applyRecurrence validates a rule for the event and stores it on the event
// as an RRULE value in timeZone, with the series' end given the exceptions.
// A nil rule makes the event a plain one.
func applyRecurrence(event db.Event, rule *recurrence.Rule, timeZone string, exceptions []db.OccurrenceException) (db.Event, error) {
	event.RecurrenceRule, event.RecurrenceTimeZone, event.RecurrenceEndsAt = nil, nil, nil
	if rule == nil {
		return event, nil
	}
	zone, err := ParseTimeZone(timeZone)
	if err != nil {
		return event, err
	}
	start := event.StartsAt.In(zone)
	validated, err := recurrence.Validate(*rule, start)
	if err != nil {
		return event, err
	}
	series := recurrence.Series{StartsAt: start, EndsAt: event.EndsAt, IsAllDay: event.IsAllDay, Rule: validated, Zone: zone}
	lastEnd, ends, err := series.LastEnd(exceptionsOf(exceptions))
	if err != nil {
		return event, err
	}
	value := validated.RRULE(start, event.IsAllDay)
	event.RecurrenceRule, event.RecurrenceTimeZone = &value, &timeZone
	if ends {
		event.RecurrenceEndsAt = &lastEnd
	}
	return event, nil
}

// matchingExceptions keeps the exceptions that are still occurrences of the
// event's series.
func matchingExceptions(event db.Event, exceptions []db.OccurrenceException) ([]db.OccurrenceException, error) {
	if event.RecurrenceRule == nil {
		return nil, nil
	}
	series, err := seriesOf(db.TimelineEntry{
		StartsAt: &event.StartsAt, EndsAt: &event.EndsAt, IsAllDay: event.IsAllDay,
		RecurrenceRule: event.RecurrenceRule, RecurrenceTimeZone: event.RecurrenceTimeZone,
	})
	if err != nil {
		return nil, err
	}
	// Exceptions outside the supported years are dropped unchecked, so they
	// cannot stretch the walk below.
	var starts []time.Time
	for _, exception := range exceptions {
		if recurrence.InYearRange(exception.OriginalStart) {
			starts = append(starts, exception.OriginalStart)
		}
	}
	occurs, err := series.OccursAmong(starts)
	if err != nil {
		return nil, err
	}
	var kept []db.OccurrenceException
	for _, exception := range exceptions {
		if occurs[exception.OriginalStart.UTC()] {
			kept = append(kept, exception)
		}
	}
	return kept, nil
}

// ParseTimeZone reads the browser's IANA time zone. "Local" and "" are
// valid for time.LoadLocation, but mean the server's zone and UTC.
func ParseTimeZone(name string) (*time.Location, error) {
	if name == "" || name == "Local" {
		return nil, ErrInvalidTimeZone
	}
	zone, err := time.LoadLocation(name)
	if err != nil {
		return nil, ErrInvalidTimeZone
	}
	return zone, nil
}

// isRecurrenceError reports whether err is a rule the user can fix.
func isRecurrenceError(err error) bool {
	for _, target := range []error{
		ErrInvalidTimeZone, recurrence.ErrInvalidRule, recurrence.ErrWeekdays,
		recurrence.ErrMonthlyBy, recurrence.ErrUntil, recurrence.ErrUntilTooLate,
	} {
		if errors.Is(err, target) {
			return true
		}
	}
	return false
}
