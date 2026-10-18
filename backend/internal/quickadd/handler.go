package quickadd

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jonathan-skantz/calendar/backend/db"
	"github.com/jonathan-skantz/calendar/backend/internal/auth"
)

const (
	maxTextLength = 500
	// Each parse is a paid upstream call, so a user gets a small budget.
	parsesPerWindow = 30
	rateWindow      = time.Hour
)

type TagLister interface {
	ListTags(ctx context.Context, userID uuid.UUID) ([]db.Tag, error)
}

type parser interface {
	Parse(ctx context.Context, text string, tagNames []string) (RawDraft, error)
}

type Handler struct {
	Tags        TagLister
	CurrentUser auth.CurrentUser
	// Client is nil when no API key is configured.
	Client parser
	now    func() time.Time

	mu      sync.Mutex
	windows map[uuid.UUID]rateCount
}

type rateCount struct {
	start time.Time
	count int
}

// New takes a nil client when quick add is not configured; requests then get
// 503 instead of failing at startup.
func New(tags TagLister, currentUser auth.CurrentUser, client *Client) *Handler {
	handler := &Handler{Tags: tags, CurrentUser: currentUser, now: time.Now, windows: map[uuid.UUID]rateCount{}}
	if client != nil {
		handler.Client = client
	}
	return handler
}

// Parse turns a sentence into an event draft for the browser to review. It
// never writes anything: the user saves the event through the normal form.
func (h *Handler) Parse(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.RequireUser(w, r, h.CurrentUser)
	if !ok {
		return
	}
	var request struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	text := strings.TrimSpace(request.Text)
	if text == "" || utf8.RuneCountInString(text) > maxTextLength {
		http.Error(w, "text must be 1 to 500 characters", http.StatusBadRequest)
		return
	}
	if h.Client == nil {
		http.Error(w, "Quick add is not configured", http.StatusServiceUnavailable)
		return
	}
	if !h.allow(userID) {
		http.Error(w, "Too many quick adds, try again later", http.StatusTooManyRequests)
		return
	}

	tags, err := h.Tags.ListTags(r.Context(), userID)
	if err != nil {
		log.Printf("quickadd: listing tags: %v", err)
		http.Error(w, "failed to load tags", http.StatusInternalServerError)
		return
	}
	names := make([]string, len(tags))
	for i, tag := range tags {
		names[i] = tag.Name
	}

	started := time.Now()
	raw, err := h.Client.Parse(r.Context(), text, names)
	if err != nil {
		log.Printf("quickadd: parsing after %s: %v", time.Since(started).Round(time.Millisecond), err)
		http.Error(w, "Couldn't read that right now, try again", http.StatusBadGateway)
		return
	}

	log.Printf("quickadd: parsed in %s", time.Since(started).Round(time.Millisecond))

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(validateDraft(raw, tags))
}

// allow counts parses per user in fixed windows. It is in memory, which is
// enough for the single server this app runs on.
func (h *Handler) allow(userID uuid.UUID) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	now := h.now()
	window := h.windows[userID]
	if now.Sub(window.start) >= rateWindow {
		window = rateCount{start: now}
	}
	if window.count >= parsesPerWindow {
		return false
	}
	window.count++
	h.windows[userID] = window
	return true
}
