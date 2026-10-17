package timeline

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/jonathan-skantz/calendar/backend/db"
)

var (
	ErrInvalidRequest = errors.New("invalid request body")
	ErrTitleRequired  = errors.New("title cannot be empty")
	ErrInvalidTimes   = errors.New("end time must be after start time")
	ErrInvalidKind    = errors.New("kind must be attendable or informational")
	ErrInvalidCommute = errors.New("commute must be whole minutes up to 24 hours and is only allowed on timed attendable events")
	ErrDeadlineNeeded = errors.New("deadline is required")

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
	maxCommuteSeconds       = 24 * 60 * 60
	maxTransitCardCount     = 20
)

func validCommuteSeconds(seconds int) bool {
	return seconds >= 0 && seconds <= maxCommuteSeconds && seconds%60 == 0
}

func validateEventRequest(request eventRequest) error {
	if request.Title == "" {
		return ErrTitleRequired
	}
	if request.Kind != "attendable" && request.Kind != "informational" {
		return ErrInvalidKind
	}
	if !request.EndsAt.After(request.StartsAt) {
		return ErrInvalidTimes
	}
	if !validCommuteSeconds(request.CommuteBeforeSeconds) || !validCommuteSeconds(request.CommuteAfterSeconds) {
		return ErrInvalidCommute
	}
	hasCommute := request.CommuteBeforeSeconds > 0 || request.CommuteAfterSeconds > 0
	if hasCommute && (request.Kind != "attendable" || request.IsAllDay) {
		return ErrInvalidCommute
	}
	return nil
}

func validateTaskRequest(request taskRequest) error {
	if request.Title == "" {
		return ErrTitleRequired
	}
	if request.DeadlineAt.IsZero() {
		return ErrDeadlineNeeded
	}
	return nil
}

func validateTagName(name string) error {
	if strings.TrimSpace(name) == "" {
		return ErrInvalidRequest
	}
	return nil
}

func validateTagUpdateRequest(request tagUpdateRequest) error {
	if err := validateTagName(request.Name); err != nil {
		return err
	}
	if !validTagColor(request.Color) {
		return ErrInvalidRequest
	}
	return nil
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

func validateHouseholdRequest(request *householdRequest) error {
	request.Name = strings.TrimSpace(request.Name)
	if request.Name == "" {
		return ErrHouseholdNameRequired
	}
	if request.TransitCardCount < 1 || request.TransitCardCount > maxTransitCardCount {
		return ErrInvalidTransitCardCount
	}
	return nil
}

// normalizeShareField trims a per-recipient value; blank means "same as event".
func normalizeShareField(value *string) *string {
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
		Title:       normalizeShareField(request.Title),
		Description: normalizeShareField(request.Description),
		URL:         normalizeShareField(request.URL),
		Location:    normalizeShareField(request.Location),
	}
}
