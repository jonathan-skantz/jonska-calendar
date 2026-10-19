package assistant

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jonathan-skantz/calendar/backend/db"
	"github.com/jonathan-skantz/calendar/backend/internal/quickadd"
)

const (
	maxProposals      = 3
	maxTitleLength    = 200
	maxLocationLength = 200
	maxReminders      = 10
)

const proposeEventDescription = "Propose one new event for the user to review. This does not save anything: the user opens the proposal in the event form and decides. " +
	"Use it once you know the date and time; tags must be the user's existing tag names."

type proposeEventInput struct {
	Title           string   `json:"title" jsonschema:"required" jsonschema_description:"Short title in the user's language, without the date or time"`
	Date            string   `json:"date" jsonschema:"required" jsonschema_description:"YYYY-MM-DD"`
	StartTime       string   `json:"start_time,omitempty" jsonschema_description:"HH:MM, 24-hour; leave out for all-day events"`
	EndTime         string   `json:"end_time,omitempty" jsonschema_description:"HH:MM, 24-hour, after start_time; leave out for one hour"`
	AllDay          bool     `json:"all_day,omitempty"`
	Location        string   `json:"location,omitempty"`
	Kind            string   `json:"kind,omitempty" jsonschema:"enum=attendable,enum=informational" jsonschema_description:"attendable if the user takes part, informational for a heads-up"`
	Tags            []string `json:"tags,omitempty" jsonschema_description:"Existing tag names only"`
	ReminderMinutes []int    `json:"reminder_minutes,omitempty" jsonschema_description:"Minutes before the start"`
}

// Proposal is an event the agent suggests. Nothing is saved: the browser
// opens it in the event form, where the user reviews and saves it.
type Proposal struct {
	Title           string      `json:"title"`
	StartsAt        time.Time   `json:"starts_at"`
	EndsAt          time.Time   `json:"ends_at"`
	IsAllDay        bool        `json:"is_all_day"`
	Location        string      `json:"location,omitempty"`
	Kind            string      `json:"kind,omitempty"`
	TagIDs          []uuid.UUID `json:"tag_ids,omitempty"`
	ReminderMinutes []int       `json:"reminder_minutes,omitempty"`
	Warnings        []string    `json:"warnings,omitempty"`
}

func (t *turn) proposeEvent(_ context.Context, input proposeEventInput) (any, string, error) {
	if len(t.proposals) >= maxProposals {
		return nil, "", fmt.Errorf("at most %d proposals per message", maxProposals)
	}
	proposal, err := t.proposal(input)
	if err != nil {
		return nil, "", err
	}
	t.proposals = append(t.proposals, proposal)

	result := map[string]any{"proposed": true, "note": "The user will review it in the event form; nothing is saved yet."}
	if len(proposal.Warnings) > 0 {
		result["warnings"] = proposal.Warnings
	}
	return result, fmt.Sprintf("Proposed %q, %s", proposal.Title, t.when(proposal)), nil
}

// proposal validates input into a proposal, mapping tag names to the user's
// own tags.
func (t *turn) proposal(input proposeEventInput) (Proposal, error) {
	title := strings.TrimSpace(input.Title)
	if title == "" || utf8.RuneCountInString(title) > maxTitleLength {
		return Proposal{}, fmt.Errorf("title must be 1 to %d characters", maxTitleLength)
	}
	location := strings.TrimSpace(input.Location)
	if utf8.RuneCountInString(location) > maxLocationLength {
		return Proposal{}, fmt.Errorf("location must be at most %d characters", maxLocationLength)
	}
	if input.Kind != "" && input.Kind != db.EventKindAttendable && input.Kind != db.EventKindInformational {
		return Proposal{}, errors.New("kind must be attendable or informational")
	}
	start, end, err := t.proposalTimes(input)
	if err != nil {
		return Proposal{}, err
	}
	reminders, err := reminderMinutes(input.ReminderMinutes)
	if err != nil {
		return Proposal{}, err
	}

	tagIDs, unknown := quickadd.MatchTags(input.Tags, t.tags)
	var warnings []string
	for _, name := range unknown {
		warnings = append(warnings, fmt.Sprintf("No tag named %q", name))
	}
	return Proposal{
		Title: title, StartsAt: start, EndsAt: end, IsAllDay: input.AllDay,
		Location: location, Kind: input.Kind, TagIDs: tagIDs, ReminderMinutes: reminders, Warnings: warnings,
	}, nil
}

// proposalTimes is the start and end of the proposed event, which must not
// have started yet. An all-day event ends at midnight after its day; a timed
// one without an end lasts an hour.
func (t *turn) proposalTimes(input proposeEventInput) (time.Time, time.Time, error) {
	day, err := t.date(input.Date, "date")
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	if input.AllDay {
		switch {
		case input.StartTime != "" || input.EndTime != "":
			return time.Time{}, time.Time{}, errors.New("all-day events have no start_time or end_time")
		case day.Before(t.today()):
			return time.Time{}, time.Time{}, errors.New("date has already passed")
		}
		return day, day.AddDate(0, 0, 1), nil
	}

	if input.StartTime == "" {
		return time.Time{}, time.Time{}, errors.New("start_time is required unless all_day is true")
	}
	start, err := t.at(day, input.StartTime, "start_time")
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	end := start.Add(time.Hour)
	if input.EndTime != "" {
		if end, err = t.at(day, input.EndTime, "end_time"); err != nil {
			return time.Time{}, time.Time{}, err
		}
	}
	switch {
	case !end.After(start):
		return time.Time{}, time.Time{}, errors.New("end_time must be after start_time on the same date")
	case start.Before(t.now):
		return time.Time{}, time.Time{}, errors.New("start_time has already passed")
	}
	return start, end, nil
}

// reminderMinutes checks the offsets and returns them sorted, without
// duplicates.
func reminderMinutes(minutes []int) ([]int, error) {
	for _, value := range minutes {
		if value < 0 || value*60 > db.MaxReminderOffsetSeconds {
			return nil, errors.New("reminder_minutes must be from 0 up to 30 days")
		}
	}
	unique := slices.Compact(slices.Sorted(slices.Values(minutes)))
	return unique[:min(len(unique), maxReminders)], nil
}

func (t *turn) when(proposal Proposal) string {
	if proposal.IsAllDay {
		return proposal.StartsAt.Format(dayFormat)
	}
	return t.format(proposal.StartsAt) + "–" + proposal.EndsAt.In(t.location).Format(clockLayout)
}
