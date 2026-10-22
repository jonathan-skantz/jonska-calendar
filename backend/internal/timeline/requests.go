package timeline

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jonathan-skantz/calendar/backend/db"
	"github.com/jonathan-skantz/calendar/backend/internal/recurrence"
)

// decodeJSON reads the request body into dst. On malformed JSON it answers
// 400 and returns false.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		http.Error(w, ErrInvalidRequest.Error(), http.StatusBadRequest)
		return false
	}
	return true
}

// pathID parses the UUID path value called name, such as "eventID". On
// failure it answers 400 ("invalid event id") and returns false.
func pathID(w http.ResponseWriter, r *http.Request, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(r.PathValue(name))
	if err != nil {
		http.Error(w, "invalid "+strings.TrimSuffix(name, "ID")+" id", http.StatusBadRequest)
		return uuid.Nil, false
	}
	return id, true
}

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
	IsSkipped            bool      `json:"is_skipped"`
	Location             *string   `json:"location"`
	GatheringSeconds     int       `json:"gathering_seconds"`
	GatheringIsOptional  bool      `json:"gathering_is_optional"`
	GatheringLabel       *string   `json:"gathering_label"`
	CommuteBeforeSeconds int       `json:"commute_before_seconds"`
	CommuteAfterSeconds  int       `json:"commute_after_seconds"`
	MainTagID            uuid.UUID `json:"main_tag_id"`
	// Recurrence makes the event repeat; nil makes it a plain event.
	// TimeZone is the browser's, which its times repeat in, and is required
	// with a rule.
	Recurrence *recurrence.Rule `json:"recurrence"`
	TimeZone   string           `json:"time_zone"`
}

func (request eventRequest) toEvent() db.Event {
	return db.Event{
		Title:                request.Title,
		Description:          request.Description,
		URL:                  request.URL,
		StartsAt:             request.StartsAt,
		EndsAt:               request.EndsAt,
		IsAllDay:             request.IsAllDay,
		StartIsApproximate:   request.StartIsApproximate,
		EndIsApproximate:     request.EndIsApproximate,
		Kind:                 request.Kind,
		IsSkipped:            request.IsSkipped,
		Location:             request.Location,
		GatheringSeconds:     request.GatheringSeconds,
		GatheringIsOptional:  request.GatheringIsOptional,
		GatheringLabel:       request.GatheringLabel,
		CommuteBeforeSeconds: request.CommuteBeforeSeconds,
		CommuteAfterSeconds:  request.CommuteAfterSeconds,
		MainTagID:            request.MainTagID,
	}
}

type reminderRequest struct {
	OffsetSeconds int `json:"offset_seconds"`
}

type reminderNotifiedRequest struct {
	OccurrenceStart *time.Time `json:"occurrence_start"`
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

// customizationRequest carries a recipient's own values for an event they
// receive. An empty or missing field and a zero shift keep the received
// value.
type customizationRequest struct {
	Title             *string `json:"title"`
	Description       *string `json:"description"`
	URL               *string `json:"url"`
	Location          *string `json:"location"`
	Note              *string `json:"note"`
	StartShiftSeconds int     `json:"start_shift_seconds"`
	EndShiftSeconds   int     `json:"end_shift_seconds"`
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
