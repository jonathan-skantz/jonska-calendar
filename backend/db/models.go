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

// SharedEvent combines the owner's current event fields with the recipient's
// local overrides. Original* values come from the joined source event; they
// are not duplicated columns in the sharing tables.
type SharedEvent struct {
	ShareID             uuid.UUID
	EventID             uuid.UUID
	RecipientUserID     uuid.UUID
	OriginalTitle       string
	OriginalDescription *string
	OriginalURL         *string
	OriginalLocation    *string
	StartsAt            time.Time
	EndsAt              time.Time
	IsAllDay            bool
	Kind                string
	LocalTitle          *string
	LocalDescription    *string
	LocalURL            *string
	LocalLocation       *string
	PrivateCustomInfo   *string
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
	SourceTitle          *string    `json:"source_title,omitempty"`
	CustomInfo           *string    `json:"custom_info,omitempty"`
	Tags                 []Tag      `json:"tags,omitempty"`
	MainTagID            *uuid.UUID `json:"main_tag_id,omitempty"`
}
