package timeline

import (
	"errors"
	"strings"

	"github.com/google/uuid"
)

var (
	ErrInvalidRequest = errors.New("invalid request body")
	ErrTitleRequired  = errors.New("title cannot be empty")
	ErrInvalidTimes   = errors.New("end time must be after start time")
	ErrInvalidKind    = errors.New("kind must be attendable or informational")
	ErrInvalidCommute = errors.New("commute duration cannot be negative")
	ErrDeadlineNeeded = errors.New("deadline is required")
)

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
	if request.CommuteBeforeSeconds < 0 || request.CommuteAfterSeconds < 0 {
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
