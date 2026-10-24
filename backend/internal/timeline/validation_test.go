package timeline

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jonathan-skantz/calendar/backend/db"
)

func TestValidateTaskRequest(t *testing.T) {
	tests := []struct {
		name    string
		request taskRequest
		wantErr error
	}{
		{name: "missing title", request: taskRequest{DeadlineAt: time.Now()}, wantErr: ErrTitleRequired},
		{name: "missing deadline", request: taskRequest{Title: "Task"}, wantErr: ErrDeadlineNeeded},
		{name: "valid", request: taskRequest{Title: "Task", DeadlineAt: time.Now()}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := validateTaskRequest(test.request)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("expected error %v, got %v", test.wantErr, err)
			}
		})
	}
}

func TestValidateTagRequests(t *testing.T) {
	if _, err := validateTagName("  "); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("expected invalid request, got %v", err)
	}
	if _, err := validateTagUpdateRequest(tagUpdateRequest{Name: "Work", Color: "#1a2B3c"}); err != nil {
		t.Fatalf("expected valid tag update, got %v", err)
	}
	if _, err := validateTagUpdateRequest(tagUpdateRequest{Name: "Work", Color: "red"}); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("expected invalid color error, got %v", err)
	}
	if err := validateTagSelectionRequest(tagSelectionRequest{}); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("expected missing tag error, got %v", err)
	}
	if err := validateTagSelectionRequest(tagSelectionRequest{TagID: uuid.New()}); err != nil {
		t.Fatalf("expected valid tag selection, got %v", err)
	}
}

func TestValidatorsReturnNormalizedRequests(t *testing.T) {
	label := "  Warm-up  "
	event, err := ValidateEvent(db.Event{
		Title:            "Football",
		Kind:             db.EventKindAttendable,
		StartsAt:         time.Date(2026, 10, 15, 18, 0, 0, 0, time.UTC),
		EndsAt:           time.Date(2026, 10, 15, 19, 0, 0, 0, time.UTC),
		GatheringSeconds: 900,
		GatheringLabel:   &label,
	})
	if err != nil || event.GatheringLabel == nil || *event.GatheringLabel != "Warm-up" {
		t.Fatalf("expected a trimmed gathering label, got %v (err %v)", event.GatheringLabel, err)
	}
	if label != "  Warm-up  " {
		t.Fatal("the caller's request must not be modified")
	}

	if name, err := validateTagName("  Work "); err != nil || name != "Work" {
		t.Fatalf("expected trimmed tag name, got %q (err %v)", name, err)
	}
	household, err := validateHouseholdRequest(householdRequest{Name: " Home ", TransitCardCount: 1})
	if err != nil || household.Name != "Home" {
		t.Fatalf("expected trimmed household name, got %q (err %v)", household.Name, err)
	}
}

func TestTextLengthsAreLimited(t *testing.T) {
	long := func(length int) *string {
		value := strings.Repeat("x", length)
		return &value
	}
	event := db.Event{
		Title:    *long(db.MaxTitleLength),
		Kind:     db.EventKindInformational,
		StartsAt: time.Date(2026, 10, 15, 18, 0, 0, 0, time.UTC),
		EndsAt:   time.Date(2026, 10, 15, 19, 0, 0, 0, time.UTC),
		Location: long(db.MaxLocationLength),
	}
	if _, err := ValidateEvent(event); err != nil {
		t.Fatalf("text at the limits: %v", err)
	}
	for name, change := range map[string]func(*db.Event){
		"title":       func(event *db.Event) { event.Title = *long(db.MaxTitleLength + 1) },
		"description": func(event *db.Event) { event.Description = long(db.MaxDescriptionLength + 1) },
		"url":         func(event *db.Event) { event.URL = long(db.MaxURLLength + 1) },
		"location":    func(event *db.Event) { event.Location = long(db.MaxLocationLength + 1) },
	} {
		changed := event
		change(&changed)
		if _, err := ValidateEvent(changed); !errors.Is(err, ErrTextTooLong) {
			t.Errorf("%s: expected %v, got %v", name, ErrTextTooLong, err)
		}
	}
	if _, err := normalizeShareFields(shareFieldsRequest{Description: long(db.MaxDescriptionLength + 1)}); !errors.Is(err, ErrTextTooLong) {
		t.Errorf("share fields: expected %v, got %v", ErrTextTooLong, err)
	}
	if _, err := validateCustomizationRequest(customizationRequest{Note: long(db.MaxDescriptionLength + 1)}); !errors.Is(err, ErrTextTooLong) {
		t.Errorf("customization note: expected %v, got %v", ErrTextTooLong, err)
	}
	if _, err := validateTagName(*long(db.MaxTagNameLength + 1)); !errors.Is(err, ErrTagNameTooLong) {
		t.Errorf("tag name: expected %v, got %v", ErrTagNameTooLong, err)
	}
}

func TestTitlesAreTrimmedAndMustNotBeBlank(t *testing.T) {
	event := db.Event{
		Title:    "   ",
		Kind:     db.EventKindInformational,
		StartsAt: time.Date(2026, 10, 15, 18, 0, 0, 0, time.UTC),
		EndsAt:   time.Date(2026, 10, 15, 19, 0, 0, 0, time.UTC),
	}
	if _, err := ValidateEvent(event); !errors.Is(err, ErrTitleRequired) {
		t.Fatalf("blank event title: expected %v, got %v", ErrTitleRequired, err)
	}
	event.Title = "  Lecture "
	if validated, err := ValidateEvent(event); err != nil || validated.Title != "Lecture" {
		t.Fatalf("expected trimmed event title, got %q (err %v)", validated.Title, err)
	}

	task := taskRequest{Title: " \t", DeadlineAt: time.Date(2026, 10, 15, 18, 0, 0, 0, time.UTC)}
	if _, err := validateTaskRequest(task); !errors.Is(err, ErrTitleRequired) {
		t.Fatalf("blank task title: expected %v, got %v", ErrTitleRequired, err)
	}
	task.Title = " Follow up "
	if validated, err := validateTaskRequest(task); err != nil || validated.Title != "Follow up" {
		t.Fatalf("expected trimmed task title, got %q (err %v)", validated.Title, err)
	}
}
