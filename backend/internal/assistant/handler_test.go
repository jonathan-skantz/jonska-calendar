package assistant

import (
	"context"
	"encoding/json"
	"errors"
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

type fakeStore struct {
	entries map[uuid.UUID][]db.TimelineEntry
	tags    map[uuid.UUID][]db.Tag
}

func (f fakeStore) ListTimeline(_ context.Context, userID uuid.UUID) ([]db.TimelineEntry, error) {
	return f.entries[userID], nil
}

func (f fakeStore) ListTags(_ context.Context, userID uuid.UUID) ([]db.Tag, error) {
	return f.tags[userID], nil
}

var stockholm = func() *time.Location {
	location, err := time.LoadLocation("Europe/Stockholm")
	if err != nil {
		panic(err)
	}
	return location
}()

func attended(title string, start, end time.Time) db.TimelineEntry {
	kind := db.EventKindAttendable
	return db.TimelineEntry{ID: uuid.New(), Type: "event", Title: title, Kind: &kind, StartsAt: &start, EndsAt: &end}
}

func send(handler *Handler, body string) *httptest.ResponseRecorder {
	response := httptest.NewRecorder()
	handler.Message(response, httptest.NewRequest(http.MethodPost, "/api/assistant/messages", strings.NewReader(body)))
	return response
}

func decode(t *testing.T, response *httptest.ResponseRecorder) messageResponse {
	t.Helper()
	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, response.Code, response.Body)
	}
	var result messageResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	return result
}

func message(conversationID uuid.UUID, text string) string {
	body, _ := json.Marshal(map[string]any{
		"conversation_id": conversationID, "text": text, "time_zone": "Europe/Stockholm", "week_start": 1,
	})
	return string(body)
}

func TestMessageValidatesRequest(t *testing.T) {
	unauthenticated := New(fakeStore{}, func(*http.Request) (uuid.UUID, error) {
		return uuid.Nil, errors.New("not authenticated")
	}, nil)
	if code := send(unauthenticated, message(uuid.Nil, "lunch")).Code; code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, code)
	}

	upstream := newFakeAnthropic(t)
	handler := New(fakeStore{}, testutil.SignedInAs(uuid.New()), upstream.client())
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

	unconfigured := New(fakeStore{}, testutil.SignedInAs(uuid.New()), nil)
	if code := send(unconfigured, message(uuid.Nil, "lunch")).Code; code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d without a key, got %d", http.StatusServiceUnavailable, code)
	}
}

// The example the assistant was built for: Friday is busy, so it checks
// Monday and proposes lunch there, then takes a follow-up in the same
// conversation.
func TestLunchFallsBackToMondayWhenFridayIsBusy(t *testing.T) {
	upstream := newFakeAnthropic(t)
	userID, otherID, friendsID := uuid.New(), uuid.New(), uuid.New()
	store := fakeStore{
		entries: map[uuid.UUID][]db.TimelineEntry{
			userID: {attended("Workshop",
				time.Date(2026, 10, 23, 10, 30, 0, 0, stockholm), time.Date(2026, 10, 23, 14, 30, 0, 0, stockholm))},
			otherID: {attended("Secret", time.Date(2026, 10, 26, 12, 0, 0, 0, stockholm), time.Date(2026, 10, 26, 13, 0, 0, 0, stockholm))},
		},
		tags: map[uuid.UUID][]db.Tag{userID: {{ID: friendsID, Name: "Friends"}}},
	}
	handler := New(store, testutil.SignedInAs(userID), upstream.client())
	handler.now = func() time.Time { return time.Date(2026, 10, 19, 9, 0, 0, 0, stockholm) }

	upstream.script(
		toolUse("toolu_1", "find_free_slots", map[string]any{"date": "2026-10-23", "earliest": "11:00", "latest": "14:00", "duration_minutes": 60}),
		toolUse("toolu_2", "find_free_slots", map[string]any{"date": "2026-10-26", "earliest": "11:00", "latest": "14:00", "duration_minutes": 60}),
		toolUse("toolu_3", "propose_event", map[string]any{
			"title": "Lunch with my friend", "date": "2026-10-26", "start_time": "12:00", "end_time": "13:00", "all_day": false,
			"location": nil, "kind": "attendable", "tags": []string{"friends", "gym"}, "reminder_minutes": []int{30},
		}),
		answer("Friday is taken by the workshop, so I suggested Monday at 12:00."),
	)
	result := decode(t, send(handler, message(uuid.Nil, "I might eat lunch with my friend on Friday. If I'm busy, I'll do it next week on Monday instead")))

	first := upstream.request(0)
	if first.Model != string(DefaultModel) || len(first.Tools) != 3 {
		t.Fatalf("expected the default model with three tools, got %s and %+v", first.Model, first.Tools)
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
	monday := upstream.request(2).toolResults()["toolu_2"]
	if !strings.Contains(monday, `"from":"Mon 2026-10-26 11:00","to":"Mon 2026-10-26 14:00"`) || strings.Contains(monday, "Secret") {
		t.Fatalf("expected all of Monday free and no other user's events, got %s", monday)
	}

	if result.Reply != "Friday is taken by the workshop, so I suggested Monday at 12:00." || result.Ended {
		t.Fatalf("unexpected reply %+v", result)
	}
	if len(result.Steps) != 3 || result.Steps[2].Tool != "propose_event" {
		t.Fatalf("expected three steps ending in a proposal, got %+v", result.Steps)
	}
	if len(result.Proposals) != 1 {
		t.Fatalf("expected one proposal, got %+v", result.Proposals)
	}
	proposal := result.Proposals[0]
	// Monday is after the switch from summer time, so the offset is +01:00.
	if proposal.Title != "Lunch with my friend" ||
		!proposal.StartsAt.Equal(time.Date(2026, 10, 26, 11, 0, 0, 0, time.UTC)) ||
		!proposal.EndsAt.Equal(time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)) ||
		len(proposal.TagIDs) != 1 || proposal.TagIDs[0] != friendsID ||
		len(proposal.ReminderMinutes) != 1 || proposal.ReminderMinutes[0] != 30 ||
		len(proposal.Warnings) != 1 || proposal.Warnings[0] != `No tag named "gym"` {
		t.Fatalf("unexpected proposal %+v", proposal)
	}

	// A follow-up continues the same history.
	upstream.script(answer("Sure, any time that day works."))
	followUp := decode(t, send(handler, message(result.ConversationID, "Could it be later?")))
	if followUp.ConversationID != result.ConversationID || len(followUp.Proposals) != 0 {
		t.Fatalf("unexpected follow-up %+v", followUp)
	}
	// The first message, three tool rounds, the answer, and the follow-up.
	if history := upstream.request(4).Messages; len(history) != 9 || history[8].Content[1].Text != "Could it be later?" {
		t.Fatalf("expected the follow-up after the whole history, got %d messages", len(history))
	}

	// The conversation belongs to the user who started it.
	handler.CurrentUser = testutil.SignedInAs(otherID)
	if code := send(handler, message(result.ConversationID, "hi")).Code; code != http.StatusNotFound {
		t.Fatalf("expected status %d for another user's conversation, got %d", http.StatusNotFound, code)
	}
}

func TestToolErrorsGoBackToTheModel(t *testing.T) {
	upstream := newFakeAnthropic(t)
	handler := New(fakeStore{}, testutil.SignedInAs(uuid.New()), upstream.client())
	handler.now = func() time.Time { return time.Date(2026, 10, 19, 9, 0, 0, 0, stockholm) }
	upstream.script(
		toolUse("toolu_1", "propose_event", map[string]any{
			"title": "Lunch", "date": "2026-10-16", "start_time": "12:00", "end_time": nil, "all_day": false,
			"location": nil, "kind": nil, "tags": []string{}, "reminder_minutes": []int{},
		}),
		answer("That day has passed."),
	)
	result := decode(t, send(handler, message(uuid.Nil, "lunch last Friday")))
	if len(result.Proposals) != 0 || len(result.Steps) != 0 {
		t.Fatalf("expected no proposal for a past date, got %+v", result)
	}
	request := upstream.request(1)
	block := request.Messages[len(request.Messages)-1].Content[0]
	if !block.IsError || !strings.Contains(block.Content[0].Text, "start_time has already passed") {
		t.Fatalf("expected the validation error as the tool result, got %+v", block)
	}
}

func TestStepLimitEndsTheConversation(t *testing.T) {
	upstream := newFakeAnthropic(t)
	handler := New(fakeStore{}, testutil.SignedInAs(uuid.New()), upstream.client())
	for i := range maxIterations {
		upstream.script(toolUse("toolu_"+string(rune('a'+i)), "list_events", map[string]any{"from_date": "2026-10-19", "to_date": "2026-10-25"}))
	}
	result := decode(t, send(handler, message(uuid.Nil, "plan my week")))
	if !result.Ended || upstream.calls() != maxIterations {
		t.Fatalf("expected the conversation to end after %d calls, got %+v after %d", maxIterations, result, upstream.calls())
	}
	if code := send(handler, message(result.ConversationID, "go on")).Code; code != http.StatusNotFound {
		t.Fatalf("expected status %d after the conversation ended, got %d", http.StatusNotFound, code)
	}
}

func TestUpstreamFailureKeepsTheConversation(t *testing.T) {
	upstream := newFakeAnthropic(t)
	handler := New(fakeStore{}, testutil.SignedInAs(uuid.New()), upstream.client())
	upstream.script(answer("Hi!"))
	result := decode(t, send(handler, message(uuid.Nil, "hello")))

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
	decode(t, send(handler, message(result.ConversationID, "again")))
	// The retry sends the first exchange and the new message, not the failed one.
	if history := upstream.request(2).Messages; len(history) != 3 {
		t.Fatalf("expected 3 messages on retry, got %d", len(history))
	}
}

func TestMessagesAreRateLimitedPerUser(t *testing.T) {
	upstream := newFakeAnthropic(t)
	handler := New(fakeStore{}, testutil.SignedInAs(uuid.New()), upstream.client())
	handler.limiter = ratelimit.New(1, time.Hour)
	upstream.script(answer("Hi!"))
	decode(t, send(handler, message(uuid.Nil, "hello")))
	if code := send(handler, message(uuid.Nil, "hello")).Code; code != http.StatusTooManyRequests {
		t.Fatalf("expected status %d over the limit, got %d", http.StatusTooManyRequests, code)
	}
}
