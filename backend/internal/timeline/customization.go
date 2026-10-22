package timeline

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/jonathan-skantz/calendar/backend/db"
)

// ReceivedValues are a received event's values as the owner or the feed
// gives them to the recipient, before the recipient's customization. They
// never include the owner's own values of fields set per recipient.
type ReceivedValues struct {
	Title       string    `json:"title"`
	Description *string   `json:"description,omitempty"`
	URL         *string   `json:"url,omitempty"`
	Location    *string   `json:"location,omitempty"`
	StartsAt    time.Time `json:"starts_at"`
	EndsAt      time.Time `json:"ends_at"`
	// Gathering is shared with recipients; commute never is, so received
	// events have none.
	GatheringSeconds    int     `json:"gathering_seconds,omitempty"`
	GatheringIsOptional bool    `json:"gathering_is_optional,omitempty"`
	GatheringLabel      *string `json:"gathering_label,omitempty"`
}

// customize applies the recipient's customization of a received event and
// keeps what they received as Original.
func customize(item *TimelineItem) {
	custom := item.Customization
	if custom == nil || item.StartsAt == nil || item.EndsAt == nil {
		return
	}
	received := ReceivedValues{
		Title: item.Title, Description: item.Description, URL: item.URL, Location: item.Location,
		StartsAt: *item.StartsAt, EndsAt: *item.EndsAt,
		GatheringSeconds: item.GatheringSeconds, GatheringIsOptional: item.GatheringIsOptional,
		GatheringLabel: item.GatheringLabel,
	}
	values := received.customized(*custom, item.IsAllDay)
	item.Original = &received
	item.Title, item.Description, item.URL, item.Location = values.Title, values.Description, values.URL, values.Location
	item.StartsAt, item.EndsAt = &values.StartsAt, &values.EndsAt
	// Commute and gathering only make sense on timed events the recipient
	// attends, as on their own events.
	if item.Kind != nil && *item.Kind == db.EventKindAttendable && !item.IsAllDay {
		item.CommuteBeforeSeconds, item.CommuteAfterSeconds = custom.CommuteBeforeSeconds, custom.CommuteAfterSeconds
		item.GatheringSeconds, item.GatheringIsOptional, item.GatheringLabel =
			values.GatheringSeconds, values.GatheringIsOptional, values.GatheringLabel
	}
}

// customized returns the values with the customization applied. The shifts
// are left out on all-day events, whose edges are midnights, and when the
// shifted event would not end after it starts, as after the owner shortened
// it.
func (values ReceivedValues) customized(custom db.Customization, isAllDay bool) ReceivedValues {
	if custom.Title != nil {
		values.Title = *custom.Title
	}
	values.Description = customizedText(custom.Description, values.Description)
	values.URL = customizedText(custom.URL, values.URL)
	values.Location = customizedText(custom.Location, values.Location)
	if custom.GatheringSeconds != nil {
		values.GatheringSeconds, values.GatheringIsOptional, values.GatheringLabel =
			*custom.GatheringSeconds, custom.GatheringIsOptional, custom.GatheringLabel
	}
	if isAllDay {
		return values
	}
	startsAt := values.StartsAt.Add(time.Duration(custom.StartShiftSeconds) * time.Second)
	endsAt := values.EndsAt.Add(time.Duration(custom.EndShiftSeconds) * time.Second)
	if endsAt.After(startsAt) {
		values.StartsAt, values.EndsAt = startsAt, endsAt
	}
	return values
}

func customizedText(custom, received *string) *string {
	if custom != nil {
		return custom
	}
	return received
}

// Only the recipient customizes an event they receive. The store finds the
// share or the subscription through the caller, so another user's event is
// not found.

var (
	sharedEventNotFound = mappedErrorResponse{
		cause:    db.ErrSharedEventNotFound,
		response: httpErrorResponse{status: http.StatusNotFound, message: "event not found"},
	}
	subscribedEventNotFound = mappedErrorResponse{
		cause:    db.ErrSubscribedEventNotFound,
		response: httpErrorResponse{status: http.StatusNotFound, message: "event not found"},
	}
	eventWithoutUID = mappedErrorResponse{
		cause:    db.ErrEventWithoutUID,
		response: httpErrorResponse{status: http.StatusConflict, message: db.ErrEventWithoutUID.Error()},
	}
)

type (
	customizeFunc   func(ctx context.Context, userID, eventID uuid.UUID, customization db.Customization) error
	uncustomizeFunc func(ctx context.Context, userID, eventID uuid.UUID) error
)

func (h *Handler) CustomizeSharedEvent(w http.ResponseWriter, r *http.Request) {
	h.customizeEvent(w, r, h.Store.CustomizeSharedEvent, sharedEventNotFound)
}

func (h *Handler) UncustomizeSharedEvent(w http.ResponseWriter, r *http.Request) {
	h.uncustomizeEvent(w, r, h.Store.UncustomizeSharedEvent, sharedEventNotFound)
}

func (h *Handler) CustomizeSubscribedEvent(w http.ResponseWriter, r *http.Request) {
	h.customizeEvent(w, r, h.Store.CustomizeSubscribedEvent, subscribedEventNotFound, eventWithoutUID)
}

func (h *Handler) UncustomizeSubscribedEvent(w http.ResponseWriter, r *http.Request) {
	h.uncustomizeEvent(w, r, h.Store.UncustomizeSubscribedEvent, subscribedEventNotFound, eventWithoutUID)
}

func (h *Handler) customizeEvent(w http.ResponseWriter, r *http.Request, customize customizeFunc, mappings ...mappedErrorResponse) {
	userID, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	eventID, ok := pathID(w, r, "eventID")
	if !ok {
		return
	}
	var request customizationRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	customization, err := validateCustomizationRequest(request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := customize(r.Context(), userID, eventID, customization); err != nil {
		writeMappedError(w, err,
			httpErrorResponse{status: http.StatusInternalServerError, message: "failed to customize event"},
			mappings...)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) uncustomizeEvent(w http.ResponseWriter, r *http.Request, uncustomize uncustomizeFunc, mappings ...mappedErrorResponse) {
	userID, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	eventID, ok := pathID(w, r, "eventID")
	if !ok {
		return
	}
	if err := uncustomize(r.Context(), userID, eventID); err != nil {
		writeMappedError(w, err,
			httpErrorResponse{status: http.StatusInternalServerError, message: "failed to remove customization"},
			mappings...)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
