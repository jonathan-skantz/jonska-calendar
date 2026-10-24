package demo

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/jonathan-skantz/calendar/backend/db"
	"github.com/jonathan-skantz/calendar/backend/internal/recurrence"
)

// seed fills a new demo account with a week that shows the main features:
// tags, a repeating event, commute, an all-day heads-up, and a task. Times
// are wall-clock times in now's location, all in now's week (Monday to
// Sunday), so the week the app opens on is full.
func seed(ctx context.Context, store *db.Store, userID uuid.UUID, now time.Time) error {
	work, err := store.CreateTag(ctx, userID, "work")
	if err != nil {
		return err
	}
	sport, err := store.CreateTag(ctx, userID, "sport")
	if err != nil {
		return err
	}

	// at is a time on a day of the week, 0 being Monday.
	at := func(weekday, hour, minute int) time.Time {
		year, month, day := now.Date()
		monday := day - (int(now.Weekday())+6)%7
		return time.Date(year, month, monday+weekday, hour, minute, 0, 0, now.Location())
	}
	text := func(value string) *string { return &value }
	standup := at(0, 9, 0)
	weekdays := recurrence.Rule{
		Frequency: recurrence.Weekly, Interval: 1, Weekdays: []string{"MO", "TU", "WE", "TH", "FR"},
	}.RRULE(standup, false)
	zone := now.Location().String()

	events := []db.Event{
		{
			Title: "Team standup", StartsAt: standup, EndsAt: standup.Add(15 * time.Minute),
			Kind: db.EventKindAttendable, MainTagID: work.ID,
			RecurrenceRule: &weekdays, RecurrenceTimeZone: &zone,
		},
		{
			Title: "Dentist", StartsAt: at(1, 10, 0), EndsAt: at(1, 10, 45),
			Kind: db.EventKindAttendable, Location: text("Folktandvården"),
			CommuteBeforeSeconds: 15 * 60, CommuteAfterSeconds: 15 * 60,
		},
		{
			Title: "Lunch with Anna", StartsAt: at(2, 12, 0), EndsAt: at(2, 13, 0),
			Kind: db.EventKindAttendable, Location: text("Café Linné"),
		},
		{
			Title: "Climbing", StartsAt: at(3, 18, 30), EndsAt: at(3, 20, 0),
			Kind: db.EventKindAttendable, MainTagID: sport.ID,
			CommuteBeforeSeconds: 20 * 60, CommuteAfterSeconds: 20 * 60,
		},
		{
			Title: "Grandma's birthday", StartsAt: at(5, 0, 0), EndsAt: at(6, 0, 0),
			IsAllDay: true, Kind: db.EventKindInformational,
			Description: text("Call her in the morning."),
		},
	}
	for _, event := range events {
		if _, err := store.CreateEvent(ctx, userID, event); err != nil {
			return err
		}
	}
	_, err = store.CreateTask(ctx, userID, db.Task{Title: "Book the summer cabin", DeadlineAt: at(4, 17, 0)})
	return err
}
