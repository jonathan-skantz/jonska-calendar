package quickadd

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jonathan-skantz/calendar/backend/db"
)

// The commute, gathering and reminder limits are the event limits from db;
// the others only keep a misbehaving reply small.
const (
	maxTitleLength          = 200
	maxLocationLength       = 200
	maxCommuteMinutes       = db.MaxCommuteSeconds / 60
	maxGatheringLabelLength = db.MaxGatheringLabelLength
	maxReminderMinutes      = db.MaxReminderOffsetSeconds / 60
	maxDurationMinutes      = 14 * 24 * 60
	maxRelativeDays         = 366
	maxReminders            = 10
	maxUnresolved           = 5
	maxUnresolvedLength     = 100
)

// DateSpec is a date as the text states it ("tomorrow", "next Friday",
// "Oct 24"). The browser turns it into a calendar date, because only it knows
// the user's today, time zone and week start.
type DateSpec struct {
	Kind       string `json:"kind"`
	Days       *int   `json:"days,omitempty"`
	Weekday    string `json:"weekday,omitempty"`
	WeekOffset *int   `json:"week_offset,omitempty"`
	Month      *int   `json:"month,omitempty"`
	Day        *int   `json:"day,omitempty"`
	Year       *int   `json:"year,omitempty"`
}

type RawReminder struct {
	MinutesBefore int `json:"minutes_before"`
}

// RawDraft is the model's tool input, untrusted until validateDraft has run.
type RawDraft struct {
	Title                *string       `json:"title"`
	Date                 *DateSpec     `json:"date"`
	EndDate              *DateSpec     `json:"end_date"`
	StartTime            *string       `json:"start_time"`
	EndTime              *string       `json:"end_time"`
	DurationMinutes      *int          `json:"duration_minutes"`
	AllDay               *bool         `json:"all_day"`
	StartIsApproximate   *bool         `json:"start_is_approximate"`
	EndIsApproximate     *bool         `json:"end_is_approximate"`
	Location             *string       `json:"location"`
	Kind                 *string       `json:"kind"`
	Tags                 []string      `json:"tags"`
	CommuteBeforeMinutes *int          `json:"commute_before_minutes"`
	CommuteAfterMinutes  *int          `json:"commute_after_minutes"`
	GatheringMinutes     *int          `json:"gathering_minutes"`
	GatheringIsOptional  *bool         `json:"gathering_is_optional"`
	GatheringLabel       *string       `json:"gathering_label"`
	Reminders            []RawReminder `json:"reminders"`
	Unresolved           []string      `json:"unresolved"`
}

// Draft holds only the fields the text stated. Omitted fields keep the event
// form's defaults.
type Draft struct {
	Title                *string     `json:"title,omitempty"`
	Date                 *DateSpec   `json:"date,omitempty"`
	EndDate              *DateSpec   `json:"end_date,omitempty"`
	StartTime            *string     `json:"start_time,omitempty"`
	EndTime              *string     `json:"end_time,omitempty"`
	DurationMinutes      *int        `json:"duration_minutes,omitempty"`
	AllDay               *bool       `json:"all_day,omitempty"`
	StartIsApproximate   *bool       `json:"start_is_approximate,omitempty"`
	EndIsApproximate     *bool       `json:"end_is_approximate,omitempty"`
	Location             *string     `json:"location,omitempty"`
	Kind                 *string     `json:"kind,omitempty"`
	TagIDs               []uuid.UUID `json:"tag_ids,omitempty"`
	CommuteBeforeMinutes *int        `json:"commute_before_minutes,omitempty"`
	CommuteAfterMinutes  *int        `json:"commute_after_minutes,omitempty"`
	GatheringMinutes     *int        `json:"gathering_minutes,omitempty"`
	GatheringIsOptional  *bool       `json:"gathering_is_optional,omitempty"`
	GatheringLabel       *string     `json:"gathering_label,omitempty"`
	ReminderMinutes      []int       `json:"reminder_minutes,omitempty"`
}

type ParseResult struct {
	Draft      Draft    `json:"draft"`
	Warnings   []string `json:"warnings"`
	Unresolved []string `json:"unresolved"`
}

var clockPattern = regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d$`)

var weekdays = map[string]bool{
	"monday": true, "tuesday": true, "wednesday": true, "thursday": true,
	"friday": true, "saturday": true, "sunday": true,
}

// validateDraft drops anything the event form could not hold and resolves tag
// names against the user's own tags. Tags are never created from a draft.
func validateDraft(raw RawDraft, tags []db.Tag) ParseResult {
	result := ParseResult{Warnings: []string{}, Unresolved: []string{}}
	warn := func(format string, args ...any) {
		result.Warnings = append(result.Warnings, fmt.Sprintf(format, args...))
	}
	draft := &result.Draft

	draft.Title = text(raw.Title, maxTitleLength)
	draft.Location = text(raw.Location, maxLocationLength)
	draft.GatheringLabel = text(raw.GatheringLabel, maxGatheringLabelLength)

	if raw.Date != nil {
		if validDate(*raw.Date) {
			draft.Date = raw.Date
		} else {
			warn("Couldn't read the date")
		}
	}
	if raw.EndDate != nil {
		if validDate(*raw.EndDate) {
			draft.EndDate = raw.EndDate
		} else {
			warn("Couldn't read the end date")
		}
	}
	draft.StartTime = clock(raw.StartTime, "start time", warn)
	draft.EndTime = clock(raw.EndTime, "end time", warn)
	draft.DurationMinutes = minutes(raw.DurationMinutes, 1, maxDurationMinutes, "duration", warn)

	draft.AllDay = raw.AllDay
	draft.StartIsApproximate = raw.StartIsApproximate
	draft.EndIsApproximate = raw.EndIsApproximate
	draft.GatheringIsOptional = raw.GatheringIsOptional
	if raw.Kind != nil {
		if *raw.Kind == db.EventKindAttendable || *raw.Kind == db.EventKindInformational {
			draft.Kind = raw.Kind
		} else {
			warn("Couldn't tell whether to attend or treat it as a heads-up")
		}
	}

	draft.CommuteBeforeMinutes = minutes(raw.CommuteBeforeMinutes, 0, maxCommuteMinutes, "commute before", warn)
	draft.CommuteAfterMinutes = minutes(raw.CommuteAfterMinutes, 0, maxCommuteMinutes, "commute after", warn)
	draft.GatheringMinutes = minutes(raw.GatheringMinutes, 0, maxCommuteMinutes, "gathering", warn)

	for _, reminder := range raw.Reminders {
		if len(draft.ReminderMinutes) == maxReminders {
			break
		}
		if reminder.MinutesBefore < 0 {
			warn("Left out a reminder after the start")
			continue
		}
		if reminder.MinutesBefore > maxReminderMinutes {
			warn("Left out a reminder more than 30 days before")
			continue
		}
		draft.ReminderMinutes = append(draft.ReminderMinutes, reminder.MinutesBefore)
	}

	byName := make(map[string]uuid.UUID, len(tags))
	for _, tag := range tags {
		byName[strings.ToLower(tag.Name)] = tag.ID
	}
	seen := map[uuid.UUID]bool{}
	for _, name := range raw.Tags {
		name = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(name), "#"))
		if name == "" {
			continue
		}
		id, ok := byName[strings.ToLower(name)]
		if !ok {
			warn("No tag named %q", name)
			continue
		}
		if !seen[id] {
			seen[id] = true
			draft.TagIDs = append(draft.TagIDs, id)
		}
	}

	for _, part := range raw.Unresolved {
		if part := text(&part, maxUnresolvedLength); part != nil && len(result.Unresolved) < maxUnresolved {
			result.Unresolved = append(result.Unresolved, *part)
		}
	}
	return result
}

// text trims value and cuts it to limit characters; blank means absent.
func text(value *string, limit int) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	if utf8.RuneCountInString(trimmed) > limit {
		trimmed = strings.TrimSpace(string([]rune(trimmed)[:limit]))
	}
	return &trimmed
}

func clock(value *string, label string, warn func(string, ...any)) *string {
	if value == nil {
		return nil
	}
	if !clockPattern.MatchString(*value) {
		warn("Couldn't read the %s", label)
		return nil
	}
	return value
}

// minutes drops values below low and clamps values above high, since an
// overlong commute is more likely a slip than a reason to lose the field.
func minutes(value *int, low, high int, label string, warn func(string, ...any)) *int {
	if value == nil {
		return nil
	}
	if *value < low {
		warn("Left out the %s", label)
		return nil
	}
	if *value > high {
		warn("Shortened the %s to %d minutes", label, high)
		clamped := high
		return &clamped
	}
	return value
}

func validDate(spec DateSpec) bool {
	inRange := func(value *int, low, high int) bool {
		return value != nil && *value >= low && *value <= high
	}
	switch spec.Kind {
	case "relative_days":
		return inRange(spec.Days, 0, maxRelativeDays)
	case "weekday":
		return weekdays[spec.Weekday] && (spec.WeekOffset == nil || inRange(spec.WeekOffset, 0, 1))
	case "calendar":
		return inRange(spec.Month, 1, 12) && inRange(spec.Day, 1, 31) &&
			(spec.Year == nil || inRange(spec.Year, 2000, 2100))
	}
	return false
}
