package ical

import (
	"io"
	"strconv"
	"strings"
	"time"

	ics "github.com/arran4/golang-ical"

	"github.com/jonathan-skantz/calendar/backend/db"
	"github.com/jonathan-skantz/calendar/backend/internal/recurrence"
)

const productID = "-//jonska-calendar//EN"

// Encode writes events as an iCalendar file. Plain timed events use UTC;
// all-day events are dates in loc, the user's time zone. Repeating events
// give their times in the zone they repeat in (TZID), so other apps keep
// their wall-clock times across daylight saving changes. stamp is the
// DTSTAMP, the time the file was made.
func Encode(w io.Writer, events []db.ExportEvent, loc *time.Location, stamp time.Time) error {
	calendar, err := encodeCalendar(events, loc, stamp)
	if err != nil {
		return err
	}
	return calendar.SerializeTo(w, ics.WithNewLineWindows)
}

// feedRefreshInterval is how often subscribing apps are asked to fetch a
// feed again. Many apps ignore it and use their own schedule.
const feedRefreshInterval = "PT1H"

// EncodeFeed writes events like Encode, as a feed named name.
func EncodeFeed(w io.Writer, name string, events []db.ExportEvent, loc *time.Location, stamp time.Time) error {
	calendar, err := encodeCalendar(events, loc, stamp)
	if err != nil {
		return err
	}
	calendar.SetName(name)
	calendar.SetXPublishedTTL(feedRefreshInterval)
	calendar.SetRefreshInterval(feedRefreshInterval)
	return calendar.SerializeTo(w, ics.WithNewLineWindows)
}

func encodeCalendar(events []db.ExportEvent, loc *time.Location, stamp time.Time) (*ics.Calendar, error) {
	calendar, err := ics.NewCalendarWithOptions(ics.WithVersion("2.0"), ics.WithProductId(productID))
	if err != nil {
		return nil, err
	}
	for _, event := range events {
		zone := loc
		if event.RecurrenceTimeZone != nil {
			if zone, err = time.LoadLocation(*event.RecurrenceTimeZone); err != nil {
				return nil, err
			}
		}
		vevent := calendar.AddEvent(event.UID)
		writeEvent(vevent, event, stamp)
		writeTimes(vevent, event.StartsAt, event.EndsAt, event.IsAllDay, zone, event.RecurrenceRule != nil)
		for _, name := range event.TagNames {
			vevent.AddCategory(name)
		}
		writeAppFields(vevent, event.Event)
		for _, offset := range event.ReminderOffsets {
			alarm := vevent.AddAlarm()
			alarm.SetAction(ics.ActionDisplay)
			alarm.SetProperty(ics.ComponentPropertyDescription, event.Title)
			alarm.SetTrigger(duration{negative: offset > 0, clock: time.Duration(offset) * time.Second}.String())
		}
		if event.RecurrenceRule == nil {
			continue
		}
		vevent.AddRrule(*event.RecurrenceRule)
		length := event.EndsAt.Sub(event.StartsAt)
		lengthDays := recurrence.DaysBetween(event.StartsAt.In(zone), event.EndsAt.In(zone))
		for _, exception := range event.Exceptions {
			if exception.IsCancelled {
				vevent.AddExdate(zonedValue(exception.OriginalStart, event.IsAllDay, zone), zonedParameter(event.IsAllDay, zone))
				continue
			}
			// A moved or skipped occurrence is its own VEVENT with the
			// series' UID and the start the rule gives it.
			override := calendar.AddEvent(event.UID)
			writeEvent(override, event, stamp)
			override.SetProperty(ics.ComponentPropertyRecurrenceId,
				zonedValue(exception.OriginalStart, event.IsAllDay, zone), zonedParameter(event.IsAllDay, zone))
			startsAt, endsAt := exception.OriginalStart, exception.OriginalStart.Add(length)
			if event.IsAllDay {
				endsAt = exception.OriginalStart.In(zone).AddDate(0, 0, lengthDays)
			}
			if exception.StartsAt != nil && exception.EndsAt != nil {
				startsAt, endsAt = *exception.StartsAt, *exception.EndsAt
			}
			writeTimes(override, startsAt, endsAt, event.IsAllDay, zone, true)
			if exception.IsSkipped != nil {
				override.SetProperty(ics.ComponentProperty(propSkipped), strings.ToUpper(strconv.FormatBool(*exception.IsSkipped)))
			}
		}
	}
	return calendar, nil
}

// writeEvent writes the fields an event and its changed occurrences share.
func writeEvent(vevent *ics.VEvent, event db.ExportEvent, stamp time.Time) {
	vevent.SetDtStampTime(stamp)
	vevent.SetSummary(event.Title)
	setOptional(vevent, ics.ComponentPropertyDescription, event.Description)
	setOptional(vevent, ics.ComponentPropertyLocation, event.Location)
	setOptional(vevent, ics.ComponentPropertyUrl, event.URL)
	if event.Kind == db.EventKindInformational {
		vevent.SetTimeTransparency(ics.TransparencyTransparent)
	}
}

// writeTimes writes DTSTART and DTEND: dates for all-day events, times in
// zone for repeating ones, and UTC otherwise.
func writeTimes(vevent *ics.VEvent, startsAt, endsAt time.Time, isAllDay bool, zone *time.Location, repeats bool) {
	if !isAllDay && !repeats {
		vevent.SetStartAt(startsAt)
		vevent.SetEndAt(endsAt)
		return
	}
	vevent.SetProperty(ics.ComponentPropertyDtStart, zonedValue(startsAt, isAllDay, zone), zonedParameter(isAllDay, zone))
	vevent.SetProperty(ics.ComponentPropertyDtEnd, zonedValue(endsAt, isAllDay, zone), zonedParameter(isAllDay, zone))
}

func zonedValue(at time.Time, isAllDay bool, zone *time.Location) string {
	if isAllDay {
		return at.In(zone).Format(dateLayout)
	}
	return at.In(zone).Format(dateTimeLayout)
}

func zonedParameter(isAllDay bool, zone *time.Location) ics.PropertyParameter {
	if isAllDay {
		return ics.WithValue(string(ics.ValueDataTypeDate))
	}
	return ics.WithTZID(zone.String())
}

// writeAppFields writes the fields only this app has, leaving out defaults,
// so other calendar apps see a plain event.
func writeAppFields(vevent *ics.VEvent, event db.Event) {
	flag := func(name string, set bool) {
		if set {
			vevent.SetProperty(ics.ComponentProperty(name), "TRUE")
		}
	}
	seconds := func(name string, value int) {
		if value > 0 {
			vevent.SetProperty(ics.ComponentProperty(name), duration{clock: time.Duration(value) * time.Second}.String())
		}
	}
	flag(propStartApproximate, event.StartIsApproximate)
	flag(propEndApproximate, event.EndIsApproximate)
	seconds(propCommuteBefore, event.CommuteBeforeSeconds)
	seconds(propCommuteAfter, event.CommuteAfterSeconds)
	seconds(propGathering, event.GatheringSeconds)
	flag(propGatheringOptional, event.GatheringIsOptional)
	setOptional(vevent, ics.ComponentProperty(propGatheringLabel), event.GatheringLabel)
	flag(propSkipped, event.IsSkipped)
}

func setOptional(vevent *ics.VEvent, name ics.ComponentProperty, value *string) {
	if value != nil {
		vevent.SetProperty(name, *value)
	}
}
