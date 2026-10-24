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
	limiter       *ratelimit.Limiter[uuid.UUID]
	conversations *conversations
}

// New takes a nil client when the assistant is not configured; messages then
// get 503 instead of failing at startup.
func New(store Store, currentUser auth.CurrentUser, client *Client) *Handler {
	return &Handler{
		Store: store, CurrentUser: currentUser, Client: client,
		now:           time.Now,
		limiter:       ratelimit.New[uuid.UUID](messagesPerWindow, rateWindow),
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
	ConversationID uuid.UUID `json:"conversation_id"`
	Reply          string    `json:"reply"`
	Actions        []Action  `json:"actions"`
	Steps          []Step    `json:"steps"`
	// Ended means the conversation cannot take another message.
	Ended bool `json:"ended"`
}

// Message runs one user message through the agent. It may create events
// right away; other changes wait for the user's decision.
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
	start, err := h.conversations.claim(userID, request.ConversationID)
	if err != nil {
		status, message := claimError(err)
		http.Error(w, message, status)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), turnTimeout)
	defer cancel()
	response, result, err := h.answer(ctx, userID, request, start)
	if err != nil {
		// The earlier history is untouched, so the message can be retried.
		h.conversations.unlock(start.id)
		log.Printf("assistant: %v", err)
		http.Error(w, "Couldn't answer right now, try again", http.StatusBadGateway)
		return
	}
	h.conversations.save(start.id, result)
	response.ConversationID = start.id
	writeJSON(w, response)
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

// answer runs the agent on one message and returns what to keep of it.
func (h *Handler) answer(ctx context.Context, userID uuid.UUID, request messageRequest, start claimed) (messageResponse, answered, error) {
	tags, err := h.Store.ListTags(ctx, userID)
	if err != nil {
		return messageResponse{}, answered{}, fmt.Errorf("listing tags: %w", err)
	}
	now := h.now()
	current := newTurn(h.Store, userID, request.location, now, tags)
	message := anthropic.NewBetaUserMessage(
		anthropic.NewBetaTextBlock(contextBlock(now, request, tags, start)),
		anthropic.NewBetaTextBlock(request.Text),
	)

	started := time.Now()
	runner := h.Client.runner(append(start.history, message), current.tools())
	final, err := runner.RunToCompletion(ctx)
	elapsed := time.Since(started).Round(time.Millisecond)
	if err != nil {
		// Events created before the failure stay in the calendar, where the
		// user finds them, but drop out of the conversation.
		return messageResponse{}, answered{}, fmt.Errorf("after %s and %d calls: %w", elapsed, runner.IterationCount(), err)
	}
	log.Printf("assistant: answered in %s with %d calls", elapsed, runner.IterationCount())

	response := messageResponse{Actions: current.actions, Steps: current.steps}
	result := answered{actions: current.actions, notesSeen: len(start.notes)}
	switch final.StopReason {
	case anthropic.BetaStopReasonEndTurn:
		response.Reply = replyText(final)
		result.history = runner.Messages()
		return response, result, nil
	case anthropic.BetaStopReasonRefusal:
		response.Reply = "I can't help with that one."
	default:
		// The iteration limit, which leaves tool calls unanswered, or the
		// output limit; either way the history cannot take another message.
		response.Reply = "I couldn't finish that. Try a more specific request."
	}
	response.Ended, result.ended = true, true
	return response, result, nil
}

// Decide carries out the user's decision on one of the agent's actions:
// apply or decline a pending change, or undo a created event.
func (h *Handler) Decide(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.RequireUser(w, r, h.CurrentUser)
	if !ok {
		return
	}
	conversationID, actionID, decision, err := decodeDecision(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	action, err := h.conversations.decide(userID, conversationID, actionID, decision)
	if err != nil {
		writeDecideError(w, err)
		return
	}
	err = carryOut(r.Context(), h.Store, userID, action, decision, h.now())
	settled := h.conversations.settle(userID, conversationID, actionID, decision, err)
	if err != nil {
		writeDecideError(w, err)
		return
	}
	writeJSON(w, settled)
}

// decodeDecision reads the path ids and the decision from the body.
func decodeDecision(r *http.Request) (conversationID, actionID uuid.UUID, decision string, err error) {
	if conversationID, err = uuid.Parse(r.PathValue("conversationID")); err != nil {
		return uuid.Nil, uuid.Nil, "", errors.New("invalid conversation id")
	}
	if actionID, err = uuid.Parse(r.PathValue("actionID")); err != nil {
		return uuid.Nil, uuid.Nil, "", errors.New("invalid action id")
	}
	var body struct {
		Decision string `json:"decision"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return uuid.Nil, uuid.Nil, "", errors.New("invalid request body")
	}
	if _, _, known := transition(body.Decision); !known {
		return uuid.Nil, uuid.Nil, "", errors.New("decision must be apply, decline or undo")
	}
	return conversationID, actionID, body.Decision, nil
}

func writeDecideError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errConversationNotFound), errors.Is(err, errActionNotFound):
		http.Error(w, "This conversation has ended", http.StatusNotFound)
	case errors.Is(err, errInvalidDecision):
		http.Error(w, "That has already been decided", http.StatusConflict)
	case errors.Is(err, db.ErrEventNotFound), errors.Is(err, db.ErrTimelineItemNotFound):
		http.Error(w, "The event no longer exists", http.StatusNotFound)
	case errors.Is(err, errInvalidChange):
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		log.Printf("assistant: deciding an action: %v", err)
		http.Error(w, "Couldn't do that right now, try again", http.StatusInternalServerError)
	}
}

// contextBlock starts each user message, so the model always knows the
// current time, what the user decided since its last answer, and what still
// waits for them. It sits in the message rather than the system prompt to
// keep the cached prefix stable.
func contextBlock(now time.Time, request messageRequest, tags []db.Tag, start claimed) string {
	var b strings.Builder
	fmt.Fprintf(&b, "<context>\nNow: %s (%s)\nWeeks start on %s.\nTags: %s\n",
		now.In(request.location).Format(timeFormat), request.location, time.Weekday(request.WeekStart), tagList(tags))
	for _, note := range start.notes {
		fmt.Fprintf(&b, "Since your last answer: %s\n", note)
	}
	for _, action := range start.pending {
		fmt.Fprintf(&b, "Still waiting for the user: %s\n", describe(action, request.location))
	}
	b.WriteString("</context>")
	return b.String()
}

func tagList(tags []db.Tag) string {
	if len(tags) == 0 {
		return "(none)"
	}
	names := make([]string, len(tags))
	for i, tag := range tags {
		names[i] = fmt.Sprintf("%q", tag.Name)
	}
	return strings.Join(names, ", ")
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

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}
