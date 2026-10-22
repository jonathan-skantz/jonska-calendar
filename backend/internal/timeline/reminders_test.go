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
	"github.com/jonathan-skantz/calendar/backend/internal/testutil"
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
	handler := New(store, testutil.SignedInAs(ownerID))

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

func TestRemindersRepeatForEachOccurrence(t *testing.T) {
	ctx := context.Background()
	store, ownerID := setupTestStore(t)
	recipientID, err := store.Register(ctx, "recipient@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	owner := New(store, testutil.SignedInAs(ownerID))
	tomorrow := time.Now().In(stockholm).AddDate(0, 0, 1)
	first := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 10, 0, 0, 0, stockholm)
	body := lecture(map[string]any{"frequency": "daily", "interval": 1, "count": 4})
	body["starts_at"], body["ends_at"] = first, first.Add(time.Hour)
	eventID := createdID(t, sendEvent(t, owner, http.MethodPost, "", body))
	if _, err := store.CreateReminder(ctx, ownerID, eventID, 900); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ShareEvent(ctx, ownerID, eventID, recipientID, db.ShareFields{}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateReminder(ctx, recipientID, eventID, 3600); err != nil {
		t.Fatal(err)
	}

	list := func(handler *Handler) []db.Reminder {
		t.Helper()
		response := httptest.NewRecorder()
		handler.ListReminders(response, httptest.NewRequest(http.MethodGet, "/api/reminders", nil))
		var reminders []db.Reminder
		if err := json.NewDecoder(response.Body).Decode(&reminders); err != nil {
			t.Fatal(err)
		}
		return reminders
	}
	markNotified := func(reminder db.Reminder) {
		t.Helper()
		request := httptest.NewRequest(http.MethodPost, "/api/reminders/"+reminder.ID.String()+"/notified",
			bytes.NewBufferString(fmt.Sprintf(`{"occurrence_start":%q}`, reminder.OccurrenceStart.Format(time.RFC3339))))
		request.SetPathValue("reminderID", reminder.ID.String())
		response := httptest.NewRecorder()
		owner.MarkReminderNotified(response, request)
		if response.Code != http.StatusNoContent {
			t.Fatalf("mark notified: expected %d, got %d", http.StatusNoContent, response.Code)
		}
	}

	reminders := list(owner)
	if len(reminders) != 4 {
		t.Fatalf("expected a reminder for each of the four occurrences, got %+v", reminders)
	}
	for index, reminder := range reminders {
		start := first.AddDate(0, 0, index)
		if !reminder.OccurrenceStart.Equal(start) || !reminder.EventStartsAt.Equal(start) ||
			!reminder.RemindAt.Equal(start.Add(-15*time.Minute)) || reminder.NotifiedAt != nil {
			t.Fatalf("reminder %d: unexpected %+v", index, reminder)
		}
	}

	// Delivering the second occurrence's reminder leaves the others due.
	markNotified(reminders[1])
	reminders = list(owner)
	if reminders[0].NotifiedAt != nil || reminders[1].NotifiedAt == nil || reminders[2].NotifiedAt != nil {
		t.Fatalf("expected only the second delivered, got %+v", reminders)
	}

	// Moving the second occurrence re-arms its reminder; skipping the third
	// shows to the owner only.
	moved := occurrenceBody(first.AddDate(0, 0, 1).Add(2*time.Hour), first.AddDate(0, 0, 1).Add(3*time.Hour), nil)
	if code := updateScoped(t, owner, eventID, scopeQuery("this", first.AddDate(0, 0, 1)), moved).Code; code != http.StatusNoContent {
		t.Fatalf("move: expected %d, got %d", http.StatusNoContent, code)
	}
	skipped := occurrenceBody(first.AddDate(0, 0, 2), first.AddDate(0, 0, 2).Add(time.Hour), nil)
	skipped["is_skipped"] = true
	if code := updateScoped(t, owner, eventID, scopeQuery("this", first.AddDate(0, 0, 2)), skipped).Code; code != http.StatusNoContent {
		t.Fatalf("skip: expected %d, got %d", http.StatusNoContent, code)
	}
	reminders = list(owner)
	if !reminders[1].EventStartsAt.Equal(first.AddDate(0, 0, 1).Add(2*time.Hour)) || reminders[1].NotifiedAt != nil {
		t.Fatalf("expected the moved occurrence's reminder due again, got %+v", reminders[1])
	}
	if !reminders[2].EventIsSkipped || reminders[3].EventIsSkipped {
		t.Fatalf("expected only the third occurrence skipped, got %+v", reminders)
	}

	shared := list(New(store, testutil.SignedInAs(recipientID)))
	if len(shared) != 4 || !shared[0].IsShared || shared[0].OffsetSeconds != 3600 {
		t.Fatalf("expected the recipient's own reminders, got %+v", shared)
	}
	for _, reminder := range shared {
		if reminder.EventIsSkipped {
			t.Fatalf("recipients never see the owner's skip, got %+v", reminder)
		}
	}
}

func TestRemindersFollowTheRecipientsCustomization(t *testing.T) {
	store, ownerID := setupTestStore(t)
	ctx := context.Background()
	recipientID, err := store.Register(ctx, "recipient@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	handler := New(store, func(*http.Request) (uuid.UUID, error) { return recipientID, nil })
	share := func(title string, startsAt time.Time, customization db.Customization) uuid.UUID {
		eventID, err := store.CreateEvent(ctx, ownerID, db.Event{
			Title: title, StartsAt: startsAt, EndsAt: startsAt.Add(time.Hour), Kind: db.EventKindAttendable,
		})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := store.ShareEvent(ctx, ownerID, eventID, recipientID, db.ShareFields{}); err != nil {
			t.Fatal(err)
		}
		if err := store.CustomizeSharedEvent(ctx, recipientID, eventID, customization); err != nil {
			t.Fatal(err)
		}
		if _, err := store.CreateReminder(ctx, recipientID, eventID, 600); err != nil {
			t.Fatal(err)
		}
		return eventID
	}
	listReminders := func() map[uuid.UUID]db.Reminder {
		response := httptest.NewRecorder()
		handler.ListReminders(response, httptest.NewRequest(http.MethodGet, "/api/reminders", nil))
		var reminders []db.Reminder
		if err := json.NewDecoder(response.Body).Decode(&reminders); err != nil {
			t.Fatal(err)
		}
		byEvent := map[uuid.UUID]db.Reminder{}
		for _, reminder := range reminders {
			byEvent[reminder.EventID] = reminder
		}
		return byEvent
	}

	startsAt := time.Now().Add(48 * time.Hour).Truncate(time.Minute).UTC()
	shifted := db.Customization{Title: new("My lecture"), StartShiftSeconds: 1800, EndShiftSeconds: 1800}
	lectureID := share("Lecture", startsAt, shifted)
	// Ended half an hour ago, but the recipient moved it an hour later.
	justEnded := time.Now().Add(-90 * time.Minute).Truncate(time.Minute).UTC()
	lateID := share("Late", justEnded, db.Customization{StartShiftSeconds: 3600, EndShiftSeconds: 3600})

	reminders := listReminders()
	lecture := reminders[lectureID]
	if lecture.EventTitle != "My lecture" || !lecture.EventStartsAt.Equal(startsAt.Add(30*time.Minute)) ||
		!lecture.OccurrenceStart.Equal(startsAt.Add(30*time.Minute)) || !lecture.RemindAt.Equal(startsAt.Add(20*time.Minute)) {
		t.Fatalf("expected the reminder to follow the customization, got %+v", lecture)
	}
	if late, ok := reminders[lateID]; !ok || !late.EventStartsAt.Equal(justEnded.Add(time.Hour)) {
		t.Fatalf("expected the event shifted out of the past listed, got %+v", reminders)
	}

	if err := store.MarkReminderNotified(ctx, recipientID, lecture.ID, &lecture.OccurrenceStart, time.Now()); err != nil {
		t.Fatal(err)
	}
	if got := listReminders()[lectureID]; got.NotifiedAt == nil {
		t.Fatalf("expected the delivery at the shifted start found, got %+v", got)
	}
	shifted.StartShiftSeconds = 900
	if err := store.CustomizeSharedEvent(ctx, recipientID, lectureID, shifted); err != nil {
		t.Fatal(err)
	}
	if got := listReminders()[lectureID]; got.NotifiedAt != nil || !got.RemindAt.Equal(startsAt.Add(5*time.Minute)) {
		t.Fatalf("expected a new shift to arm the reminder again, got %+v", got)
	}
}

func TestExpandRemindersShiftsEachOccurrence(t *testing.T) {
	now := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	rule, zone := "FREQ=DAILY", "UTC"
	// Daily 11:00 to 11:30 from the day before; the shift makes today's end
	// at 12:30, so it has not ended yet.
	firstStart := now.AddDate(0, 0, -1).Add(-time.Hour)
	reminder := db.Reminder{
		ID: uuid.New(), OffsetSeconds: 300, EventTitle: "Stand-up",
		EventStartsAt: firstStart, EventEndsAt: firstStart.Add(30 * time.Minute),
		RecurrenceRule: &rule, RecurrenceTimeZone: &zone, IsShared: true,
		Customization: &db.Customization{StartShiftSeconds: 3600, EndShiftSeconds: 3600},
	}
	expanded, err := expandReminders([]db.Reminder{reminder}, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(expanded) != 31 || !expanded[0].EventStartsAt.Equal(now) || !expanded[0].RemindAt.Equal(now.Add(-5*time.Minute)) {
		t.Fatalf("expected today's occurrence shifted to noon first, got %d: %+v", len(expanded), expanded[0])
	}
}
