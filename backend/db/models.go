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

// Event kinds. Attendable events take up the user's time; informational
// ones are heads-ups.
const (
	EventKindAttendable    = "attendable"
	EventKindInformational = "informational"
)

// Limits shared by event validation and quick add. They mirror the
// database's CHECK constraints.
const (
	MaxCommuteSeconds        = 24 * 60 * 60
	MaxGatheringLabelLength  = 40
	MaxReminderOffsetSeconds = 30 * 24 * 60 * 60
)

type Event struct {
	ID                   uuid.UUID
	Title                string
	Description          *string
	URL                  *string
	StartsAt             time.Time
	EndsAt               time.Time
	IsAllDay             bool
	StartIsApproximate   bool
	EndIsApproximate     bool
	Kind                 string
	IsSkipped            bool
	Location             *string
	GatheringSeconds     int
	GatheringIsOptional  bool
	GatheringLabel       *string
	CommuteBeforeSeconds int
	CommuteAfterSeconds  int
	MainTagID            uuid.UUID
	// A repeating event has a rule, an RRULE value, and the time zone its
	// wall-clock times repeat in; StartsAt and EndsAt are its first
	// occurrence. RecurrenceEndsAt is when the last occurrence ends, nil
	// when it never ends.
	RecurrenceRule     *string
	RecurrenceTimeZone *string
	RecurrenceEndsAt   *time.Time
}

// OccurrenceException is an occurrence of a repeating event that differs
// from its series, identified by the start the rule gives it.
type OccurrenceException struct {
	OriginalStart time.Time
	IsCancelled   bool
	// StartsAt and EndsAt are set together when the occurrence was moved.
	StartsAt *time.Time
	EndsAt   *time.Time
	// IsSkipped is the owner's decision; nil follows the series. It is never
	// shown to recipients.
	IsSkipped *bool
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

// Customization is a recipient's own change to an event they receive. A nil
// field and a zero shift keep the received value. Note is the recipient's
// own text, shown beside the event's.
type Customization struct {
	Title             *string `json:"title,omitempty"`
	Description       *string `json:"description,omitempty"`
	URL               *string `json:"url,omitempty"`
	Location          *string `json:"location,omitempty"`
	Note              *string `json:"note,omitempty"`
	StartShiftSeconds int     `json:"start_shift_seconds,omitempty"`
	EndShiftSeconds   int     `json:"end_shift_seconds,omitempty"`
}

// EventShare is the owner's view of one recipient of an event.
type EventShare struct {
	ID                   uuid.UUID `json:"id"`
	RecipientUserID      uuid.UUID `json:"recipient_user_id"`
	RecipientEmail       string    `json:"recipient_email"`
	RecipientDisplayName string    `json:"recipient_display_name"`
	ShareFields
}

// Reminder carries the event fields a client needs to deliver it without
// loading the timeline. RemindAt is the event start minus the offset. A
// reminder is delivered once per occurrence, which OccurrenceStart, the
// occurrence's start, identifies.
type Reminder struct {
	ID              uuid.UUID  `json:"id"`
	EventID         uuid.UUID  `json:"event_id"`
	OffsetSeconds   int        `json:"offset_seconds"`
	RemindAt        time.Time  `json:"remind_at"`
	OccurrenceStart time.Time  `json:"occurrence_start"`
	NotifiedAt      *time.Time `json:"notified_at,omitempty"`
	EventTitle      string     `json:"event_title"`
	EventLocation   *string    `json:"event_location,omitempty"`
	EventStartsAt   time.Time  `json:"event_starts_at"`
	EventEndsAt     time.Time  `json:"event_ends_at"`
	EventIsAllDay   bool       `json:"event_is_all_day,omitempty"`
	EventIsSkipped  bool       `json:"event_is_skipped,omitempty"`
	IsShared        bool       `json:"is_shared,omitempty"`
	// A reminder of a repeating event is listed once, with the series, for
	// the caller to expand: the event fields are the first occurrence's,
	// and Deliveries holds when each occurrence, by its UTC start, was
	// delivered.
	RecurrenceRule     *string                 `json:"-"`
	RecurrenceTimeZone *string                 `json:"-"`
	Exceptions         []OccurrenceException   `json:"-"`
	Deliveries         map[time.Time]time.Time `json:"-"`
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
	StartIsApproximate   bool       `json:"start_is_approximate,omitempty"`
	EndIsApproximate     bool       `json:"end_is_approximate,omitempty"`
	DeadlineAt           *time.Time `json:"deadline_at,omitempty"`
	Kind                 *string    `json:"kind,omitempty"`
	IsSkipped            bool       `json:"is_skipped,omitempty"`
	Location             *string    `json:"location,omitempty"`
	GatheringSeconds     int        `json:"gathering_seconds,omitempty"`
	GatheringIsOptional  bool       `json:"gathering_is_optional,omitempty"`
	GatheringLabel       *string    `json:"gathering_label,omitempty"`
	CommuteBeforeSeconds int        `json:"commute_before_seconds,omitempty"`
	CommuteAfterSeconds  int        `json:"commute_after_seconds,omitempty"`
	IsCompleted          bool       `json:"is_completed"`
	IsShared             bool       `json:"is_shared,omitempty"`
	SharedBy             *string    `json:"shared_by,omitempty"`
	ShareCount           int        `json:"share_count,omitempty"`
	// Subscription is set on the read-only events of a subscribed calendar.
	Subscription *SubscriptionRef `json:"subscription,omitempty"`
	// Customization is the recipient's own change to a received event.
	Customization *Customization `json:"customization,omitempty"`
	Tags          []Tag          `json:"tags,omitempty"`
	MainTagID     *uuid.UUID     `json:"main_tag_id,omitempty"`
	// The series fields of a repeating event, expanded into occurrences
	// before they reach a client.
	RecurrenceRule     *string               `json:"-"`
	RecurrenceTimeZone *string               `json:"-"`
	RecurrenceEndsAt   *time.Time            `json:"-"`
	Exceptions         []OccurrenceException `json:"-"`
}

// Event returns the stored fields of one of the user's own events, ready to
// change and pass to UpdateEvent. Entries of events shared with the user hold
// the recipient's view instead, so they must not be written back.
func (entry TimelineEntry) Event() Event {
	event := Event{
		ID: entry.ID, Title: entry.Title, Description: entry.Description, URL: entry.URL,
		IsAllDay: entry.IsAllDay, StartIsApproximate: entry.StartIsApproximate, EndIsApproximate: entry.EndIsApproximate,
		IsSkipped: entry.IsSkipped, Location: entry.Location,
		GatheringSeconds: entry.GatheringSeconds, GatheringIsOptional: entry.GatheringIsOptional, GatheringLabel: entry.GatheringLabel,
		CommuteBeforeSeconds: entry.CommuteBeforeSeconds, CommuteAfterSeconds: entry.CommuteAfterSeconds,
		RecurrenceRule: entry.RecurrenceRule, RecurrenceTimeZone: entry.RecurrenceTimeZone,
		RecurrenceEndsAt: entry.RecurrenceEndsAt,
	}
	if entry.StartsAt != nil && entry.EndsAt != nil {
		event.StartsAt, event.EndsAt = *entry.StartsAt, *entry.EndsAt
	}
	if entry.Kind != nil {
		event.Kind = *entry.Kind
	}
	if entry.MainTagID != nil {
		event.MainTagID = *entry.MainTagID
	}
	return event
}
