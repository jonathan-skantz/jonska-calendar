package ical

import (
	"io"
	"time"

	ics "github.com/arran4/golang-ical"

	"github.com/jonathan-skantz/calendar/backend/db"
)

const productID = "-//jonska-calendar//EN"

// Encode writes events as an iCalendar file. Timed events use UTC, so the
// file needs no VTIMEZONE; all-day events are dates in loc, the user's time
// zone. stamp is the DTSTAMP, the time the file was made.
func Encode(w io.Writer, events []db.ExportEvent, loc *time.Location, stamp time.Time) error {
	calendar, err := ics.NewCalendarWithOptions(ics.WithVersion("2.0"), ics.WithProductId(productID))
	if err != nil {
		return err
	}
	for _, event := range events {
		vevent := calendar.AddEvent(event.UID)
		vevent.SetDtStampTime(stamp)
		vevent.SetSummary(event.Title)
		if event.IsAllDay {
			vevent.SetAllDayStartAt(event.StartsAt.In(loc))
			vevent.SetAllDayEndAt(event.EndsAt.In(loc))
		} else {
			vevent.SetStartAt(event.StartsAt)
			vevent.SetEndAt(event.EndsAt)
		}
		setOptional(vevent, ics.ComponentPropertyDescription, event.Description)
		setOptional(vevent, ics.ComponentPropertyLocation, event.Location)
		setOptional(vevent, ics.ComponentPropertyUrl, event.URL)
		if event.Kind == db.EventKindInformational {
			vevent.SetTimeTransparency(ics.TransparencyTransparent)
		}
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
	}
	return calendar.SerializeTo(w, ics.WithNewLineWindows)
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
