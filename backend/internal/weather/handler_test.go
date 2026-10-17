package weather

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jonathan-skantz/calendar/backend/internal/testutil"
)

const upstreamBody = `{"hourly":{"time":["2026-10-17T10:00","2026-10-17T11:00"],"weather_code":[0,0],"is_day":[1,1]}}`

type fakeUpstream struct {
	calls   atomic.Int32
	failing atomic.Bool
	server  *httptest.Server
}

func newFakeUpstream(t *testing.T) *fakeUpstream {
	upstream := &fakeUpstream{}
	upstream.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstream.calls.Add(1)
		if upstream.failing.Load() {
			http.Error(w, "down", http.StatusBadGateway)
			return
		}
		_, _ = w.Write([]byte(upstreamBody))
	}))
	t.Cleanup(upstream.server.Close)
	return upstream
}

func authenticated(*http.Request) (uuid.UUID, error) { return uuid.New(), nil }

func get(handler *Handler, query string) *httptest.ResponseRecorder {
	response := httptest.NewRecorder()
	handler.Forecast(response, httptest.NewRequest(http.MethodGet, "/api/weather?"+query, nil))
	return response
}

func TestForecastValidatesRequest(t *testing.T) {
	unauthenticated := New(nil, func(*http.Request) (uuid.UUID, error) {
		return uuid.Nil, errors.New("not authenticated")
	}, nil)
	if code := get(unauthenticated, "lat=59.3&lon=18.1").Code; code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, code)
	}

	handler := New(nil, authenticated, nil)
	for _, query := range []string{"", "lat=59.3", "lat=abc&lon=18", "lat=91&lon=18", "lat=59&lon=-181", "lat=NaN&lon=18"} {
		if code := get(handler, query).Code; code != http.StatusBadRequest {
			t.Fatalf("query %q: expected status %d, got %d", query, http.StatusBadRequest, code)
		}
	}
}

func TestForecastSharesCacheAndRefreshes(t *testing.T) {
	store := testutil.NewStore(t)
	upstream := newFakeUpstream(t)
	handler := New(store, authenticated, &Client{BaseURL: upstream.server.URL, HTTP: upstream.server.Client()})
	now := time.Date(2026, 10, 17, 8, 0, 0, 0, time.UTC)
	handler.now = func() time.Time { return now }

	// Both positions round to the same 0.1-degree cell.
	for _, query := range []string{"lat=59.33&lon=18.06", "lat=59.31&lon=18.09"} {
		response := get(handler, query)
		if response.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d: %s", http.StatusOK, response.Code, response.Body)
		}
		var days []Day
		if err := json.NewDecoder(response.Body).Decode(&days); err != nil {
			t.Fatal(err)
		}
		if len(days) != 1 || days[0].Date != "2026-10-17" || len(days[0].Conditions) != 1 || days[0].Conditions[0] != Clear {
			t.Fatalf("unexpected days %+v", days)
		}
	}
	if calls := upstream.calls.Load(); calls != 1 {
		t.Fatalf("expected one upstream call for a shared cell, got %d", calls)
	}

	now = now.Add(cacheTTL + time.Minute)
	if code := get(handler, "lat=59.3&lon=18.1").Code; code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, code)
	}
	if calls := upstream.calls.Load(); calls != 2 {
		t.Fatalf("expected a refresh after the TTL, got %d calls", calls)
	}

	now = now.Add(cacheTTL + time.Minute)
	upstream.failing.Store(true)
	if code := get(handler, "lat=59.3&lon=18.1").Code; code != http.StatusOK {
		t.Fatalf("expected the stale forecast when upstream fails, got status %d", code)
	}
	if code := get(handler, "lat=10&lon=10").Code; code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d for an uncached cell, got %d", http.StatusServiceUnavailable, code)
	}
}
