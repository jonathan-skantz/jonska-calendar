// Package ical reads and writes iCalendar (RFC 5545) files. It maps VEVENTs
// to the app's events and back. Tasks and attendees are not supported.
package ical

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	ics "github.com/arran4/golang-ical"

	"github.com/jonathan-skantz/calendar/backend/db"
	"github.com/jonathan-skantz/calendar/backend/internal/recurrence"
)

// MaxEvents bounds the VEVENTs one file may hold, so one import cannot hold
// a transaction open for long.
const MaxEvents = 5000

var (
	errTooManyExceptions = errors.New("too many changed or excluded occurrences")
	ErrInvalidFile       = errors.New("the file is not a valid iCalendar file")
	ErrTooManyEvents     = fmt.Errorf("a file can hold at most %d events", MaxEvents)
)

// Reasons an event in a file is not imported.
const (
	ReasonUnsupportedRule  = "unsupported repeat rule"
	ReasonOrphanOccurrence = "a changed occurrence of a repeating event that is not in the file"
	ReasonCancelled        = "cancelled"
	ReasonNoStart          = "no start time"
	ReasonBadTime          = "unreadable start or end time"
)

// MaxExceptions bounds the changed and excluded occurrences one series keeps,
// so a file cannot make a series costly to store and read.
const MaxExceptions = 1000

const warningTooManyExceptions = "only the first 1000 changed or excluded occurrences were kept"

// Properties for the fields only this app has, so an exported file imports
// back unchanged. Durations use the iCalendar format, such as PT30M.
const (
	propStartApproximate  = "X-JONSKA-START-APPROX"
	propEndApproximate    = "X-JONSKA-END-APPROX"
	propCommuteBefore     = "X-JONSKA-COMMUTE-BEFORE"
	propCommuteAfter      = "X-JONSKA-COMMUTE-AFTER"
	propGathering         = "X-JONSKA-GATHERING"
	propGatheringOptional = "X-JONSKA-GATHERING-OPTIONAL"
	propGatheringLabel    = "X-JONSKA-GATHERING-LABEL"
	propStartsAtGathering = "X-JONSKA-STARTS-AT-GATHERING"
	propSkipped           = "X-JONSKA-SKIPPED"
)

// defaultDuration is the length of a timed event that gives no end, or ends
// when it starts. Events must end after they start.
const defaultDuration = 30 * time.Minute

// Event is one VEVENT mapped to the app. It is not validated yet.
type Event struct {
	db.Event
	UID             string
	Categories      []string
	ReminderOffsets []int
	// A repeating event's rule, the IANA time zone it repeats in, and the
	// occurrences that differ from it: excluded dates and changed
	// occurrences.
	Recurrence *recurrence.Rule
	TimeZone   string
	Exceptions []db.OccurrenceException
	// lead is how much earlier than the event the file gave its start: by
	// its gathering, in files this app wrote. Its changed occurrences are
	// earlier by as much.
	lead time.Duration
}

// Skipped is an event the file holds that is not imported, and why.
type Skipped struct {
	Title  string `json:"title"`
	Reason string `json:"reason"`
}

type Result struct {
	// Name is the calendar's own name (NAME or X-WR-CALNAME), if it has one.
	Name     string
	Events   []Event
	Skipped  []Skipped
	Warnings []string
}

// Parse reads the VEVENTs of an iCalendar file. Floating times and all-day
// dates are read in loc, the user's time zone. Other components are ignored,
// and malformed lines are skipped.
func Parse(r io.Reader, loc *time.Location) (Result, error) {
	calendar, err := ics.ParseCalendarWithOptions(r,
		ics.WithPropertyParser(ics.SkipPropertyParser),
		ics.WithUnknownPropertyHandler(ics.AcceptUnknownPropertyHandler),
	)
	if err != nil {
		return Result{}, fmt.Errorf("%w: %v", ErrInvalidFile, err)
	}
	// Malformed lines are skipped, so a file with no calendar in it parses
	// as an empty one. A real calendar has at least VERSION and PRODID.
	if len(calendar.CalendarProperties) == 0 && len(calendar.Components) == 0 {
		return Result{}, ErrInvalidFile
	}
	vevents := calendar.Events()
	if len(vevents) > MaxEvents {
		return Result{}, ErrTooManyEvents
	}
	var result Result
	for _, property := range calendar.CalendarProperties {
		name := strings.TrimSpace(property.Value)
		switch ics.Property(property.IANAToken) {
		case ics.PropertyName:
			result.Name = name
		case ics.PropertyXWRCalName:
			if result.Name == "" {
				result.Name = name
			}
		}
	}
	warn := func(title string, warnings []string) {
		for _, warning := range warnings {
			result.Warnings = append(result.Warnings, title+": "+warning)
		}
	}
	// Changed occurrences of a repeating event share its UID and carry a
	// RECURRENCE-ID; they become exceptions of the series once it is read.
	var overrides []*ics.VEvent
	series := make(map[string]int)
	for _, vevent := range vevents {
		if vevent.HasProperty(ics.ComponentPropertyRecurrenceId) {
			overrides = append(overrides, vevent)
			continue
		}
		event, warnings, reason := parseEvent(vevent, loc)
		warn(event.Title, warnings)
		if reason != "" {
			result.Skipped = append(result.Skipped, Skipped{Title: event.Title, Reason: reason})
			continue
		}
		if event.Recurrence != nil && event.UID != "" {
			series[event.UID] = len(result.Events)
		}
		result.Events = append(result.Events, event)
	}
	// Series that reached MaxExceptions, warned about once each.
	full := map[int]bool{}
	for _, override := range overrides {
		title := titleOf(override)
		index, ok := -1, false
		if uid := textValue(override, ics.ComponentPropertyUniqueId); uid != nil {
			index, ok = series[*uid]
		}
		if !ok {
			result.Skipped = append(result.Skipped, Skipped{Title: title, Reason: ReasonOrphanOccurrence})
			continue
		}
		master := &result.Events[index]
		exception, warnings, err := parseOverride(override, *master, loc)
		warn(title, warnings)
		if err != nil {
			result.Skipped = append(result.Skipped, Skipped{Title: title, Reason: ReasonBadTime})
			continue
		}
		var kept bool
		if master.Exceptions, kept = setException(master.Exceptions, exception); !kept && !full[index] {
			full[index] = true
			warn(master.Title, []string{warningTooManyExceptions})
		}
	}
	return result, nil
}

func titleOf(vevent *ics.VEvent) string {
	if title := truncated(textValue(vevent, ics.ComponentPropertySummary), db.MaxTitleLength); title != nil {
		return *title
	}
	return "Untitled"
}

// truncated cuts value to at most limit characters.
func truncated(value *string, limit int) *string {
	if value == nil {
		return nil
	}
	if runes := []rune(*value); len(runes) > limit {
		cut := strings.TrimSpace(string(runes[:limit]))
		return &cut
	}
	return value
}

// setException adds an exception, replacing one for the same occurrence. It
// reports false, leaving exceptions unchanged, when the series already holds
// MaxExceptions.
func setException(exceptions []db.OccurrenceException, exception db.OccurrenceException) ([]db.OccurrenceException, bool) {
	for index, other := range exceptions {
		if other.OriginalStart.Equal(exception.OriginalStart) {
			exceptions[index] = exception
			return exceptions, true
		}
	}
	if len(exceptions) >= MaxExceptions {
		return exceptions, false
	}
	return append(exceptions, exception), true
}

// parseEvent maps one VEVENT. A non-empty reason means it is skipped; the
// event's title is set either way.
func parseEvent(vevent *ics.VEvent, loc *time.Location) (event Event, warnings []string, reason string) {
	event.Title = titleOf(vevent)
	if hasAny(vevent, ics.ComponentPropertyRdate, ics.ComponentPropertyExrule) {
		return event, nil, ReasonUnsupportedRule
	}
	if status := textValue(vevent, ics.ComponentPropertyStatus); status != nil &&
		strings.EqualFold(*status, string(ics.ObjectStatusCancelled)) {
		return event, nil, ReasonCancelled
	}

	startProperty := vevent.GetProperty(ics.ComponentPropertyDtStart)
	if startProperty == nil {
		return event, nil, ReasonNoStart
	}
	start, err := readTime(startProperty, loc)
	if err != nil {
		return event, nil, ReasonBadTime
	}
	if start.warning != "" {
		warnings = append(warnings, start.warning)
	}
	event.StartsAt = start.at
	event.IsAllDay = start.isDate
	switch {
	case vevent.HasProperty(ics.ComponentPropertyDtEnd):
		end, err := readTime(vevent.GetProperty(ics.ComponentPropertyDtEnd), loc)
		if err != nil || end.isDate != start.isDate {
			return event, warnings, ReasonBadTime
		}
		event.EndsAt = end.at
	case vevent.HasProperty(ics.ComponentPropertyDuration):
		length, err := parseDuration(vevent.GetProperty(ics.ComponentPropertyDuration).Value)
		if err != nil {
			return event, warnings, ReasonBadTime
		}
		event.EndsAt = length.addTo(event.StartsAt)
	case event.IsAllDay:
		event.EndsAt = event.StartsAt.AddDate(0, 0, 1)
	}
	if !event.IsAllDay && (event.EndsAt.IsZero() || event.EndsAt.Equal(event.StartsAt)) {
		event.EndsAt = event.StartsAt.Add(defaultDuration)
	}
	if property := vevent.GetProperty(ics.ComponentPropertyRrule); property != nil {
		// A series repeats in its start's zone; times given in UTC repeat
		// in the user's.
		zone := start.zone
		if zone == nil {
			zone = loc
		}
		rule, err := recurrence.FromRRULE(property.Value, event.StartsAt.In(zone))
		if err != nil {
			return event, warnings, ReasonUnsupportedRule
		}
		event.Recurrence, event.TimeZone = &rule, zone.String()
		excluded, err := excludedDates(vevent, loc)
		switch {
		case errors.Is(err, errTooManyExceptions):
			warnings = append(warnings, warningTooManyExceptions)
		case err != nil:
			warnings = append(warnings, "excluded dates that could not be read were left out")
		}
		event.Exceptions = excluded
	}

	// Text past the app's limits is cut rather than skipping the event; a
	// URL cannot be cut, so a long one is left out.
	event.Description = truncated(textValue(vevent, ics.ComponentPropertyDescription), db.MaxDescriptionLength)
	if event.URL = textValue(vevent, ics.ComponentPropertyUrl); event.URL != nil && len(*event.URL) > db.MaxURLLength {
		event.URL = nil
	}
	event.Location = truncated(textValue(vevent, ics.ComponentPropertyLocation), db.MaxLocationLength)
	if uid := textValue(vevent, ics.ComponentPropertyUniqueId); uid != nil {
		event.UID = *uid
	}
	event.Kind = db.EventKindAttendable
	if transparency := textValue(vevent, ics.ComponentPropertyTransp); transparency != nil &&
		strings.EqualFold(*transparency, string(ics.TransparencyTransparent)) {
		event.Kind = db.EventKindInformational
	}
	for _, property := range vevent.GetProperties(ics.ComponentPropertyCategories) {
		for name := range strings.SplitSeq(property.Value, ",") {
			if name = strings.TrimSpace(name); name != "" {
				event.Categories = append(event.Categories, name)
			}
		}
	}
	readAppFields(vevent, &event.Event)
	dtStart := event.StartsAt
	if value := textValue(vevent, ics.ComponentProperty(propStartsAtGathering)); value != nil &&
		strings.EqualFold(*value, "TRUE") && !event.IsAllDay {
		event.lead = time.Duration(event.GatheringSeconds) * time.Second
		event.StartsAt = event.StartsAt.Add(event.lead)
		for index := range event.Exceptions {
			event.Exceptions[index].OriginalStart = event.Exceptions[index].OriginalStart.Add(event.lead)
		}
	}

	for _, alarm := range vevent.Alarms() {
		offset, ok := reminderOffset(alarm, event.Event, dtStart, loc)
		if !ok {
			warnings = append(warnings, "a reminder that is not 0 to 30 days before the start was left out")
			continue
		}
		event.ReminderOffsets = append(event.ReminderOffsets, offset)
	}
	return event, warnings, ""
}

// readAppFields reads the X-JONSKA properties. Values that cannot be read are
// ignored; event validation rejects combinations the app does not allow.
func readAppFields(vevent *ics.VEvent, event *db.Event) {
	flag := func(name string) bool {
		value := textValue(vevent, ics.ComponentProperty(name))
		return value != nil && strings.EqualFold(*value, "TRUE")
	}
	seconds := func(name string) int {
		value := textValue(vevent, ics.ComponentProperty(name))
		if value == nil {
			return 0
		}
		length, err := parseDuration(*value)
		if err != nil || length.negative {
			return 0
		}
		return int(length.nominal().Seconds())
	}
	event.StartIsApproximate = flag(propStartApproximate)
	event.EndIsApproximate = flag(propEndApproximate)
	event.CommuteBeforeSeconds = seconds(propCommuteBefore)
	event.CommuteAfterSeconds = seconds(propCommuteAfter)
	event.GatheringSeconds = seconds(propGathering)
	event.GatheringIsOptional = flag(propGatheringOptional)
	event.GatheringLabel = textValue(vevent, ics.ComponentProperty(propGatheringLabel))
	event.IsSkipped = flag(propSkipped)
}

// reminderOffset turns a VALARM trigger into seconds before the event
// starts, in whole minutes. A trigger is relative to dtStart, the file's
// start, which may be the gathering's. Reminders after the start, or more
// than 30 days before it, are not supported.
func reminderOffset(alarm *ics.VAlarm, event db.Event, dtStart time.Time, loc *time.Location) (int, bool) {
	trigger := alarm.GetProperty(ics.ComponentPropertyTrigger)
	if trigger == nil {
		return 0, false
	}
	var remindAt time.Time
	if parameter(trigger, ics.ParameterValue) == string(ics.ValueDataTypeDateTime) {
		at, err := readTime(trigger, loc)
		if err != nil {
			return 0, false
		}
		remindAt = at.at
	} else {
		length, err := parseDuration(trigger.Value)
		if err != nil {
			return 0, false
		}
		base := dtStart
		if strings.EqualFold(parameter(trigger, ics.ParameterRelated), "END") {
			base = event.EndsAt
		}
		remindAt = length.addTo(base)
	}
	offset := event.StartsAt.Sub(remindAt)
	if offset < 0 || offset > db.MaxReminderOffsetSeconds*time.Second {
		return 0, false
	}
	return int(offset/time.Minute) * 60, true
}

type icalTime struct {
	at     time.Time
	isDate bool
	// zone is the zone the value was read in: its TZID's, or loc for dates
	// and floating times. It is nil for UTC times.
	zone    *time.Location
	warning string
}

const (
	dateLayout     = "20060102"
	dateTimeLayout = "20060102T150405"
)

// readTime reads a DATE or DATE-TIME value. Dates and floating times are in
// loc. A TZID that is neither an IANA nor a Windows zone name is read as loc,
// with a warning.
func readTime(property *ics.IANAProperty, loc *time.Location) (icalTime, error) {
	value := strings.TrimSpace(property.Value)
	if len(value) == len(dateLayout) {
		day, err := time.Parse(dateLayout, value)
		if err != nil {
			return icalTime{}, err
		}
		return icalTime{at: time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, loc), isDate: true, zone: loc}, nil
	}
	if utc, ok := strings.CutSuffix(value, "Z"); ok {
		at, err := time.Parse(dateTimeLayout, utc)
		return icalTime{at: at}, err
	}
	zone, warning := loc, ""
	if tzid := parameter(property, ics.ParameterTzid); tzid != "" {
		if named, ok := lookupZone(tzid); ok {
			zone = named
		} else {
			warning = fmt.Sprintf("unknown time zone %q was read as %s", tzid, loc)
		}
	}
	at, err := time.ParseInLocation(dateTimeLayout, value, zone)
	return icalTime{at: at, zone: zone, warning: warning}, err
}

// excludedDates reads a series' EXDATE values as cancelled occurrences.
func excludedDates(vevent *ics.VEvent, loc *time.Location) ([]db.OccurrenceException, error) {
	var exceptions []db.OccurrenceException
	for _, property := range vevent.GetProperties(ics.ComponentPropertyExdate) {
		for value := range strings.SplitSeq(property.Value, ",") {
			single := *property
			single.Value = value
			excluded, err := readTime(&single, loc)
			if err != nil {
				return exceptions, err
			}
			var kept bool
			exceptions, kept = setException(exceptions, db.OccurrenceException{OriginalStart: excluded.at, IsCancelled: true})
			if !kept {
				return exceptions, errTooManyExceptions
			}
		}
	}
	return exceptions, nil
}

// parseOverride reads a changed occurrence of master. Only its time, whether
// it happens, and the app's own skip carry over; other changes are reported.
func parseOverride(vevent *ics.VEvent, master Event, loc *time.Location) (db.OccurrenceException, []string, error) {
	original, err := readTime(vevent.GetProperty(ics.ComponentPropertyRecurrenceId), loc)
	if err != nil {
		return db.OccurrenceException{}, nil, err
	}
	original.at = original.at.Add(master.lead)
	exception := db.OccurrenceException{OriginalStart: original.at}
	if status := textValue(vevent, ics.ComponentPropertyStatus); status != nil &&
		strings.EqualFold(*status, string(ics.ObjectStatusCancelled)) {
		exception.IsCancelled = true
		return exception, nil, nil
	}
	var warnings []string
	if titleOf(vevent) != master.Title ||
		!sameText(textValue(vevent, ics.ComponentPropertyLocation), master.Location) ||
		!sameText(textValue(vevent, ics.ComponentPropertyDescription), master.Description) {
		warnings = append(warnings, "changes to one occurrence other than its time were left out")
	}
	if value := textValue(vevent, ics.ComponentProperty(propSkipped)); value != nil {
		skipped := strings.EqualFold(*value, "TRUE")
		exception.IsSkipped = &skipped
	}

	startsAt, endsAt := original.at, original.at.Add(master.EndsAt.Sub(master.StartsAt))
	if master.IsAllDay {
		endsAt = original.at.AddDate(0, 0, recurrence.DaysBetween(master.StartsAt.In(loc), master.EndsAt.In(loc)))
	}
	if property := vevent.GetProperty(ics.ComponentPropertyDtStart); property != nil {
		start, err := readTime(property, loc)
		if err != nil {
			return exception, warnings, err
		}
		length := endsAt.Sub(startsAt)
		startsAt, endsAt = start.at.Add(master.lead), start.at.Add(master.lead+length)
		switch {
		case vevent.HasProperty(ics.ComponentPropertyDtEnd):
			end, err := readTime(vevent.GetProperty(ics.ComponentPropertyDtEnd), loc)
			if err != nil {
				return exception, warnings, err
			}
			endsAt = end.at
		case vevent.HasProperty(ics.ComponentPropertyDuration):
			length, err := parseDuration(vevent.GetProperty(ics.ComponentPropertyDuration).Value)
			if err != nil {
				return exception, warnings, err
			}
			endsAt = length.addTo(startsAt)
		}
	}
	if !endsAt.After(startsAt) {
		return exception, warnings, errInvalidDuration
	}
	if !startsAt.Equal(original.at) || !endsAt.Equal(original.at.Add(master.EndsAt.Sub(master.StartsAt))) {
		exception.StartsAt, exception.EndsAt = &startsAt, &endsAt
	}
	return exception, warnings, nil
}

func sameText(a, b *string) bool {
	return (a == nil && b == nil) || (a != nil && b != nil && *a == *b)
}

func lookupZone(tzid string) (*time.Location, bool) {
	tzid = strings.Trim(tzid, `"`)
	// "Local" and "" are valid for LoadLocation but mean the server's zone
	// and UTC, not what the file says.
	if tzid == "" || tzid == "Local" {
		return nil, false
	}
	if zone, err := time.LoadLocation(tzid); err == nil {
		return zone, true
	}
	if zone := ics.WindowsTimezoneToIANA(tzid); zone != nil {
		return zone, true
	}
	return nil, false
}

// textValue returns a property's value trimmed, or nil when it is missing or
// blank.
func textValue(vevent *ics.VEvent, name ics.ComponentProperty) *string {
	property := vevent.GetProperty(name)
	if property == nil {
		return nil
	}
	value := strings.TrimSpace(property.Value)
	if value == "" {
		return nil
	}
	return &value
}

func hasAny(vevent *ics.VEvent, names ...ics.ComponentProperty) bool {
	for _, name := range names {
		if vevent.HasProperty(name) {
			return true
		}
	}
	return false
}

func parameter(property *ics.IANAProperty, name ics.Parameter) string {
	values := property.ICalParameters[string(name)]
	if len(values) == 0 {
		return ""
	}
	return values[0]
}
