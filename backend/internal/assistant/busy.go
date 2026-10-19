package assistant

import (
	"time"

	"github.com/jonathan-skantz/calendar/backend/db"
)

// occupiedSpan is the time an entry blocks, and false when it leaves the time
// free. It mirrors isBusy and getOccupiedRange in frontend/src/lib/timeline.ts,
// which draw the same rules in the week view. Both test suites run the cases in
// frontend/src/lib/testdata/occupied-spans.json, so change them together.
//
// Only attended events that are not skipped are busy. On timed ones, commute
// is busy, mandatory gathering is busy, and optional gathering only counts
// when there is a commute to it, because then the user plans to be there.
func occupiedSpan(entry db.TimelineEntry) (start, end time.Time, busy bool) {
	if entry.Type != "event" || entry.StartsAt == nil || entry.EndsAt == nil ||
		entry.Kind == nil || *entry.Kind != db.EventKindAttendable || entry.IsSkipped {
		return time.Time{}, time.Time{}, false
	}
	start, end = *entry.StartsAt, *entry.EndsAt
	if entry.IsAllDay {
		return start, end, true
	}
	gathering := seconds(entry.GatheringSeconds)
	switch {
	case entry.CommuteBeforeSeconds > 0:
		start = start.Add(-gathering - seconds(entry.CommuteBeforeSeconds))
	case !entry.GatheringIsOptional:
		start = start.Add(-gathering)
	}
	return start, end.Add(seconds(entry.CommuteAfterSeconds)), true
}

func seconds(value int) time.Duration {
	return time.Duration(value) * time.Second
}
