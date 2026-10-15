package timeline

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
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
			err := validateTaskRequest(test.request)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("expected error %v, got %v", test.wantErr, err)
			}
		})
	}
}

func TestValidateTagRequests(t *testing.T) {
	if err := validateTagName("  "); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("expected invalid request, got %v", err)
	}
	if err := validateTagUpdateRequest(tagUpdateRequest{Name: "Work", Color: "#1a2B3c"}); err != nil {
		t.Fatalf("expected valid tag update, got %v", err)
	}
	if err := validateTagUpdateRequest(tagUpdateRequest{Name: "Work", Color: "red"}); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("expected invalid color error, got %v", err)
	}
	if err := validateTagSelectionRequest(tagSelectionRequest{}); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("expected missing tag error, got %v", err)
	}
	if err := validateTagSelectionRequest(tagSelectionRequest{TagID: uuid.New()}); err != nil {
		t.Fatalf("expected valid tag selection, got %v", err)
	}
}
