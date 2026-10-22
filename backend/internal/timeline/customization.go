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
}

// customize applies the recipient's customization of a received event and
// keeps what they received as Original. The shifts are left out on all-day
// events, whose edges are midnights, and when the shifted event would not
// end after it starts, as after the owner shortened it.
func customize(item *TimelineItem) {
	custom := item.Customization
	if custom == nil || item.StartsAt == nil || item.EndsAt == nil {
		return
	}
	item.Original = &ReceivedValues{
		Title: item.Title, Description: item.Description, URL: item.URL, Location: item.Location,
		StartsAt: *item.StartsAt, EndsAt: *item.EndsAt,
	}
	if custom.Title != nil {
		item.Title = *custom.Title
	}
	item.Description = customized(custom.Description, item.Description)
	item.URL = customized(custom.URL, item.URL)
	item.Location = customized(custom.Location, item.Location)
	if item.IsAllDay {
		return
	}
	startsAt := item.StartsAt.Add(time.Duration(custom.StartShiftSeconds) * time.Second)
	endsAt := item.EndsAt.Add(time.Duration(custom.EndShiftSeconds) * time.Second)
	if endsAt.After(startsAt) {
		item.StartsAt, item.EndsAt = &startsAt, &endsAt
	}
}

func customized(custom, received *string) *string {
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
