package quickadd

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/google/uuid"
	"github.com/jonathan-skantz/calendar/backend/db"
	"github.com/jonathan-skantz/calendar/backend/internal/auth"
)

const upstreamDraft = `{"title":"Lunch with Anna","date":{"kind":"relative_days","days":1},` +
	`"start_time":"12:00","end_time":"13:00","tags":["work","gym"]}`

// upstreamRequest is the part of a Messages API request the tests inspect.
type upstreamRequest struct {
	System []struct {
		Text string `json:"text"`
	} `json:"system"`
	Messages []struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	} `json:"messages"`
	OutputConfig struct {
		Format struct {
			Type   string         `json:"type"`
			Schema map[string]any `json:"schema"`
		} `json:"format"`
	} `json:"output_config"`
	Temperature *float64 `json:"temperature"`
}

type fakeAnthropic struct {
	calls   atomic.Int32
	failing atomic.Bool
	last    atomic.Pointer[upstreamRequest]
	server  *httptest.Server
}

func newFakeAnthropic(t *testing.T) *fakeAnthropic {
	upstream := &fakeAnthropic{}
	upstream.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstream.calls.Add(1)
		if r.URL.Path != "/v1/messages" || r.Header.Get("x-api-key") != "test-key" {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		var request upstreamRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		upstream.last.Store(&request)
		if upstream.failing.Load() {
			http.Error(w, "overloaded", http.StatusServiceUnavailable)
			return
		}
		reply, _ := json.Marshal(map[string]any{
			"id": "msg_test", "type": "message", "role": "assistant", "model": DefaultModel,
			"content":     []map[string]any{{"type": "text", "text": upstreamDraft}},
			"stop_reason": "end_turn",
			"usage":       map[string]any{"input_tokens": 1, "output_tokens": 1},
		})
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(reply)
	}))
	t.Cleanup(upstream.server.Close)
	return upstream
}

func (f *fakeAnthropic) client() *Client {
	return NewClient("test-key", "", option.WithBaseURL(f.server.URL), option.WithMaxRetries(0))
}

type fakeTags struct {
	byUser map[uuid.UUID][]db.Tag
}

func (f fakeTags) ListTags(_ context.Context, userID uuid.UUID) ([]db.Tag, error) {
	return f.byUser[userID], nil
}

func userFunc(id uuid.UUID) auth.CurrentUser {
	return func(*http.Request) (uuid.UUID, error) { return id, nil }
}

func post(handler *Handler, body string) *httptest.ResponseRecorder {
	response := httptest.NewRecorder()
	handler.Parse(response, httptest.NewRequest(http.MethodPost, "/api/events/parse", strings.NewReader(body)))
	return response
}

func TestParseValidatesRequest(t *testing.T) {
	unauthenticated := New(nil, func(*http.Request) (uuid.UUID, error) {
		return uuid.Nil, errors.New("not authenticated")
	}, nil)
	if code := post(unauthenticated, `{"text":"lunch"}`).Code; code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, code)
	}

	upstream := newFakeAnthropic(t)
	handler := New(fakeTags{}, userFunc(uuid.New()), upstream.client())
	for _, body := range []string{`{`, `{"text":"   "}`, `{"text":"` + strings.Repeat("å", maxTextLength+1) + `"}`} {
		if code := post(handler, body).Code; code != http.StatusBadRequest {
			t.Fatalf("expected status %d, got %d", http.StatusBadRequest, code)
		}
	}
	if calls := upstream.calls.Load(); calls != 0 {
		t.Fatalf("expected no upstream calls for invalid requests, got %d", calls)
	}

	unconfigured := New(fakeTags{}, userFunc(uuid.New()), nil)
	if code := post(unconfigured, `{"text":"lunch"}`).Code; code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d without a key, got %d", http.StatusServiceUnavailable, code)
	}
}

func TestParseReturnsValidatedDraftWithOwnTags(t *testing.T) {
	upstream := newFakeAnthropic(t)
	userID, otherID, workID := uuid.New(), uuid.New(), uuid.New()
	tags := fakeTags{byUser: map[uuid.UUID][]db.Tag{
		userID:  {{ID: workID, Name: "Work"}},
		otherID: {{ID: uuid.New(), Name: "Secret"}},
	}}
	handler := New(tags, userFunc(userID), upstream.client())

	response := post(handler, `{"text":"lunch with Anna tomorrow 12-13 #work #gym"}`)
	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, response.Code, response.Body)
	}
	var result ParseResult
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if *result.Draft.Title != "Lunch with Anna" || *result.Draft.Date.Days != 1 || *result.Draft.StartTime != "12:00" ||
		len(result.Draft.TagIDs) != 1 || result.Draft.TagIDs[0] != workID {
		t.Fatalf("unexpected draft %+v", result.Draft)
	}
	if len(result.Warnings) != 1 || result.Warnings[0] != `No tag named "gym"` {
		t.Fatalf("unexpected warnings %v", result.Warnings)
	}

	request := upstream.last.Load()
	if request.OutputConfig.Format.Type != "json_schema" || request.OutputConfig.Format.Schema["type"] != "object" {
		t.Fatalf("expected a structured output schema, got %+v", request.OutputConfig)
	}
	if request.Temperature != nil {
		t.Fatalf("expected no temperature, got %v", *request.Temperature)
	}
	system := request.System[0].Text
	if !strings.Contains(system, `"Work"`) || strings.Contains(system, "Secret") {
		t.Fatalf("expected only the caller's tags in the prompt, got %q", system)
	}
	if request.Messages[0].Content[0].Text != "lunch with Anna tomorrow 12-13 #work #gym" || strings.Contains(system, "Anna") {
		t.Fatalf("expected the text only in the user message, got %+v", request.Messages)
	}
}

func TestParseRateLimitsPerUser(t *testing.T) {
	upstream := newFakeAnthropic(t)
	userID := uuid.New()
	handler := New(fakeTags{}, userFunc(userID), upstream.client())
	now := time.Date(2026, 10, 17, 8, 0, 0, 0, time.UTC)
	handler.now = func() time.Time { return now }

	for range parsesPerWindow {
		if code := post(handler, `{"text":"lunch"}`).Code; code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, code)
		}
	}
	if code := post(handler, `{"text":"lunch"}`).Code; code != http.StatusTooManyRequests {
		t.Fatalf("expected status %d over the limit, got %d", http.StatusTooManyRequests, code)
	}
	handler.CurrentUser = userFunc(uuid.New())
	if code := post(handler, `{"text":"lunch"}`).Code; code != http.StatusOK {
		t.Fatalf("expected another user to have their own budget, got %d", code)
	}
	handler.CurrentUser = userFunc(userID)
	now = now.Add(rateWindow)
	if code := post(handler, `{"text":"lunch"}`).Code; code != http.StatusOK {
		t.Fatalf("expected the budget to reset after the window, got %d", code)
	}
}

func TestParseReportsUpstreamFailure(t *testing.T) {
	upstream := newFakeAnthropic(t)
	upstream.failing.Store(true)
	handler := New(fakeTags{}, userFunc(uuid.New()), upstream.client())
	if code := post(handler, `{"text":"lunch"}`).Code; code != http.StatusBadGateway {
		t.Fatalf("expected status %d, got %d", http.StatusBadGateway, code)
	}
}
