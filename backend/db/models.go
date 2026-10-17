package db

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	DisplayName  *string
}

// TransitCardDay is one reported day. Days without a row are not reported.
type TransitCardDay struct {
	Day         time.Time
	UserID      uuid.UUID
	DisplayName string
	Need        string
}

type HouseholdMember struct {
	UserID      uuid.UUID `json:"user_id"`
	DisplayName string    `json:"display_name"`
}

type Household struct {
	ID               uuid.UUID         `json:"id"`
	Name             string            `json:"name"`
	TransitCardCount int               `json:"transit_card_count"`
	Members          []HouseholdMember `json:"members"`
}

type Event struct {
	ID                   uuid.UUID
	Title                string
	Description          *string
	URL                  *string
	StartsAt             time.Time
	EndsAt               time.Time
	IsAllDay             bool
	Kind                 string
	Location             *string
	CommuteBeforeSeconds int
	CommuteAfterSeconds  int
	MainTagID            uuid.UUID
}

type Task struct {
	ID          uuid.UUID
	Title       string
	Description *string
	URL         *string
	DeadlineAt  time.Time
	IsCompleted bool
}

type Tag struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Color     string    `json:"color"`
	IsDefault bool      `json:"is_default"`
}

// ShareFields are the owner's per-recipient presentation of a shared event.
// A nil field means the recipient sees the event's own value.
type ShareFields struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	URL         *string `json:"url,omitempty"`
	Location    *string `json:"location,omitempty"`
}

// EventShare is the owner's view of one recipient of an event.
type EventShare struct {
	ID                   uuid.UUID `json:"id"`
	RecipientUserID      uuid.UUID `json:"recipient_user_id"`
	RecipientEmail       string    `json:"recipient_email"`
	RecipientDisplayName string    `json:"recipient_display_name"`
	ShareFields
}

type TimelineEntry struct {
	ID                   uuid.UUID  `json:"id"`
	Type                 string     `json:"type"`
	Title                string     `json:"title"`
	Description          *string    `json:"description,omitempty"`
	URL                  *string    `json:"url,omitempty"`
	StartsAt             *time.Time `json:"starts_at,omitempty"`
	EndsAt               *time.Time `json:"ends_at,omitempty"`
	IsAllDay             bool       `json:"is_all_day,omitempty"`
	DeadlineAt           *time.Time `json:"deadline_at,omitempty"`
	Kind                 *string    `json:"kind,omitempty"`
	Location             *string    `json:"location,omitempty"`
	CommuteBeforeSeconds int        `json:"commute_before_seconds,omitempty"`
	CommuteAfterSeconds  int        `json:"commute_after_seconds,omitempty"`
	IsCompleted          bool       `json:"is_completed"`
	IsShared             bool       `json:"is_shared,omitempty"`
	SharedBy             *string    `json:"shared_by,omitempty"`
	ShareCount           int        `json:"share_count,omitempty"`
	Tags                 []Tag      `json:"tags,omitempty"`
	MainTagID            *uuid.UUID `json:"main_tag_id,omitempty"`
}
