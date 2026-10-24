package subscription

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/jonathan-skantz/calendar/backend/db"
	"github.com/jonathan-skantz/calendar/backend/internal/auth"
	"github.com/jonathan-skantz/calendar/backend/internal/ratelimit"
	"github.com/jonathan-skantz/calendar/backend/internal/timeline"
)

const (
	maxNameLength = 100
	defaultColor  = "#547968"
	// maxPerUser bounds the feeds the background refresh fetches for one
	// user.
	maxPerUser = 20
	// fetchesPerWindow bounds how often one user can make the server fetch
	// a feed by subscribing or refreshing.
	fetchesPerWindow = 20
	fetchWindow      = time.Hour
)

var (
	errNameTooLong  = errors.New("name must be at most 100 characters")
	errNameRequired = errors.New("name must be 1 to 100 characters")
	errInvalidColor = errors.New("color must be a #rrggbb hex color")
	errTooMany      = errors.New("you can follow at most 20 calendars")
	errTooOften     = errors.New("too many fetches, try again later")
)

type Handler struct {
	Store       *db.Store
	CurrentUser auth.CurrentUser
	Refresher   *Refresher
	fetches     *ratelimit.Limiter[uuid.UUID]
}

func New(store *db.Store, currentUser auth.CurrentUser, client *Client) *Handler {
	return &Handler{
		Store: store, CurrentUser: currentUser, Refresher: &Refresher{Store: store, Client: client},
		fetches: ratelimit.New[uuid.UUID](fetchesPerWindow, fetchWindow),
	}
}

type createRequest struct {
	URL   string `json:"url"`
	Name  string `json:"name"`
	Color string `json:"color"`
	// TimeZone is the browser's: floating times and all-day dates in the
	// feed are read in it.
	TimeZone string `json:"time_zone"`
}

type updateRequest struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

type subscriptionResponse struct {
	ID            uuid.UUID `json:"id"`
	Name          string    `json:"name"`
	URL           string    `json:"url"`
	Color         string    `json:"color"`
	LastFetchedAt time.Time `json:"last_fetched_at"`
	LastSuccessAt time.Time `json:"last_success_at"`
	LastError     *string   `json:"last_error,omitempty"`
}

func newResponse(subscription db.Subscription) subscriptionResponse {
	return subscriptionResponse{
		ID: subscription.ID, Name: subscription.Name, URL: subscription.URL, Color: subscription.Color,
		LastFetchedAt: subscription.LastFetchedAt, LastSuccessAt: subscription.LastSuccessAt,
		LastError: subscription.LastError,
	}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.RequireUser(w, r, h.CurrentUser)
	if !ok {
		return
	}
	subscriptions, err := h.Store.ListSubscriptions(r.Context(), userID)
	if err != nil {
		writeServerError(w, "failed to list subscriptions", err)
		return
	}
	response := make([]subscriptionResponse, 0, len(subscriptions))
	for _, subscription := range subscriptions {
		response = append(response, newResponse(subscription))
	}
	writeJSON(w, http.StatusOK, response)
}

// Create subscribes to a calendar. The feed is fetched first, so an address
// that does not work is refused and nothing is saved.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.RequireUser(w, r, h.CurrentUser)
	if !ok || auth.RejectDemo(w, r, h.Store, userID) {
		return
	}
	var request createRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, timeline.ErrInvalidRequest.Error(), http.StatusBadRequest)
		return
	}
	feedURL, err := NormalizeURL(request.URL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	loc, err := timeline.ParseTimeZone(request.TimeZone)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	name, color, err := validateFields(request.Name, request.Color)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// Checked before fetching, so a refused subscription costs no fetch.
	existing, err := h.Store.ListSubscriptions(r.Context(), userID)
	if err != nil {
		writeServerError(w, "failed to subscribe", err)
		return
	}
	for _, other := range existing {
		if other.URL == feedURL {
			http.Error(w, db.ErrSubscriptionExists.Error(), http.StatusConflict)
			return
		}
	}
	if len(existing) >= maxPerUser {
		http.Error(w, errTooMany.Error(), http.StatusConflict)
		return
	}
	if !h.fetches.Allow(userID) {
		http.Error(w, errTooOften.Error(), http.StatusTooManyRequests)
		return
	}
	feed, err := h.Refresher.Load(r.Context(), feedURL, loc)
	if IsFeedError(err) {
		http.Error(w, Message(err), http.StatusBadRequest)
		return
	}
	if err != nil {
		writeServerError(w, "failed to subscribe", err)
		return
	}
	if name == "" {
		name = defaultName(feed.Name, feedURL)
	}
	subscription, err := h.Store.CreateSubscription(r.Context(), db.Subscription{
		UserID: userID, Name: name, URL: feedURL, Color: color, TimeZone: loc.String(),
	}, feed.Events)
	if errors.Is(err, db.ErrSubscriptionExists) {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	if err != nil {
		writeServerError(w, "failed to subscribe", err)
		return
	}
	writeJSON(w, http.StatusCreated, newResponse(subscription))
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.RequireUser(w, r, h.CurrentUser)
	if !ok {
		return
	}
	subscriptionID, ok := subscriptionIDFrom(w, r)
	if !ok {
		return
	}
	var request updateRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, timeline.ErrInvalidRequest.Error(), http.StatusBadRequest)
		return
	}
	name, color, err := validateUpdate(request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	subscription, err := h.Store.UpdateSubscription(r.Context(), userID, subscriptionID, name, color)
	writeSubscription(w, subscription, err)
}

// Refresh fetches a subscription now. A feed that fails answers 502 with
// why; the subscription keeps its last events.
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.RequireUser(w, r, h.CurrentUser)
	if !ok || auth.RejectDemo(w, r, h.Store, userID) {
		return
	}
	subscriptionID, ok := subscriptionIDFrom(w, r)
	if !ok {
		return
	}
	subscription, err := h.Store.GetSubscription(r.Context(), userID, subscriptionID)
	if err != nil {
		writeSubscription(w, subscription, err)
		return
	}
	if !h.fetches.Allow(userID) {
		http.Error(w, errTooOften.Error(), http.StatusTooManyRequests)
		return
	}
	subscription, err = h.Refresher.Refresh(r.Context(), subscription)
	if IsFeedError(err) {
		http.Error(w, Message(err), http.StatusBadGateway)
		return
	}
	writeSubscription(w, subscription, err)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.RequireUser(w, r, h.CurrentUser)
	if !ok {
		return
	}
	subscriptionID, ok := subscriptionIDFrom(w, r)
	if !ok {
		return
	}
	err := h.Store.DeleteSubscription(r.Context(), userID, subscriptionID)
	if errors.Is(err, db.ErrSubscriptionNotFound) {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if err != nil {
		writeServerError(w, "failed to unsubscribe", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// validateFields trims the name, which may be empty, and defaults the color.
func validateFields(name, color string) (string, string, error) {
	name = strings.TrimSpace(name)
	if utf8.RuneCountInString(name) > maxNameLength {
		return "", "", errNameTooLong
	}
	if color == "" {
		color = defaultColor
	}
	if !timeline.ValidColor(color) {
		return "", "", errInvalidColor
	}
	return name, color, nil
}

// validateUpdate checks a rename: it replaces both fields, so both are
// required and nothing is defaulted.
func validateUpdate(request updateRequest) (string, string, error) {
	name := strings.TrimSpace(request.Name)
	if name == "" || utf8.RuneCountInString(name) > maxNameLength {
		return "", "", errNameRequired
	}
	if !timeline.ValidColor(request.Color) {
		return "", "", errInvalidColor
	}
	return name, request.Color, nil
}

// defaultName is the calendar's own name, or its host.
func defaultName(calendarName, feedURL string) string {
	if name := strings.TrimSpace(calendarName); name != "" {
		if runes := []rune(name); len(runes) > maxNameLength {
			name = string(runes[:maxNameLength])
		}
		return name
	}
	parsed, err := url.Parse(feedURL)
	if err != nil || parsed.Hostname() == "" {
		return "Calendar"
	}
	return parsed.Hostname()
}

func subscriptionIDFrom(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	subscriptionID, err := uuid.Parse(r.PathValue("subscriptionID"))
	if err != nil {
		http.Error(w, db.ErrSubscriptionNotFound.Error(), http.StatusNotFound)
		return uuid.Nil, false
	}
	return subscriptionID, true
}

func writeSubscription(w http.ResponseWriter, subscription db.Subscription, err error) {
	switch {
	case errors.Is(err, db.ErrSubscriptionNotFound):
		http.Error(w, err.Error(), http.StatusNotFound)
	case err != nil:
		writeServerError(w, "failed to save subscription", err)
	default:
		writeJSON(w, http.StatusOK, newResponse(subscription))
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeServerError(w http.ResponseWriter, message string, err error) {
	log.Printf("%s: %v", message, err)
	http.Error(w, message, http.StatusInternalServerError)
}
