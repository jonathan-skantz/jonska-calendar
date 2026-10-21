package timeline

import (
	"errors"
	"net/http"
	"net/url"
	"slices"
	"time"

	"github.com/google/uuid"

	"github.com/jonathan-skantz/calendar/backend/db"
	"github.com/jonathan-skantz/calendar/backend/internal/recurrence"
)

// A change to an occurrence of a repeating event applies to that occurrence
// only, to it and the following ones, or to all of them.
const (
	scopeThis      = "this"
	scopeFollowing = "following"
	scopeAll       = "all"
)

var (
	ErrInvalidScope       = errors.New("scope must be this, following or all, with occurrence as an RFC 3339 time")
	ErrNotRepeating       = errors.New("only repeating events have occurrences")
	ErrOccurrenceNotFound = errors.New("occurrence not found")
	ErrOccurrenceFields   = errors.New("only the time and skip can change for one occurrence")
)

// occurrenceScope is which occurrences of a series a request changes. An
// empty scope means the event as stored.
type occurrenceScope struct {
	scope      string
	occurrence time.Time
}

func parseScope(query url.Values) (occurrenceScope, error) {
	scope := query.Get("scope")
	if scope == "" && !query.Has("occurrence") {
		return occurrenceScope{}, nil
	}
	if scope != scopeThis && scope != scopeFollowing && scope != scopeAll {
		return occurrenceScope{}, ErrInvalidScope
	}
	occurrence, err := time.Parse(time.RFC3339, query.Get("occurrence"))
	if err != nil {
		return occurrenceScope{}, ErrInvalidScope
	}
	return occurrenceScope{scope: scope, occurrence: occurrence}, nil
}

// scopedSeries checks that the occurrence belongs to the series.
func scopedSeries(existing db.TimelineEntry, scope occurrenceScope) (recurrence.Series, error) {
	if existing.RecurrenceRule == nil {
		return recurrence.Series{}, ErrNotRepeating
	}
	series, err := seriesOf(existing)
	if err != nil {
		return recurrence.Series{}, err
	}
	ok, err := series.IsOccurrence(scope.occurrence)
	if err != nil {
		return recurrence.Series{}, err
	}
	if !ok {
		return recurrence.Series{}, ErrOccurrenceNotFound
	}
	return series, nil
}

// withSeries gives the event its rule, keeps the candidate exceptions that
// are occurrences of it, and sets the series' end with them.
func withSeries(event db.Event, rule *recurrence.Rule, timeZone string, candidates []db.OccurrenceException) (db.Event, []db.OccurrenceException, error) {
	withRule, err := applyRecurrence(event, rule, timeZone, nil)
	if err != nil {
		return event, nil, err
	}
	kept, err := matchingExceptions(withRule, candidates)
	if err != nil {
		return event, nil, err
	}
	withRule, err = applyRecurrence(event, rule, timeZone, kept)
	return withRule, kept, err
}

// updateOccurrences applies a change to the scope's occurrences. It returns
// the id of the new series that changing the following occurrences starts.
func (h *Handler) updateOccurrences(r *http.Request, userID uuid.UUID, existing db.TimelineEntry,
	event db.Event, request eventRequest, scope occurrenceScope) (uuid.UUID, error) {
	series, err := scopedSeries(existing, scope)
	if err != nil {
		return uuid.Nil, err
	}
	switch {
	case scope.scope == scopeThis:
		return uuid.Nil, h.updateOccurrence(r, userID, existing, series, event, scope.occurrence)
	// From the first occurrence on is the whole series.
	case scope.scope == scopeAll, scope.occurrence.Equal(*existing.StartsAt):
		return uuid.Nil, h.updateAllOccurrences(r, userID, existing, series, event, request, scope.occurrence)
	default:
		return h.updateFollowingOccurrences(r, userID, existing, series, event, request, scope.occurrence)
	}
}

// updateOccurrence moves or skips one occurrence. The rest of the event must
// be as the series has it.
func (h *Handler) updateOccurrence(r *http.Request, userID uuid.UUID, existing db.TimelineEntry,
	series recurrence.Series, event db.Event, occurrence time.Time) error {
	if !sameSeriesFields(existing.Event(), event) {
		return ErrOccurrenceFields
	}
	exception := db.OccurrenceException{OriginalStart: occurrence}
	if !event.StartsAt.Equal(occurrence) || !event.EndsAt.Equal(series.OccurrenceEnd(occurrence)) {
		exception.StartsAt, exception.EndsAt = &event.StartsAt, &event.EndsAt
	}
	if event.IsSkipped != existing.IsSkipped {
		exception.IsSkipped = &event.IsSkipped
	}
	return h.setException(r, userID, existing, series, exception)
}

// setException replaces the exception for one occurrence, keeping the
// series' end in step with it.
func (h *Handler) setException(r *http.Request, userID uuid.UUID, existing db.TimelineEntry,
	series recurrence.Series, exception db.OccurrenceException) error {
	exceptions := slices.DeleteFunc(slices.Clone(existing.Exceptions), func(other db.OccurrenceException) bool {
		return other.OriginalStart.Equal(exception.OriginalStart)
	})
	endsAt, err := seriesEnd(series, append(exceptions, exception))
	if err != nil {
		return err
	}
	return h.Store.SetOccurrenceException(r.Context(), userID, existing.ID, exception, endsAt)
}

// updateAllOccurrences changes the whole series from one of its occurrences:
// the series and its exceptions move as that occurrence moved.
func (h *Handler) updateAllOccurrences(r *http.Request, userID uuid.UUID, existing db.TimelineEntry,
	series recurrence.Series, event db.Event, request eventRequest, occurrence time.Time) error {
	movedTo := event.StartsAt
	rule := followMove(series, request.Recurrence, occurrence, movedTo)
	length := event.EndsAt.Sub(event.StartsAt)
	lengthDays := recurrence.DaysBetween(event.StartsAt.In(series.Zone), event.EndsAt.In(series.Zone))
	event.StartsAt = recurrence.Shift(*existing.StartsAt, occurrence, movedTo, series.Zone)
	event.EndsAt = event.StartsAt.Add(length)
	if event.IsAllDay {
		event.EndsAt = event.StartsAt.AddDate(0, 0, lengthDays)
	}
	moved := make([]db.OccurrenceException, len(existing.Exceptions))
	for index, exception := range existing.Exceptions {
		exception.OriginalStart = recurrence.Shift(exception.OriginalStart, occurrence, movedTo, series.Zone)
		moved[index] = exception
	}
	updated, kept, err := withSeries(event, rule, request.TimeZone, moved)
	if err != nil {
		return err
	}
	return h.Store.UpdateSeries(r.Context(), userID, existing.ID, updated, kept)
}

// updateFollowingOccurrences ends the series before the occurrence and
// starts a new one there with the request's fields. It returns the new
// series' id.
func (h *Handler) updateFollowingOccurrences(r *http.Request, userID uuid.UUID, existing db.TimelineEntry,
	series recurrence.Series, event db.Event, request eventRequest, occurrence time.Time) (uuid.UUID, error) {
	before := series
	before.Rule = series.Rule.EndingBefore(occurrence.In(series.Zone))
	var earlier, later []db.OccurrenceException
	for _, exception := range existing.Exceptions {
		switch {
		case exception.OriginalStart.Before(occurrence):
			earlier = append(earlier, exception)
		case exception.OriginalStart.After(occurrence):
			exception.OriginalStart = recurrence.Shift(exception.OriginalStart, occurrence, event.StartsAt, series.Zone)
			later = append(later, exception)
		}
	}
	beforeEnd, err := seriesEnd(before, earlier)
	if err != nil {
		return uuid.Nil, err
	}

	rule := followMove(series, request.Recurrence, occurrence, event.StartsAt)
	// A rule that still has the series' count goes on for the occurrences
	// that are left of it.
	if rule != nil && rule.Count > 0 && rule.Count == series.Rule.Count {
		done, err := series.CountBefore(occurrence)
		if err != nil {
			return uuid.Nil, err
		}
		left := *rule
		left.Count -= done
		rule = &left
	}
	after, kept, err := withSeries(event, rule, request.TimeZone, later)
	if err != nil {
		return uuid.Nil, err
	}
	return h.Store.SplitSeries(r.Context(), userID, existing.ID, occurrence,
		before.Rule.RRULE(before.StartsAt, before.IsAllDay), beforeEnd, after, kept)
}

// deleteOccurrences cancels one occurrence, or ends the series before it.
func (h *Handler) deleteOccurrences(r *http.Request, userID, eventID uuid.UUID, scope occurrenceScope) error {
	existing, err := h.Store.GetOwnEvent(r.Context(), userID, eventID)
	if err != nil {
		return err
	}
	series, err := scopedSeries(existing, scope)
	if err != nil {
		return err
	}
	if scope.scope == scopeThis {
		return h.setException(r, userID, existing, series,
			db.OccurrenceException{OriginalStart: scope.occurrence, IsCancelled: true})
	}
	if scope.occurrence.Equal(*existing.StartsAt) {
		return h.Store.DeleteTimelineItem(r.Context(), userID, existing.ID)
	}
	before := series
	before.Rule = series.Rule.EndingBefore(scope.occurrence.In(series.Zone))
	earlier := slices.DeleteFunc(slices.Clone(existing.Exceptions), func(exception db.OccurrenceException) bool {
		return !exception.OriginalStart.Before(scope.occurrence)
	})
	endsAt, err := seriesEnd(before, earlier)
	if err != nil {
		return err
	}
	return h.Store.EndSeries(r.Context(), userID, existing.ID, scope.occurrence,
		before.Rule.RRULE(before.StartsAt, before.IsAllDay), endsAt)
}

// followMove keeps a weekly series on the weekdays its occurrence moved to,
// unless the request chose other weekdays.
func followMove(series recurrence.Series, rule *recurrence.Rule, from, to time.Time) *recurrence.Rule {
	if rule == nil || rule.Frequency != recurrence.Weekly || series.Rule.Frequency != recurrence.Weekly {
		return rule
	}
	normalized, err := recurrence.Validate(*rule, series.StartsAt)
	if err != nil || !slices.Equal(normalized.Weekdays, series.Rule.Weekdays) {
		return rule
	}
	shifted := normalized.ShiftWeekdays(recurrence.DaysBetween(from.In(series.Zone), to.In(series.Zone)))
	return &shifted
}

func seriesEnd(series recurrence.Series, exceptions []db.OccurrenceException) (*time.Time, error) {
	lastEnd, ends, err := series.LastEnd(exceptionsOf(exceptions))
	if err != nil || !ends {
		return nil, err
	}
	return &lastEnd, nil
}

// sameSeriesFields reports whether an occurrence's fields, apart from its
// time and skip, are the series' own. An omitted main tag counts as the same.
func sameSeriesFields(series, occurrence db.Event) bool {
	return series.Title == occurrence.Title && equalText(series.Description, occurrence.Description) &&
		equalText(series.URL, occurrence.URL) && equalText(series.Location, occurrence.Location) &&
		series.Kind == occurrence.Kind && series.IsAllDay == occurrence.IsAllDay &&
		series.StartIsApproximate == occurrence.StartIsApproximate &&
		series.EndIsApproximate == occurrence.EndIsApproximate &&
		series.GatheringSeconds == occurrence.GatheringSeconds &&
		series.GatheringIsOptional == occurrence.GatheringIsOptional &&
		equalText(series.GatheringLabel, occurrence.GatheringLabel) &&
		series.CommuteBeforeSeconds == occurrence.CommuteBeforeSeconds &&
		series.CommuteAfterSeconds == occurrence.CommuteAfterSeconds &&
		(occurrence.MainTagID == uuid.Nil || occurrence.MainTagID == series.MainTagID)
}

// equalText compares optional text, where blank means not set.
func equalText(a, b *string) bool {
	a, b = normalizeOptionalText(a), normalizeOptionalText(b)
	return (a == nil && b == nil) || (a != nil && b != nil && *a == *b)
}
