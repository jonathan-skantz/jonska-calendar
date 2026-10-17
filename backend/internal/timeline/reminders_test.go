package timeline

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jonathan-skantz/calendar/backend/db"
)

func TestCreateReminderValidatesOffset(t *testing.T) {
	handler := New(nil, func(*http.Request) (uuid.UUID, error) { return uuid.New(), nil })
	for name, body := range map[string]string{
		"negative":       `{"offset_seconds":-60}`,
		"partial minute": `{"offset_seconds":90}`,
		"over 30 days":   `{"offset_seconds":2592060}`,
	} {
		t.Run(name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/events/x/reminders", bytes.NewBufferString(body))
			request.SetPathValue("eventID", uuid.NewString())
			response := httptest.NewRecorder()
			handler.CreateReminder(response, request)
			if response.Code != http.StatusBadRequest || response.Body.String() != ErrInvalidOffset.Error()+"\n" {
				t.Fatalf("expected offset error, got %d: %s", response.Code, response.Body.String())
			}
		})
	}
}

func TestRemindersArePrivatePerUser(t *testing.T) {
	store, ownerID := setupTestStore(t)
	ctx := context.Background()
	recipientID, err := store.Register(ctx, "recipient@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	strangerID, err := store.Register(ctx, "stranger@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	currentUser := ownerID
	handler := New(store, func(*http.Request) (uuid.UUID, error) { return currentUser, nil })
	as := func(userID uuid.UUID) { currentUser = userID }

	startsAt := time.Now().Add(48 * time.Hour).Truncate(time.Minute).UTC()
	eventID, err := store.CreateEvent(ctx, ownerID, db.Event{
		Title:    "Lecture",
		StartsAt: startsAt,
		EndsAt:   startsAt.Add(2 * time.Hour),
		Kind:     "attendable",
	})
	if err != nil {
		t.Fatal(err)
	}
	shareID, err := store.ShareEvent(ctx, ownerID, eventID, recipientID, db.ShareFields{Title: new("Lecture for you")})
	if err != nil {
		t.Fatal(err)
	}

	createReminder := func(offsetSeconds int) *httptest.ResponseRecorder {
		request := httptest.NewRequest(http.MethodPost, "/api/events/"+eventID.String()+"/reminders",
			bytes.NewBufferString(fmt.Sprintf(`{"offset_seconds":%d}`, offsetSeconds)))
		request.SetPathValue("eventID", eventID.String())
		response := httptest.NewRecorder()
		handler.CreateReminder(response, request)
		return response
	}
	listReminders := func() []db.Reminder {
		response := httptest.NewRecorder()
		handler.ListReminders(response, httptest.NewRequest(http.MethodGet, "/api/reminders", nil))
		var reminders []db.Reminder
		if err := json.NewDecoder(response.Body).Decode(&reminders); err != nil {
			t.Fatal(err)
		}
		return reminders
	}

	if code := createReminder(900).Code; code != http.StatusCreated {
		t.Fatalf("owner reminder: expected %d, got %d", http.StatusCreated, code)
	}
	if code := createReminder(900).Code; code != http.StatusConflict {
		t.Fatalf("duplicate offset: expected %d, got %d", http.StatusConflict, code)
	}
	as(recipientID)
	if code := createReminder(86400).Code; code != http.StatusCreated {
		t.Fatalf("recipient reminder: expected %d, got %d", http.StatusCreated, code)
	}
	as(strangerID)
	if code := createReminder(900).Code; code != http.StatusNotFound {
		t.Fatalf("stranger reminder: expected %d, got %d", http.StatusNotFound, code)
	}
	if reminders := listReminders(); len(reminders) != 0 {
		t.Fatalf("stranger: expected no reminders, got %+v", reminders)
	}

	as(ownerID)
	owned := listReminders()
	if len(owned) != 1 || owned[0].OffsetSeconds != 900 || owned[0].IsShared ||
		!owned[0].RemindAt.Equal(startsAt.Add(-15*time.Minute)) {
		t.Fatalf("owner: unexpected reminders %+v", owned)
	}
	as(recipientID)
	shared := listReminders()
	if len(shared) != 1 || shared[0].OffsetSeconds != 86400 || !shared[0].IsShared ||
		shared[0].EventTitle != "Lecture for you" {
		t.Fatalf("recipient: unexpected reminders %+v", shared)
	}

	// Another user's reminder cannot be removed or marked.
	as(strangerID)
	request := httptest.NewRequest(http.MethodDelete, "/api/reminders/"+owned[0].ID.String(), nil)
	request.SetPathValue("reminderID", owned[0].ID.String())
	response := httptest.NewRecorder()
	handler.DeleteReminder(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("stranger delete: expected %d, got %d", http.StatusNotFound, response.Code)
	}

	// Removing the share removes the recipient's reminders.
	if err := store.DeleteEventShare(ctx, ownerID, eventID, shareID); err != nil {
		t.Fatal(err)
	}
	as(recipientID)
	if reminders := listReminders(); len(reminders) != 0 {
		t.Fatalf("recipient after unshare: expected no reminders, got %+v", reminders)
	}
	if _, err := store.ShareEvent(ctx, ownerID, eventID, recipientID, db.ShareFields{}); err != nil {
		t.Fatal(err)
	}
	if reminders := listReminders(); len(reminders) != 0 {
		t.Fatalf("recipient after reshare: expected old reminders to stay deleted, got %+v", reminders)
	}
}

func TestReminderNotifiedResetsWhenEventMoves(t *testing.T) {
	store, ownerID := setupTestStore(t)
	ctx := context.Background()
	handler := New(store, func(*http.Request) (uuid.UUID, error) { return ownerID, nil })

	startsAt := time.Now().Add(48 * time.Hour).Truncate(time.Minute).UTC()
	event := db.Event{Title: "Dentist", StartsAt: startsAt, EndsAt: startsAt.Add(time.Hour), Kind: "attendable"}
	eventID, err := store.CreateEvent(ctx, ownerID, event)
	if err != nil {
		t.Fatal(err)
	}
	reminderID, err := store.CreateReminder(ctx, ownerID, eventID, 3600)
	if err != nil {
		t.Fatal(err)
	}

	for range 2 {
		request := httptest.NewRequest(http.MethodPost, "/api/reminders/"+reminderID.String()+"/notified", nil)
		request.SetPathValue("reminderID", reminderID.String())
		response := httptest.NewRecorder()
		handler.MarkReminderNotified(response, request)
		if response.Code != http.StatusNoContent {
			t.Fatalf("mark notified: expected %d, got %d", http.StatusNoContent, response.Code)
		}
	}
	notifiedAt := func() *time.Time {
		reminders, err := store.ListReminders(ctx, ownerID)
		if err != nil {
			t.Fatal(err)
		}
		if len(reminders) != 1 {
			t.Fatalf("expected one reminder, got %+v", reminders)
		}
		return reminders[0].NotifiedAt
	}
	if notifiedAt() == nil {
		t.Fatal("expected the reminder to be marked notified")
	}

	// Saving the event without moving it keeps the delivery.
	if err := store.UpdateEvent(ctx, ownerID, eventID, event); err != nil {
		t.Fatal(err)
	}
	if notifiedAt() == nil {
		t.Fatal("expected an unmoved event to keep its delivery")
	}

	event.StartsAt = startsAt.Add(time.Hour)
	event.EndsAt = startsAt.Add(2 * time.Hour)
	if err := store.UpdateEvent(ctx, ownerID, eventID, event); err != nil {
		t.Fatal(err)
	}
	if notifiedAt() != nil {
		t.Fatal("expected a moved event's reminder to fire again")
	}

	if err := store.DeleteTimelineItem(ctx, ownerID, eventID); err != nil {
		t.Fatal(err)
	}
	if reminders, err := store.ListReminders(ctx, ownerID); err != nil || len(reminders) != 0 {
		t.Fatalf("expected reminders to be deleted with the event, got %+v, %v", reminders, err)
	}
}
