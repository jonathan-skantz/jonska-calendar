package assistant

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/jonathan-skantz/calendar/backend/db"
)

const maxSlotMinutes = 12 * 60

const findFreeSlotsDescription = "Find free time on one date between two clock times, in the user's time zone. " +
	"Returns the free gaps at least duration_minutes long and what blocks the rest."

type findFreeSlotsInput struct {
	Date            string `json:"date" jsonschema:"required" jsonschema_description:"YYYY-MM-DD"`
	Earliest        string `json:"earliest" jsonschema:"required" jsonschema_description:"HH:MM, 24-hour"`
	Latest          string `json:"latest" jsonschema:"required" jsonschema_description:"HH:MM, 24-hour, after earliest"`
	DurationMinutes int    `json:"duration_minutes" jsonschema:"required,minimum=1,maximum=720" jsonschema_description:"Shortest useful gap"`
}

// interval is a stretch of time, with the title of what fills it when busy.
type interval struct {
	start, end time.Time
	title      string
}

type span struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Title string `json:"title,omitempty"`
}

func (t *turn) findFreeSlots(ctx context.Context, input findFreeSlotsInput) (any, string, error) {
	window, err := t.window(input)
	if err != nil {
		return nil, "", err
	}
	// Commute and gathering can block up to a day each around an event.
	entries, err := t.timeline(ctx, window.start.Add(-48*time.Hour), window.end.Add(48*time.Hour))
	if err != nil {
		return nil, "", err
	}

	busy := blocking(entries, window)
	free := freeGaps(window, busy, time.Duration(input.DurationMinutes)*time.Minute)

	summary := fmt.Sprintf("Checked %s %s–%s: ", window.start.Format(dayFormat), input.Earliest, input.Latest)
	if len(free) == 0 {
		summary += "no free time"
	} else {
		summary += fmt.Sprintf("%d free slots", len(free))
	}
	return map[string]any{"free": t.spans(free), "busy": t.spans(busy)}, summary, nil
}

// window validates the input and returns the time range it asks about.
func (t *turn) window(input findFreeSlotsInput) (interval, error) {
	day, err := t.date(input.Date, "date")
	if err != nil {
		return interval{}, err
	}
	start, err := t.at(day, input.Earliest, "earliest")
	if err != nil {
		return interval{}, err
	}
	end, err := t.at(day, input.Latest, "latest")
	if err != nil {
		return interval{}, err
	}
	if !end.After(start) {
		return interval{}, errors.New("latest must be after earliest")
	}
	if input.DurationMinutes < 1 || input.DurationMinutes > maxSlotMinutes {
		return interval{}, fmt.Errorf("duration_minutes must be 1 to %d", maxSlotMinutes)
	}
	return interval{start: start, end: end}, nil
}

// blocking returns the busy spans that overlap window, by start time.
func blocking(entries []db.TimelineEntry, window interval) []interval {
	var busy []interval
	for _, entry := range entries {
		start, end, ok := occupiedSpan(entry)
		if ok && start.Before(window.end) && end.After(window.start) {
			busy = append(busy, interval{start, end, entry.Title})
		}
	}
	slices.SortFunc(busy, func(a, b interval) int { return a.start.Compare(b.start) })
	return busy
}

// freeGaps returns the parts of window that no busy span covers and that are
// at least minimum long. busy must be sorted by start and may overlap.
func freeGaps(window interval, busy []interval, minimum time.Duration) []interval {
	var free []interval
	cursor := window.start
	add := func(until time.Time) {
		if until.Sub(cursor) >= minimum {
			free = append(free, interval{start: cursor, end: until})
		}
	}
	for _, block := range busy {
		if block.start.After(cursor) {
			add(block.start)
		}
		if block.end.After(cursor) {
			cursor = block.end
		}
	}
	if window.end.After(cursor) {
		add(window.end)
	}
	return free
}

func (t *turn) spans(intervals []interval) []span {
	spans := make([]span, len(intervals))
	for i, value := range intervals {
		spans[i] = span{From: t.format(value.start), To: t.format(value.end), Title: value.title}
	}
	return spans
}
