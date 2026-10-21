package timeline

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/jonathan-skantz/calendar/backend/db"
)

var (
	ErrInvalidRequest = errors.New("invalid request body")
	ErrTitleRequired  = errors.New("title cannot be empty")
	ErrInvalidTimes   = errors.New("end time must be after start time")
	ErrInvalidKind    = errors.New("kind must be attendable or informational")
	ErrInvalidSkip    = errors.New("only attendable events can be skipped")
	ErrInvalidCommute = errors.New("commute must be whole minutes up to 24 hours and is only allowed on timed attendable events")
	ErrDeadlineNeeded = errors.New("deadline is required")
	ErrInvalidOffset  = errors.New("reminder offset must be whole minutes, from 0 up to 30 days before the event")

	ErrInvalidApproximate = errors.New("approximate start or end is only allowed on timed events")
	ErrInvalidGathering   = errors.New("gathering must be whole minutes up to 24 hours on a timed attendable event, with a label of at most 40 characters")

	ErrInvalidDate             = errors.New("dates must use the YYYY-MM-DD format")
	ErrInvalidDateRange        = errors.New("date range must cover 1 to 62 days")
	ErrInvalidTransitCardDates = errors.New("needed and maybe dates must be unique and inside the range")
	ErrPastTransitCardDay      = errors.New("days that have ended cannot be changed")
	ErrHouseholdNameRequired   = errors.New("household name cannot be empty")
	ErrInvalidTransitCardCount = errors.New("a household must have between 1 and 20 transit cards")
)

const (
	dateLayout              = "2006-01-02"
	maxTransitCardRangeDays = 62
	maxTransitCardCount     = 20
)

func validCommuteSeconds(seconds int) bool {
	return seconds >= 0 && seconds <= db.MaxCommuteSeconds && seconds%60 == 0
}

// ValidateEvent returns the event with its title and gathering label trimmed;
// a blank label means no label. Every path that writes an event uses it: the
// event routes and the assistant.
func ValidateEvent(event db.Event) (db.Event, error) {
	event.Title = strings.TrimSpace(event.Title)
	event.GatheringLabel = normalizeOptionalText(event.GatheringLabel)
	if event.Title == "" {
		return event, ErrTitleRequired
	}
	if event.Kind != db.EventKindAttendable && event.Kind != db.EventKindInformational {
		return event, ErrInvalidKind
	}
	if event.IsSkipped && event.Kind != db.EventKindAttendable {
		return event, ErrInvalidSkip
	}
	if !event.EndsAt.After(event.StartsAt) {
		return event, ErrInvalidTimes
	}
	// Commute and gathering only make sense for timed events the user attends.
	timedAttendable := event.Kind == db.EventKindAttendable && !event.IsAllDay
	if !validCommuteSeconds(event.CommuteBeforeSeconds) || !validCommuteSeconds(event.CommuteAfterSeconds) {
		return event, ErrInvalidCommute
	}
	hasCommute := event.CommuteBeforeSeconds > 0 || event.CommuteAfterSeconds > 0
	if hasCommute && !timedAttendable {
		return event, ErrInvalidCommute
	}
	if !validCommuteSeconds(event.GatheringSeconds) || (event.GatheringSeconds > 0 && !timedAttendable) {
		return event, ErrInvalidGathering
	}
	hasGatheringDetails := event.GatheringIsOptional || event.GatheringLabel != nil
	if hasGatheringDetails && event.GatheringSeconds == 0 {
		return event, ErrInvalidGathering
	}
	if event.GatheringLabel != nil && utf8.RuneCountInString(*event.GatheringLabel) > db.MaxGatheringLabelLength {
		return event, ErrInvalidGathering
	}
	if event.IsAllDay && (event.StartIsApproximate || event.EndIsApproximate) {
		return event, ErrInvalidApproximate
	}
	return event, nil
}

func validateReminderRequest(request reminderRequest) error {
	if request.OffsetSeconds < 0 || request.OffsetSeconds > db.MaxReminderOffsetSeconds || request.OffsetSeconds%60 != 0 {
		return ErrInvalidOffset
	}
	return nil
}

// validateTaskRequest returns the request with its title trimmed.
func validateTaskRequest(request taskRequest) (taskRequest, error) {
	request.Title = strings.TrimSpace(request.Title)
	if request.Title == "" {
		return request, ErrTitleRequired
	}
	if request.DeadlineAt.IsZero() {
		return request, ErrDeadlineNeeded
	}
	return request, nil
}

// validateTagName returns the name trimmed.
func validateTagName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", ErrInvalidRequest
	}
	return name, nil
}

// validateTagUpdateRequest returns the request with its name trimmed.
func validateTagUpdateRequest(request tagUpdateRequest) (tagUpdateRequest, error) {
	name, err := validateTagName(request.Name)
	if err != nil {
		return request, err
	}
	request.Name = name
	if !validTagColor(request.Color) {
		return request, ErrInvalidRequest
	}
	return request, nil
}

func validateTagSelectionRequest(request tagSelectionRequest) error {
	if request.TagID == uuid.Nil {
		return ErrInvalidRequest
	}
	return nil
}

func validTagColor(color string) bool {
	if len(color) != 7 || color[0] != '#' {
		return false
	}
	for _, character := range color[1:] {
		if !((character >= '0' && character <= '9') || (character >= 'a' && character <= 'f') || (character >= 'A' && character <= 'F')) {
			return false
		}
	}
	return true
}

type dateRange struct {
	from time.Time
	to   time.Time
}

type transitCardSubmission struct {
	days   dateRange
	needed []time.Time
	maybe  []time.Time
}

func parseDateRange(from, to string) (dateRange, error) {
	fromDate, err := time.Parse(dateLayout, from)
	if err != nil {
		return dateRange{}, ErrInvalidDate
	}
	toDate, err := time.Parse(dateLayout, to)
	if err != nil {
		return dateRange{}, ErrInvalidDate
	}
	if toDate.Before(fromDate) || toDate.Sub(fromDate) >= maxTransitCardRangeDays*24*time.Hour {
		return dateRange{}, ErrInvalidDateRange
	}
	return dateRange{from: fromDate, to: toDate}, nil
}

// validateTransitCardDaysRequest takes today's UTC date as a parameter. The
// server does not know the user's time zone, so it accepts yesterday in UTC:
// that is still today somewhere, and the frontend enforces the exact local day.
func validateTransitCardDaysRequest(request transitCardDaysRequest, todayUTC time.Time) (transitCardSubmission, error) {
	days, err := parseDateRange(request.From, request.To)
	if err != nil {
		return transitCardSubmission{}, err
	}
	if days.from.Before(todayUTC.AddDate(0, 0, -1)) {
		return transitCardSubmission{}, ErrPastTransitCardDay
	}
	// seen spans both lists, so a day cannot be both needed and maybe.
	seen := map[time.Time]bool{}
	parse := func(values []string) ([]time.Time, error) {
		parsed := make([]time.Time, 0, len(values))
		for _, value := range values {
			day, err := time.Parse(dateLayout, value)
			if err != nil {
				return nil, ErrInvalidDate
			}
			if seen[day] || day.Before(days.from) || day.After(days.to) {
				return nil, ErrInvalidTransitCardDates
			}
			seen[day] = true
			parsed = append(parsed, day)
		}
		return parsed, nil
	}
	needed, err := parse(request.NeededDates)
	if err != nil {
		return transitCardSubmission{}, err
	}
	maybe, err := parse(request.MaybeDates)
	if err != nil {
		return transitCardSubmission{}, err
	}
	return transitCardSubmission{days: days, needed: needed, maybe: maybe}, nil
}

// validateHouseholdRequest returns the request with its name trimmed.
func validateHouseholdRequest(request householdRequest) (householdRequest, error) {
	request.Name = strings.TrimSpace(request.Name)
	if request.Name == "" {
		return request, ErrHouseholdNameRequired
	}
	if request.TransitCardCount < 1 || request.TransitCardCount > maxTransitCardCount {
		return request, ErrInvalidTransitCardCount
	}
	return request, nil
}

// normalizeOptionalText trims an optional text value; blank means "not set".
// For per-recipient share fields that means "same as the event".
func normalizeOptionalText(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func normalizeShareFields(request shareFieldsRequest) db.ShareFields {
	return db.ShareFields{
		Title:       normalizeOptionalText(request.Title),
		Description: normalizeOptionalText(request.Description),
		URL:         normalizeOptionalText(request.URL),
		Location:    normalizeOptionalText(request.Location),
	}
}
