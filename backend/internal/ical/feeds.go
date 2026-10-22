package ical

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/jonathan-skantz/calendar/backend/db"
	"github.com/jonathan-skantz/calendar/backend/internal/auth"
	"github.com/jonathan-skantz/calendar/backend/internal/timeline"
)

const maxFeedNameLength = 100

var errInvalidFeedName = errors.New("name must be 1 to 100 characters")

type feedRequest struct {
	Name          string     `json:"name"`
	TagID         *uuid.UUID `json:"tag_id"`
	IncludeAlarms bool       `json:"include_alarms"`
	// TimeZone is read on create only: all-day dates are written in it.
	TimeZone string `json:"time_zone"`
}

type feedResponse struct {
	ID            uuid.UUID  `json:"id"`
	Name          string     `json:"name"`
	TagID         *uuid.UUID `json:"tag_id,omitempty"`
	IncludeAlarms bool       `json:"include_alarms"`
	// Path is where the feed is served; the client adds its own origin.
	Path string `json:"path"`
}

func newFeedResponse(feed db.Feed) feedResponse {
	return feedResponse{
		ID: feed.ID, Name: feed.Name, TagID: feed.TagID, IncludeAlarms: feed.IncludeAlarms,
		Path: "/api/feeds/" + feed.Token + "/calendar.ics",
	}
}

func (request feedRequest) fields() (db.FeedFields, error) {
	name := strings.TrimSpace(request.Name)
	if name == "" || utf8.RuneCountInString(name) > maxFeedNameLength {
		return db.FeedFields{}, errInvalidFeedName
	}
	return db.FeedFields{Name: name, TagID: request.TagID, IncludeAlarms: request.IncludeAlarms}, nil
}

// newFeedToken returns 256 random bits, URL-safe. The token is the only
// credential a feed request has.
func newFeedToken() (string, error) {
	token := make([]byte, 32)
	if _, err := rand.Read(token); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(token), nil
}

func (h *Handler) ListFeeds(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.RequireUser(w, r, h.CurrentUser)
	if !ok {
		return
	}
	feeds, err := h.Store.ListFeeds(r.Context(), userID)
	if err != nil {
		writeServerError(w, "failed to list feeds", err)
		return
	}
	response := make([]feedResponse, 0, len(feeds))
	for _, feed := range feeds {
		response = append(response, newFeedResponse(feed))
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) CreateFeed(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.RequireUser(w, r, h.CurrentUser)
	if !ok {
		return
	}
	request, fields, ok := decodeFeedRequest(w, r)
	if !ok {
		return
	}
	loc, err := timeline.ParseTimeZone(request.TimeZone)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	token, err := newFeedToken()
	if err != nil {
		writeServerError(w, "failed to create feed", err)
		return
	}
	feed, err := h.Store.CreateFeed(r.Context(), userID, fields, token, loc.String())
	h.writeFeed(w, http.StatusCreated, feed, err)
}

func (h *Handler) UpdateFeed(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.RequireUser(w, r, h.CurrentUser)
	if !ok {
		return
	}
	feedID, ok := feedIDFrom(w, r)
	if !ok {
		return
	}
	_, fields, ok := decodeFeedRequest(w, r)
	if !ok {
		return
	}
	feed, err := h.Store.UpdateFeed(r.Context(), userID, feedID, fields)
	h.writeFeed(w, http.StatusOK, feed, err)
}

// ResetFeedToken gives the feed a new URL; the old one stops working.
func (h *Handler) ResetFeedToken(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.RequireUser(w, r, h.CurrentUser)
	if !ok {
		return
	}
	feedID, ok := feedIDFrom(w, r)
	if !ok {
		return
	}
	token, err := newFeedToken()
	if err != nil {
		writeServerError(w, "failed to reset feed", err)
		return
	}
	feed, err := h.Store.ResetFeedToken(r.Context(), userID, feedID, token)
	h.writeFeed(w, http.StatusOK, feed, err)
}

func (h *Handler) DeleteFeed(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.RequireUser(w, r, h.CurrentUser)
	if !ok {
		return
	}
	feedID, ok := feedIDFrom(w, r)
	if !ok {
		return
	}
	err := h.Store.DeleteFeed(r.Context(), userID, feedID)
	if errors.Is(err, db.ErrFeedNotFound) {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if err != nil {
		writeServerError(w, "failed to delete feed", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ServeFeed answers a feed's events to calendar apps. It takes no session:
// the secret token in the path is the credential.
func (h *Handler) ServeFeed(w http.ResponseWriter, r *http.Request) {
	feed, err := h.Store.FeedByToken(r.Context(), r.PathValue("token"))
	if errors.Is(err, db.ErrFeedNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		writeServerError(w, "failed to load feed", err)
		return
	}
	loc, err := time.LoadLocation(feed.TimeZone)
	if err != nil {
		writeServerError(w, "failed to load feed", err)
		return
	}
	events, err := h.Store.ListEventsForFeed(r.Context(), feed)
	if err != nil {
		writeServerError(w, "failed to load feed", err)
		return
	}
	var file bytes.Buffer
	if err := EncodeFeed(&file, feed.Name, events, loc, h.now()); err != nil {
		writeServerError(w, "failed to load feed", err)
		return
	}
	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	w.Header().Set("Cache-Control", "private, max-age=300")
	_, _ = file.WriteTo(w)
}

func decodeFeedRequest(w http.ResponseWriter, r *http.Request) (feedRequest, db.FeedFields, bool) {
	var request feedRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, timeline.ErrInvalidRequest.Error(), http.StatusBadRequest)
		return request, db.FeedFields{}, false
	}
	fields, err := request.fields()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return request, db.FeedFields{}, false
	}
	return request, fields, true
}

func feedIDFrom(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	feedID, err := uuid.Parse(r.PathValue("feedID"))
	if err != nil {
		http.Error(w, "feed not found", http.StatusNotFound)
		return uuid.Nil, false
	}
	return feedID, true
}

func (h *Handler) writeFeed(w http.ResponseWriter, status int, feed db.Feed, err error) {
	switch {
	case errors.Is(err, db.ErrFeedNotFound), errors.Is(err, db.ErrTagNotFound):
		http.Error(w, err.Error(), http.StatusNotFound)
	case err != nil:
		writeServerError(w, "failed to save feed", err)
	default:
		writeJSON(w, status, newFeedResponse(feed))
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
