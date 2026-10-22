package assistant

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/google/uuid"
	"github.com/jonathan-skantz/calendar/backend/db"
	"github.com/jonathan-skantz/calendar/backend/internal/ratelimit"
	"github.com/jonathan-skantz/calendar/backend/internal/recurrence"
	"github.com/jonathan-skantz/calendar/backend/internal/testutil"
)

// upstreamMessage is the part of a request's message the tests inspect.
type upstreamMessage struct {
	Role    string `json:"role"`
	Content []struct {
		Type      string `json:"type"`
		Text      string `json:"text"`
		ToolUseID string `json:"tool_use_id"`
		IsError   bool   `json:"is_error"`
		Content   []struct {
			Text string `json:"text"`
		} `json:"content"`
	} `json:"content"`
}

type upstreamRequest struct {
	Model        string            `json:"model"`
	Fallbacks    any               `json:"fallbacks"`
	CacheControl map[string]any    `json:"cache_control"`
	Beta         string            `json:"-"`
	Messages     []upstreamMessage `json:"messages"`
	Tools        []struct {
		Name string `json:"name"`
	} `json:"tools"`
}

// toolResults returns the text of each tool result in the request's last
// message, keyed by tool_use_id.
func (r upstreamRequest) toolResults() map[string]string {
	results := map[string]string{}
	last := r.Messages[len(r.Messages)-1]
	for _, block := range last.Content {
		if block.Type == "tool_result" {
			var text strings.Builder
			for _, part := range block.Content {
				text.WriteString(part.Text)
			}
			results[block.ToolUseID] = text.String()
		}
	}
	return results
}

// fakeAnthropic answers each Messages API call with the next scripted reply.
type fakeAnthropic struct {
	mu       sync.Mutex
	replies  []map[string]any
	requests []upstreamRequest
	failing  bool
	server   *httptest.Server
}

func newFakeAnthropic(t *testing.T) *fakeAnthropic {
	upstream := &fakeAnthropic{}
	upstream.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" || r.Header.Get("x-api-key") != "test-key" {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		var request upstreamRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		upstream.mu.Lock()
		defer upstream.mu.Unlock()
		request.Beta = r.Header.Get("anthropic-beta")
		upstream.requests = append(upstream.requests, request)
		if upstream.failing || len(upstream.replies) == 0 {
			http.Error(w, "overloaded", http.StatusServiceUnavailable)
			return
		}
		reply := upstream.replies[0]
		upstream.replies = upstream.replies[1:]
		reply["id"], reply["type"], reply["role"], reply["model"] = "msg_test", "message", "assistant", DefaultModel
		reply["usage"] = map[string]any{"input_tokens": 1, "output_tokens": 1}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(reply)
	}))
	t.Cleanup(upstream.server.Close)
	return upstream
}

func (f *fakeAnthropic) client() *Client {
	return NewClient("test-key", "", option.WithBaseURL(f.server.URL), option.WithMaxRetries(0))
}

func (f *fakeAnthropic) script(replies ...map[string]any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.replies = append(f.replies, replies...)
}

func (f *fakeAnthropic) request(i int) upstreamRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.requests[i]
}

func (f *fakeAnthropic) calls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.requests)
}

func toolUse(id, name string, input map[string]any) map[string]any {
	return map[string]any{
		"content":     []map[string]any{{"type": "tool_use", "id": id, "name": name, "input": input}},
		"stop_reason": "tool_use",
	}
}

func answer(text string) map[string]any {
	return map[string]any{"content": []map[string]any{{"type": "text", "text": text}}, "stop_reason": "end_turn"}
}

var stockholm = func() *time.Location {
	location, err := time.LoadLocation("Europe/Stockholm")
	if err != nil {
		panic(err)
	}
	return location
}()

// monday is the test's "now": Monday 19 October 2026, 09:00 in Stockholm.
var monday = time.Date(2026, 10, 19, 9, 0, 0, 0, stockholm)

func at(day, hour, minute int) time.Time {
	return time.Date(2026, 10, day, hour, minute, 0, 0, stockholm)
}

// setup returns a store with one registered user, and a handler signed in as
// that user whose clock reads monday.
func setup(t *testing.T) (*db.Store, uuid.UUID, *fakeAnthropic, *Handler) {
	t.Helper()
	store := testutil.NewStore(t)
	userID := register(t, store, "me@example.com")
	upstream := newFakeAnthropic(t)
	handler := New(store, testutil.SignedInAs(userID), upstream.client())
	handler.now = func() time.Time { return monday }
	return store, userID, upstream, handler
}

func register(t *testing.T, store *db.Store, email string) uuid.UUID {
	t.Helper()
	userID, err := store.Register(context.Background(), email, "hash")
	if err != nil {
		t.Fatal(err)
	}
	return userID
}

func seedEvent(t *testing.T, store *db.Store, userID uuid.UUID, event db.Event) uuid.UUID {
	t.Helper()
	if event.Kind == "" {
		event.Kind = db.EventKindAttendable
	}
	id, err := store.CreateEvent(context.Background(), userID, event)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

// storedEvent returns the user's own event id, and false once it is gone.
func storedEvent(t *testing.T, store *db.Store, userID, id uuid.UUID) (db.TimelineEntry, bool) {
	t.Helper()
	entry, err := store.GetOwnEvent(context.Background(), userID, id)
	if errors.Is(err, db.ErrEventNotFound) {
		return db.TimelineEntry{}, false
	}
	if err != nil {
		t.Fatal(err)
	}
	return entry, true
}

func send(handler *Handler, body string) *httptest.ResponseRecorder {
	response := httptest.NewRecorder()
	handler.Message(response, httptest.NewRequest(http.MethodPost, "/api/assistant/messages", strings.NewReader(body)))
	return response
}

func message(conversationID uuid.UUID, text string) string {
	body, _ := json.Marshal(map[string]any{
		"conversation_id": conversationID, "text": text, "time_zone": "Europe/Stockholm", "week_start": 1,
	})
	return string(body)
}

func decide(handler *Handler, conversationID, actionID uuid.UUID, decision string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, "/api/assistant/conversations/x/actions/y",
		strings.NewReader(`{"decision":"`+decision+`"}`))
	request.SetPathValue("conversationID", conversationID.String())
	request.SetPathValue("actionID", actionID.String())
	response := httptest.NewRecorder()
	handler.Decide(response, request)
	return response
}

func decodeOK[T any](t *testing.T, response *httptest.ResponseRecorder) T {
	t.Helper()
	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, response.Code, response.Body)
	}
	var result T
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	return result
}

func TestMessageValidatesRequest(t *testing.T) {
	unauthenticated := New(nil, func(*http.Request) (uuid.UUID, error) {
		return uuid.Nil, errors.New("not authenticated")
	}, nil)
	if code := send(unauthenticated, message(uuid.Nil, "lunch")).Code; code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, code)
	}

	upstream := newFakeAnthropic(t)
	handler := New(nil, testutil.SignedInAs(uuid.New()), upstream.client())
	for _, body := range []string{
		`{`,
		`{"text":"   ","time_zone":"Europe/Stockholm"}`,
		`{"text":"` + strings.Repeat("å", maxTextLength+1) + `","time_zone":"Europe/Stockholm"}`,
		`{"text":"lunch","time_zone":"Mars/Olympus"}`,
		`{"text":"lunch","time_zone":"Local"}`,
		`{"text":"lunch"}`,
		`{"text":"lunch","time_zone":"Europe/Stockholm","week_start":7}`,
	} {
		if code := send(handler, body).Code; code != http.StatusBadRequest {
			t.Fatalf("expected status %d for %s, got %d", http.StatusBadRequest, body, code)
		}
	}
	if calls := upstream.calls(); calls != 0 {
		t.Fatalf("expected no upstream calls for invalid requests, got %d", calls)
	}

	unconfigured := New(nil, testutil.SignedInAs(uuid.New()), nil)
	if code := send(unconfigured, message(uuid.Nil, "lunch")).Code; code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d without a key, got %d", http.StatusServiceUnavailable, code)
	}
	if code := decide(handler, uuid.New(), uuid.New(), "maybe").Code; code != http.StatusBadRequest {
		t.Fatalf("expected status %d for an unknown decision, got %d", http.StatusBadRequest, code)
	}
}

// The example the assistant was built for: Friday is busy, so it checks
// Monday and creates lunch there. The user undoes it, and the agent hears
// about that with the next message.
func TestLunchIsCreatedOnMondayWhenFridayIsBusy(t *testing.T) {
	store, userID, upstream, handler := setup(t)
	ctx := context.Background()
	friends, err := store.CreateTag(ctx, userID, "Friends")
	if err != nil {
		t.Fatal(err)
	}
	seedEvent(t, store, userID, db.Event{Title: "Workshop", StartsAt: at(23, 10, 30), EndsAt: at(23, 14, 30)})
	otherID := register(t, store, "other@example.com")
	seedEvent(t, store, otherID, db.Event{Title: "Secret", StartsAt: at(26, 12, 0), EndsAt: at(26, 13, 0)})

	upstream.script(
		toolUse("toolu_1", "find_free_slots", map[string]any{"date": "2026-10-23", "earliest": "11:00", "latest": "14:00", "duration_minutes": 60}),
		toolUse("toolu_2", "find_free_slots", map[string]any{"date": "2026-10-26", "earliest": "11:00", "latest": "14:00", "duration_minutes": 60}),
		toolUse("toolu_3", "create_event", map[string]any{
			"title": "Lunch with my friend", "date": "2026-10-26", "start_time": "12:00", "end_time": "13:00",
			"tags": []string{"friends", "gym"}, "reminder_minutes": []int{30},
		}),
		answer("Friday is taken by the workshop, so I put lunch on Monday at 12:00."),
	)
	result := decodeOK[messageResponse](t, send(handler, message(uuid.Nil, "I might eat lunch with my friend on Friday. If I'm busy, I'll do it next week on Monday instead")))

	first := upstream.request(0)
	if first.Model != string(DefaultModel) || len(first.Tools) != 6 {
		t.Fatalf("expected the default model with six tools, got %s and %+v", first.Model, first.Tools)
	}
	if first.Fallbacks != "default" || !strings.Contains(first.Beta, "server-side-fallback-2026-07-01") || first.CacheControl["type"] != "ephemeral" {
		t.Fatalf("expected default fallbacks and prompt caching, got %v, %q, %v", first.Fallbacks, first.Beta, first.CacheControl)
	}
	contextText := first.Messages[0].Content[0].Text
	if !strings.Contains(contextText, "Now: Mon 2026-10-19 09:00 (Europe/Stockholm)") ||
		!strings.Contains(contextText, "Weeks start on Monday") || !strings.Contains(contextText, `"Friends"`) {
		t.Fatalf("unexpected context block %q", contextText)
	}
	friday := upstream.request(1).toolResults()["toolu_1"]
	if !strings.Contains(friday, `"free":[]`) || !strings.Contains(friday, "Workshop") {
		t.Fatalf("expected Friday to come back busy, got %s", friday)
	}
	mondayResult := upstream.request(2).toolResults()["toolu_2"]
	if !strings.Contains(mondayResult, `"from":"Mon 2026-10-26 11:00","to":"Mon 2026-10-26 14:00"`) || strings.Contains(mondayResult, "Secret") {
		t.Fatalf("expected all of Monday free and no other user's events, got %s", mondayResult)
	}

	if len(result.Actions) != 1 || len(result.Steps) != 3 || result.Ended {
		t.Fatalf("expected one action after three steps, got %+v", result)
	}
	created := result.Actions[0]
	// Monday is after the switch from summer time, so the offset is +01:00.
	if created.Kind != kindCreate || created.Status != statusDone || created.Title != "Lunch with my friend" ||
		!created.StartsAt.Equal(time.Date(2026, 10, 26, 11, 0, 0, 0, time.UTC)) ||
		len(created.Warnings) != 1 || created.Warnings[0] != `No tag named "gym"` {
		t.Fatalf("unexpected action %+v", created)
	}
	entry, ok := storedEvent(t, store, userID, created.EventID)
	if !ok || *entry.MainTagID != friends.ID || !entry.EndsAt.Equal(at(26, 13, 0)) {
		t.Fatalf("expected the lunch saved with the Friends tag, got %+v", entry)
	}
	reminders, err := store.ListReminders(ctx, userID)
	if err != nil || len(reminders) != 1 || reminders[0].OffsetSeconds != 1800 {
		t.Fatalf("expected one reminder 30 minutes before, got %+v, %v", reminders, err)
	}

	// Only the owner of the conversation can decide, and only once.
	handler.CurrentUser = testutil.SignedInAs(otherID)
	if code := decide(handler, result.ConversationID, created.ID, decisionUndo).Code; code != http.StatusNotFound {
		t.Fatalf("expected status %d for another user, got %d", http.StatusNotFound, code)
	}
	handler.CurrentUser = testutil.SignedInAs(userID)
	if undone := decodeOK[Action](t, decide(handler, result.ConversationID, created.ID, decisionUndo)); undone.Status != statusUndone {
		t.Fatalf("expected the action undone, got %+v", undone)
	}
	if _, ok := storedEvent(t, store, userID, created.EventID); ok {
		t.Fatal("expected undo to delete the event")
	}
	if code := decide(handler, result.ConversationID, created.ID, decisionUndo).Code; code != http.StatusConflict {
		t.Fatalf("expected status %d for a second undo, got %d", http.StatusConflict, code)
	}

	upstream.script(answer("Okay, I removed it."))
	decodeOK[messageResponse](t, send(handler, message(result.ConversationID, "Never mind")))
	// The first message, three tool rounds, the answer, and the follow-up.
	history := upstream.request(4).Messages
	if len(history) != 9 || history[8].Content[1].Text != "Never mind" ||
		!strings.Contains(history[8].Content[0].Text, `The user's undo of create "Lunch with my friend": undone.`) {
		t.Fatalf("expected the follow-up with the undo note after the whole history, got %+v", history[len(history)-1])
	}
}

func TestChangesWaitForTheUser(t *testing.T) {
	store, userID, upstream, handler := setup(t)
	dentist := seedEvent(t, store, userID, db.Event{
		Title: "Dentist", StartsAt: at(22, 10, 0), EndsAt: at(22, 11, 0), CommuteBeforeSeconds: 900,
	})
	lecture := seedEvent(t, store, userID, db.Event{Title: "Lecture", StartsAt: at(22, 13, 0), EndsAt: at(22, 15, 0)})

	upstream.script(
		toolUse("toolu_1", "move_event", map[string]any{"event_id": dentist, "date": "2026-10-23"}),
		toolUse("toolu_2", "skip_event", map[string]any{"event_id": lecture, "skipped": true}),
		answer("Both are waiting for you."),
	)
	result := decodeOK[messageResponse](t, send(handler, message(uuid.Nil, "Move the dentist to Friday and skip the lecture")))
	if len(result.Actions) != 2 || result.Actions[0].Status != statusPending || result.Actions[1].Kind != kindSkip {
		t.Fatalf("expected two pending actions, got %+v", result.Actions)
	}
	move, skip := result.Actions[0], result.Actions[1]
	if entry, _ := storedEvent(t, store, userID, dentist); !entry.StartsAt.Equal(at(22, 10, 0)) {
		t.Fatalf("expected nothing to change before the user decides, got %+v", entry)
	}

	// The move keeps the time of day, the duration and everything else.
	if applied := decodeOK[Action](t, decide(handler, result.ConversationID, move.ID, decisionApply)); applied.Status != statusApplied {
		t.Fatalf("expected the move applied, got %+v", applied)
	}
	entry, _ := storedEvent(t, store, userID, dentist)
	if !entry.StartsAt.Equal(at(23, 10, 0)) || !entry.EndsAt.Equal(at(23, 11, 0)) || entry.CommuteBeforeSeconds != 900 {
		t.Fatalf("expected the dentist moved to Friday with its commute, got %+v", entry)
	}
	if code := decide(handler, result.ConversationID, move.ID, decisionApply).Code; code != http.StatusConflict {
		t.Fatalf("expected status %d when applying twice, got %d", http.StatusConflict, code)
	}
	if code := decide(handler, result.ConversationID, move.ID, decisionUndo).Code; code != http.StatusConflict {
		t.Fatalf("expected status %d when undoing a move, got %d", http.StatusConflict, code)
	}

	if declined := decodeOK[Action](t, decide(handler, result.ConversationID, skip.ID, decisionDecline)); declined.Status != statusDeclined {
		t.Fatalf("expected the skip declined, got %+v", declined)
	}
	if entry, _ := storedEvent(t, store, userID, lecture); entry.IsSkipped {
		t.Fatal("expected a declined skip to change nothing")
	}
}

func TestApplyingAfterTheEventIsGoneKeepsTheActionOpen(t *testing.T) {
	store, userID, upstream, handler := setup(t)
	dentist := seedEvent(t, store, userID, db.Event{Title: "Dentist", StartsAt: at(22, 10, 0), EndsAt: at(22, 11, 0)})
	upstream.script(
		toolUse("toolu_1", "delete_event", map[string]any{"event_id": dentist}),
		answer("Waiting for you."),
	)
	result := decodeOK[messageResponse](t, send(handler, message(uuid.Nil, "Cancel the dentist")))
	if err := store.DeleteTimelineItem(context.Background(), userID, dentist); err != nil {
		t.Fatal(err)
	}

	action := result.Actions[0]
	if code := decide(handler, result.ConversationID, action.ID, decisionApply).Code; code != http.StatusNotFound {
		t.Fatalf("expected status %d for a deleted event, got %d", http.StatusNotFound, code)
	}
	if declined := decodeOK[Action](t, decide(handler, result.ConversationID, action.ID, decisionDecline)); declined.Status != statusDeclined {
		t.Fatalf("expected the failed action to stay open, got %+v", declined)
	}
}

func TestToolErrorsGoBackToTheModel(t *testing.T) {
	store, userID, upstream, handler := setup(t)
	headsUp := seedEvent(t, store, userID, db.Event{
		Title: "Phone release", StartsAt: at(22, 10, 0), EndsAt: at(22, 11, 0), Kind: db.EventKindInformational,
	})
	ownerID := register(t, store, "owner@example.com")
	shared := seedEvent(t, store, ownerID, db.Event{Title: "Party", StartsAt: at(24, 18, 0), EndsAt: at(24, 22, 0)})
	if _, err := store.ShareEvent(context.Background(), ownerID, shared, userID, db.ShareFields{}); err != nil {
		t.Fatal(err)
	}

	upstream.script(
		toolUse("toolu_1", "create_event", map[string]any{"title": "Lunch", "date": "2026-10-16", "start_time": "12:00"}),
		toolUse("toolu_2", "skip_event", map[string]any{"event_id": headsUp, "skipped": true}),
		toolUse("toolu_3", "delete_event", map[string]any{"event_id": shared}),
		answer("None of that worked."),
	)
	result := decodeOK[messageResponse](t, send(handler, message(uuid.Nil, "do things")))
	if len(result.Actions) != 0 || len(result.Steps) != 0 {
		t.Fatalf("expected no actions, got %+v", result)
	}
	for i, want := range []string{"start_time has already passed", "only attendable events can be skipped", "cannot be changed"} {
		request := upstream.request(i + 1)
		block := request.Messages[len(request.Messages)-1].Content[0]
		if !block.IsError || !strings.Contains(block.Content[0].Text, want) {
			t.Fatalf("expected %q as the tool error, got %+v", want, block)
		}
	}
}

func TestStepLimitEndsMessagesButNotDecisions(t *testing.T) {
	store, userID, upstream, handler := setup(t)
	upstream.script(toolUse("toolu_0", "create_event", map[string]any{"title": "Gym", "date": "2026-10-20", "start_time": "18:00"}))
	for i := 1; i < maxIterations; i++ {
		upstream.script(toolUse(fmt.Sprintf("toolu_%d", i), "list_events", map[string]any{"from_date": "2026-10-19", "to_date": "2026-10-25"}))
	}
	result := decodeOK[messageResponse](t, send(handler, message(uuid.Nil, "plan my week")))
	if !result.Ended || upstream.calls() != maxIterations || len(result.Actions) != 1 {
		t.Fatalf("expected the conversation to end after %d calls with one action, got %+v", maxIterations, result)
	}
	if code := send(handler, message(result.ConversationID, "go on")).Code; code != http.StatusNotFound {
		t.Fatalf("expected status %d for a message after the end, got %d", http.StatusNotFound, code)
	}
	decodeOK[Action](t, decide(handler, result.ConversationID, result.Actions[0].ID, decisionUndo))
	if _, ok := storedEvent(t, store, userID, result.Actions[0].EventID); ok {
		t.Fatal("expected the created event to be undone")
	}
}

func TestUpstreamFailureKeepsTheConversation(t *testing.T) {
	_, _, upstream, handler := setup(t)
	upstream.script(answer("Hi!"))
	result := decodeOK[messageResponse](t, send(handler, message(uuid.Nil, "hello")))

	upstream.mu.Lock()
	upstream.failing = true
	upstream.mu.Unlock()
	if code := send(handler, message(result.ConversationID, "again")).Code; code != http.StatusBadGateway {
		t.Fatalf("expected status %d, got %d", http.StatusBadGateway, code)
	}

	upstream.mu.Lock()
	upstream.failing = false
	upstream.mu.Unlock()
	upstream.script(answer("Still here."))
	decodeOK[messageResponse](t, send(handler, message(result.ConversationID, "again")))
	// The retry sends the first exchange and the new message, not the failed one.
	if history := upstream.request(2).Messages; len(history) != 3 {
		t.Fatalf("expected 3 messages on retry, got %d", len(history))
	}
}

func TestMessagesAreRateLimitedPerUser(t *testing.T) {
	_, _, upstream, handler := setup(t)
	handler.limiter = ratelimit.New(1, time.Hour)
	upstream.script(answer("Hi!"))
	decodeOK[messageResponse](t, send(handler, message(uuid.Nil, "hello")))
	if code := send(handler, message(uuid.Nil, "hello")).Code; code != http.StatusTooManyRequests {
		t.Fatalf("expected status %d over the limit, got %d", http.StatusTooManyRequests, code)
	}
}

func TestLaterToolsSeeCreatedEvents(t *testing.T) {
	_, _, upstream, handler := setup(t)
	upstream.script(
		toolUse("toolu_1", "find_free_slots", map[string]any{"date": "2026-10-20", "earliest": "17:00", "latest": "19:00", "duration_minutes": 60}),
		toolUse("toolu_2", "create_event", map[string]any{"title": "Gym", "date": "2026-10-20", "start_time": "17:00", "end_time": "18:00"}),
		toolUse("toolu_3", "find_free_slots", map[string]any{"date": "2026-10-20", "earliest": "17:00", "latest": "19:00", "duration_minutes": 60}),
		answer("Done."),
	)
	decodeOK[messageResponse](t, send(handler, message(uuid.Nil, "gym tomorrow, then dinner after")))
	if after := upstream.request(3).toolResults()["toolu_3"]; !strings.Contains(after, `"title":"Gym"`) {
		t.Fatalf("expected the new gym session to block the time, got %s", after)
	}
}

func TestMovesCannotBeAppliedOnceTheirTimeHasPassed(t *testing.T) {
	store, userID, upstream, handler := setup(t)
	dentist := seedEvent(t, store, userID, db.Event{Title: "Dentist", StartsAt: at(22, 10, 0), EndsAt: at(22, 11, 0)})
	upstream.script(
		toolUse("toolu_1", "move_event", map[string]any{"event_id": dentist, "date": "2026-10-19", "start_time": "15:00"}),
		answer("Waiting for you."),
	)
	result := decodeOK[messageResponse](t, send(handler, message(uuid.Nil, "dentist this afternoon instead")))

	handler.now = func() time.Time { return at(19, 16, 0) }
	response := decide(handler, result.ConversationID, result.Actions[0].ID, decisionApply)
	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "already passed") {
		t.Fatalf("expected status %d for a move into the past, got %d: %s", http.StatusBadRequest, response.Code, response.Body)
	}
	if entry, _ := storedEvent(t, store, userID, dentist); !entry.StartsAt.Equal(at(22, 10, 0)) {
		t.Fatalf("expected the dentist to stay, got %+v", entry)
	}
}

func TestRepeatingEventsAreSeenButNotChanged(t *testing.T) {
	store, userID, upstream, handler := setup(t)
	start := at(13, 10, 0)
	rule := recurrence.Rule{Frequency: recurrence.Weekly, Interval: 1, Weekdays: []string{"TU", "TH"}}.RRULE(start, false)
	lecture := seedEvent(t, store, userID, db.Event{
		Title: "Lecture", StartsAt: start, EndsAt: start.Add(2 * time.Hour),
		RecurrenceRule: &rule, RecurrenceTimeZone: new("Europe/Stockholm"),
	})
	upstream.script(
		toolUse("toolu_1", "list_events", map[string]any{"from_date": "2026-10-19", "to_date": "2026-10-25"}),
		toolUse("toolu_2", "find_free_slots", map[string]any{"date": "2026-10-22", "earliest": "08:00", "latest": "13:00", "duration_minutes": 60}),
		toolUse("toolu_3", "move_event", map[string]any{"event_id": lecture, "date": "2026-10-23"}),
		answer("The lecture repeats; move it in the app."),
	)
	result := decodeOK[messageResponse](t, send(handler, message(uuid.Nil, "move Thursday's lecture to Friday")))

	listed := upstream.request(1).toolResults()["toolu_1"]
	if strings.Count(listed, `"title":"Lecture"`) != 2 || !strings.Contains(listed, `"repeats":true`) ||
		!strings.Contains(listed, `"start":"Tue 2026-10-20 10:00"`) || !strings.Contains(listed, `"start":"Thu 2026-10-22 10:00"`) {
		t.Fatalf("expected both occurrences of the week, got %s", listed)
	}
	if free := upstream.request(2).toolResults()["toolu_2"]; !strings.Contains(free, `"title":"Lecture"`) {
		t.Fatalf("expected the occurrence to block the morning, got %s", free)
	}
	if moved := upstream.request(3).toolResults()["toolu_3"]; !strings.Contains(moved, errRepeating.Error()) {
		t.Fatalf("expected the move to be refused, got %s", moved)
	}
	if len(result.Actions) != 0 {
		t.Fatalf("expected no actions, got %+v", result.Actions)
	}
}

func TestSubscribedEventsAreSeenButNotChanged(t *testing.T) {
	store, userID, upstream, handler := setup(t)
	start := at(22, 10, 0)
	subscription, err := store.CreateSubscription(context.Background(), db.Subscription{
		UserID: userID, Name: "School", URL: "https://example.com/school.ics", Color: "#336699", TimeZone: "Europe/Stockholm",
	}, []db.SubscribedEvent{{Event: db.Event{
		Title: "Exam", StartsAt: start, EndsAt: start.Add(2 * time.Hour), Kind: db.EventKindAttendable,
	}}})
	if err != nil {
		t.Fatal(err)
	}
	entries, err := store.ListTimeline(context.Background(), userID, at(22, 0, 0), at(23, 0, 0))
	if err != nil || len(entries) != 1 || entries[0].Subscription == nil || entries[0].Subscription.ID != subscription.ID {
		t.Fatalf("expected the subscribed exam, got %+v, %v", entries, err)
	}
	upstream.script(
		toolUse("toolu_1", "list_events", map[string]any{"from_date": "2026-10-22", "to_date": "2026-10-22"}),
		toolUse("toolu_2", "delete_event", map[string]any{"event_id": entries[0].ID}),
		answer("The exam comes from a subscribed calendar."),
	)
	result := decodeOK[messageResponse](t, send(handler, message(uuid.Nil, "delete the exam")))

	if listed := upstream.request(1).toolResults()["toolu_1"]; !strings.Contains(listed, `"calendar":"School"`) {
		t.Fatalf("expected the exam marked with its calendar, got %s", listed)
	}
	if deleted := upstream.request(2).toolResults()["toolu_2"]; !strings.Contains(deleted, errNotOwnEvent.Error()) {
		t.Fatalf("expected the delete to be refused, got %s", deleted)
	}
	if len(result.Actions) != 0 {
		t.Fatalf("expected no actions, got %+v", result.Actions)
	}
}
