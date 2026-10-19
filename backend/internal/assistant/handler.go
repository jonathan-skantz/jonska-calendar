package assistant

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/google/uuid"
	"github.com/jonathan-skantz/calendar/backend/db"
	"github.com/jonathan-skantz/calendar/backend/internal/auth"
	"github.com/jonathan-skantz/calendar/backend/internal/ratelimit"
)

const (
	maxTextLength = 1000
	// A message can make several model calls, so the budget is smaller than
	// quick add's.
	messagesPerWindow = 20
	rateWindow        = time.Hour
	turnTimeout       = 2 * time.Minute
)

type Handler struct {
	Store       Store
	CurrentUser auth.CurrentUser
	// Client is nil when no API key is configured.
	Client *Client

	now           func() time.Time
	limiter       *ratelimit.Limiter
	conversations *conversations
}

// New takes a nil client when the assistant is not configured; requests then
// get 503 instead of failing at startup.
func New(store Store, currentUser auth.CurrentUser, client *Client) *Handler {
	return &Handler{
		Store: store, CurrentUser: currentUser, Client: client,
		now:           time.Now,
		limiter:       ratelimit.New(messagesPerWindow, rateWindow),
		conversations: newConversations(time.Now),
	}
}

type messageRequest struct {
	ConversationID uuid.UUID `json:"conversation_id"`
	Text           string    `json:"text"`
	TimeZone       string    `json:"time_zone"`
	// WeekStart is 0 for Sunday to 6 for Saturday, like date-fns.
	WeekStart int `json:"week_start"`

	location *time.Location
}

type messageResponse struct {
	ConversationID uuid.UUID  `json:"conversation_id"`
	Reply          string     `json:"reply"`
	Proposals      []Proposal `json:"proposals"`
	Steps          []Step     `json:"steps"`
	// Ended means the conversation cannot take another message.
	Ended bool `json:"ended"`
}

// Message runs one user message through the agent. The agent only reads the
// calendar and proposes events; it never writes.
func (h *Handler) Message(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.RequireUser(w, r, h.CurrentUser)
	if !ok {
		return
	}
	request, err := decodeMessage(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if h.Client == nil {
		http.Error(w, "The assistant is not configured", http.StatusServiceUnavailable)
		return
	}
	if !h.limiter.Allow(userID) {
		http.Error(w, "Too many messages, try again later", http.StatusTooManyRequests)
		return
	}
	id, history, err := h.conversations.claim(userID, request.ConversationID)
	if err != nil {
		status, message := claimError(err)
		http.Error(w, message, status)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), turnTimeout)
	defer cancel()
	response, next, err := h.answer(ctx, userID, request, history)
	if err != nil {
		// The earlier history is untouched, so the message can be retried.
		h.conversations.unlock(id)
		log.Printf("assistant: %v", err)
		http.Error(w, "Couldn't answer right now, try again", http.StatusBadGateway)
		return
	}
	h.conversations.save(id, next)
	response.ConversationID = id

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

// decodeMessage reads and validates the request body.
func decodeMessage(r *http.Request) (messageRequest, error) {
	var request messageRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return request, errors.New("invalid request body")
	}
	request.Text = strings.TrimSpace(request.Text)
	if request.Text == "" || utf8.RuneCountInString(request.Text) > maxTextLength {
		return request, fmt.Errorf("text must be 1 to %d characters", maxTextLength)
	}
	location, err := time.LoadLocation(request.TimeZone)
	if err != nil || request.TimeZone == "" || request.TimeZone == "Local" {
		return request, errors.New("time_zone must be an IANA time zone")
	}
	request.location = location
	if request.WeekStart < 0 || request.WeekStart > 6 {
		return request, errors.New("week_start must be 0 to 6")
	}
	return request, nil
}

func claimError(err error) (int, string) {
	switch {
	case errors.Is(err, errConversationBusy):
		return http.StatusConflict, "Still working on the previous message"
	case errors.Is(err, errConversationFull):
		return http.StatusConflict, "This conversation is full, start a new one"
	default:
		return http.StatusNotFound, "This conversation has ended, start a new one"
	}
}

// answer runs the agent on one message. It returns the history to keep, which
// is nil when the conversation cannot take another message.
func (h *Handler) answer(ctx context.Context, userID uuid.UUID, request messageRequest, history []anthropic.BetaMessageParam) (messageResponse, []anthropic.BetaMessageParam, error) {
	tags, err := h.Store.ListTags(ctx, userID)
	if err != nil {
		return messageResponse{}, nil, fmt.Errorf("listing tags: %w", err)
	}
	now := h.now()
	current := newTurn(h.Store, userID, request.location, now, tags)
	message := anthropic.NewBetaUserMessage(
		anthropic.NewBetaTextBlock(contextBlock(now, request.location, request.WeekStart, tags)),
		anthropic.NewBetaTextBlock(request.Text),
	)

	started := time.Now()
	runner := h.Client.runner(append(history, message), current.tools())
	final, err := runner.RunToCompletion(ctx)
	elapsed := time.Since(started).Round(time.Millisecond)
	if err != nil {
		return messageResponse{}, nil, fmt.Errorf("after %s and %d calls: %w", elapsed, runner.IterationCount(), err)
	}
	log.Printf("assistant: answered in %s with %d calls", elapsed, runner.IterationCount())

	response := messageResponse{Proposals: current.proposals, Steps: current.steps}
	switch final.StopReason {
	case anthropic.BetaStopReasonEndTurn:
		response.Reply = replyText(final)
		return response, runner.Messages(), nil
	case anthropic.BetaStopReasonRefusal:
		response.Reply = "I can't help with that one."
	default:
		// The iteration limit, which leaves tool calls unanswered, or the
		// output limit; either way the history cannot take another message.
		response.Reply = "I couldn't finish that. Try a more specific request."
	}
	response.Ended = true
	return response, nil, nil
}

// contextBlock starts each user message, so the model always knows the
// current time. It sits in the message rather than the system prompt to keep
// the cached prefix stable.
func contextBlock(now time.Time, location *time.Location, weekStart int, tags []db.Tag) string {
	tagList := "(none)"
	if len(tags) > 0 {
		names := make([]string, len(tags))
		for i, tag := range tags {
			names[i] = fmt.Sprintf("%q", tag.Name)
		}
		tagList = strings.Join(names, ", ")
	}
	return fmt.Sprintf("<context>\nNow: %s (%s)\nWeeks start on %s.\nTags: %s\n</context>",
		now.In(location).Format(timeFormat), location, time.Weekday(weekStart), tagList)
}

func replyText(message *anthropic.BetaMessage) string {
	var parts []string
	for _, block := range message.Content {
		if text := strings.TrimSpace(block.Text); block.Type == "text" && text != "" {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, "\n\n")
}
