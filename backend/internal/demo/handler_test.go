package demo

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/jonathan-skantz/calendar/backend/db"
	"github.com/jonathan-skantz/calendar/backend/internal/auth"
	"github.com/jonathan-skantz/calendar/backend/internal/testutil"
	"github.com/jonathan-skantz/calendar/backend/internal/timeline"
)

// startDemo starts a demo in Stockholm and returns the new user's ID and
// session cookie.
func startDemo(t *testing.T, handler *Handler) (uuid.UUID, *http.Cookie) {
	t.Helper()
	response := httptest.NewRecorder()
	handler.Start(response, httptest.NewRequest(http.MethodPost, "/api/auth/demo", strings.NewReader(`{"time_zone":"Europe/Stockholm"}`)))
	if response.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, response.Code, response.Body.String())
	}
	var started startResponse
	if err := json.NewDecoder(response.Body).Decode(&started); err != nil {
		t.Fatal(err)
	}
	cookies := response.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatal("expected a session cookie")
	}
	return started.ID, cookies[0]
}

func TestStartSignsInToASeededDemoAccount(t *testing.T) {
	store := testutil.NewStore(t)
	authHandler := auth.New(store, auth.Config{})
	handler := New(store, authHandler)
	now := time.Now()
	handler.Now = func() time.Time { return now }
	userID, cookie := startDemo(t, handler)

	if !cookie.Expires.Equal(now.Add(lifetime).Truncate(time.Second)) {
		t.Fatalf("expected the session to end with the demo, got %v", cookie.Expires)
	}
	me := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	me.AddCookie(cookie)
	meResponse := httptest.NewRecorder()
	authHandler.CurrentUser(meResponse, me)
	if !strings.Contains(meResponse.Body.String(), `"demo_expires_at"`) {
		t.Fatalf("expected the demo expiry in /me, got %s", meResponse.Body.String())
	}

	// The visitor's week, Monday to Monday, with a day's margin each side.
	stockholm, err := time.LoadLocation("Europe/Stockholm")
	if err != nil {
		t.Fatal(err)
	}
	local := now.In(stockholm)
	year, month, day := local.Date()
	monday := time.Date(year, month, day-(int(local.Weekday())+6)%7, 0, 0, 0, 0, stockholm)
	from, to := monday.Add(-time.Hour), monday.AddDate(0, 0, 7).Add(-time.Hour)
	timelineHandler := timeline.New(store, testutil.SignedInAs(userID))
	response := httptest.NewRecorder()
	path := "/api/timeline?from=" + from.UTC().Format(time.RFC3339) + "&to=" + to.UTC().Format(time.RFC3339)
	timelineHandler.ListTimeline(response, httptest.NewRequest(http.MethodGet, path, nil))
	body := response.Body.String()
	for _, title := range []string{"Team standup", "Dentist", "Lunch with Anna", "Climbing", "Grandma's birthday", "Book the summer cabin"} {
		if !strings.Contains(body, title) {
			t.Errorf("expected %q in the demo week", title)
		}
	}
	// The five weekday standups of the visitor's week.
	if got := strings.Count(body, "Team standup"); got != 5 {
		t.Errorf("expected 5 standups, got %d", got)
	}
}

func TestDemoUsersCannotReachOtherUsers(t *testing.T) {
	store := testutil.NewStore(t)
	userID, _ := startDemo(t, New(store, auth.New(store, auth.Config{})))
	timelineHandler := timeline.New(store, testutil.SignedInAs(userID))

	response := httptest.NewRecorder()
	timelineHandler.CreateHousehold(response, httptest.NewRequest(http.MethodPost, "/api/household", bytes.NewBufferString(`{"name":"Home","card_count":1}`)))
	if response.Code != http.StatusForbidden {
		t.Fatalf("household: expected status %d, got %d", http.StatusForbidden, response.Code)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/events/x/shares", bytes.NewBufferString(`{"email":"someone@example.com"}`))
	request.SetPathValue("eventID", uuid.NewString())
	response = httptest.NewRecorder()
	timelineHandler.ShareEvent(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("share: expected status %d, got %d", http.StatusForbidden, response.Code)
	}
}

func TestExpiredDemoUsersAreDeletedWithTheirData(t *testing.T) {
	ctx := context.Background()
	store := testutil.NewStore(t)
	// Users cannot be backdated, so this one expires soon.
	userID, err := store.CreateDemoUser(ctx, newEmail(), displayName, time.Now().Add(300*time.Millisecond))
	if err != nil {
		t.Fatal(err)
	}
	if err := seed(ctx, store, userID, time.Now()); err != nil {
		t.Fatal(err)
	}
	keptID, err := store.CreateDemoUser(ctx, newEmail(), displayName, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(600 * time.Millisecond)

	deleted, err := store.DeleteExpiredDemoUsers(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 1 {
		t.Fatalf("expected 1 deleted demo user, got %d", deleted)
	}
	if _, err := store.IsDemoUser(ctx, userID); !errors.Is(err, db.ErrUserNotFound) {
		t.Fatalf("expected the expired demo user to be gone, got %v", err)
	}
	if isDemo, err := store.IsDemoUser(ctx, keptID); err != nil || !isDemo {
		t.Fatalf("expected the live demo user to remain, got %v, %v", isDemo, err)
	}
}
