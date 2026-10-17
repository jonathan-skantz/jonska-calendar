package timeline

import (
	"time"

	"github.com/google/uuid"
)

type eventRequest struct {
	Title                string    `json:"title"`
	Description          *string   `json:"description"`
	URL                  *string   `json:"url"`
	StartsAt             time.Time `json:"starts_at"`
	EndsAt               time.Time `json:"ends_at"`
	IsAllDay             bool      `json:"is_all_day"`
	StartIsApproximate   bool      `json:"start_is_approximate"`
	EndIsApproximate     bool      `json:"end_is_approximate"`
	Kind                 string    `json:"kind"`
	Location             *string   `json:"location"`
	CommuteBeforeSeconds int       `json:"commute_before_seconds"`
	CommuteAfterSeconds  int       `json:"commute_after_seconds"`
	MainTagID            uuid.UUID `json:"main_tag_id"`
}

type taskRequest struct {
	Title       string    `json:"title"`
	Description *string   `json:"description"`
	URL         *string   `json:"url"`
	DeadlineAt  time.Time `json:"deadline_at"`
}

type shareRequest struct {
	RecipientEmail string `json:"recipient_email"`
	shareFieldsRequest
}

// shareFieldsRequest carries the owner's per-recipient values. An empty or
// missing field means "same as the event".
type shareFieldsRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	URL         *string `json:"url"`
	Location    *string `json:"location"`
}

type tagRequest struct {
	Name string `json:"name"`
}

type tagUpdateRequest struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

type tagSelectionRequest struct {
	TagID uuid.UUID `json:"tag_id"`
}

type taskCompletionRequest struct {
	IsCompleted bool `json:"is_completed"`
}

type transitCardDaysRequest struct {
	From        string   `json:"from"`
	To          string   `json:"to"`
	NeededDates []string `json:"needed_dates"`
	MaybeDates  []string `json:"maybe_dates"`
}

type transitCardDayResponse struct {
	Date        string    `json:"date"`
	UserID      uuid.UUID `json:"user_id"`
	DisplayName string    `json:"display_name"`
	Need        string    `json:"need"`
	Status      string    `json:"status"`
}

type householdRequest struct {
	Name             string `json:"name"`
	TransitCardCount int    `json:"transit_card_count"`
}

type householdMemberRequest struct {
	Email string `json:"email"`
}

type createResponse struct {
	ID uuid.UUID `json:"id"`
}
