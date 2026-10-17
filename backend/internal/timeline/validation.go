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
	ErrInvalidCommute = errors.New("commute must be whole minutes up to 24 hours and is only allowed on timed attendable events")
	ErrDeadlineNeeded = errors.New("deadline is required")

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
	maxCommuteSeconds       = 24 * 60 * 60
	maxTransitCardCount     = 20
	maxGatheringLabelLength = 40
)

func validCommuteSeconds(seconds int) bool {
	return seconds >= 0 && seconds <= maxCommuteSeconds && seconds%60 == 0
}

// validateEventRequest also trims the gathering label; blank means no label.
func validateEventRequest(request *eventRequest) error {
	request.GatheringLabel = normalizeOptionalText(request.GatheringLabel)
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
	if !validCommuteSeconds(request.GatheringSeconds) ||
		(request.GatheringSeconds > 0 && (request.Kind != "attendable" || request.IsAllDay)) {
		return ErrInvalidGathering
	}
	hasGatheringDetails := request.GatheringIsOptional || request.GatheringLabel != nil
	if hasGatheringDetails && request.GatheringSeconds == 0 {
		return ErrInvalidGathering
	}
	if request.GatheringLabel != nil && utf8.RuneCountInString(*request.GatheringLabel) > maxGatheringLabelLength {
		return ErrInvalidGathering
	}
	if request.IsAllDay && (request.StartIsApproximate || request.EndIsApproximate) {
		return ErrInvalidApproximate
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
