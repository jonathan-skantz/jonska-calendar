package timeline

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/jonathan-skantz/calendar/backend/db"
	"github.com/jonathan-skantz/calendar/backend/internal/testutil"
)

func TestHousehold(t *testing.T) {
	store, ownerID := setupTestStore(t)
	ctx := context.Background()
	memberID, err := store.Register(ctx, "member@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Register(ctx, "outsider@example.com", "hash"); err != nil {
		t.Fatal(err)
	}
	currentUser := ownerID
	handler := New(store, func(*http.Request) (uuid.UUID, error) { return currentUser, nil })

	call := func(method string, fn http.HandlerFunc, body string) *httptest.ResponseRecorder {
		response := httptest.NewRecorder()
		fn(response, httptest.NewRequest(method, "/api/household", bytes.NewBufferString(body)))
		return response
	}
	get := func() *db.Household {
		response := call(http.MethodGet, handler.GetHousehold, "")
		if response.Code != http.StatusOK {
			t.Fatalf("get: expected status %d, got %d", http.StatusOK, response.Code)
		}
		var household *db.Household
		if err := json.NewDecoder(response.Body).Decode(&household); err != nil {
			t.Fatal(err)
		}
		return household
	}

	if household := get(); household != nil {
		t.Fatalf("expected no household, got %+v", household)
	}
	if code := call(http.MethodPost, handler.CreateHousehold, `{"name":" ","transit_card_count":1}`).Code; code != http.StatusBadRequest {
		t.Fatalf("expected blank name to be rejected, got %d", code)
	}
	if code := call(http.MethodPost, handler.CreateHousehold, `{"name":"Family","transit_card_count":0}`).Code; code != http.StatusBadRequest {
		t.Fatalf("expected zero cards to be rejected, got %d", code)
	}
	if code := call(http.MethodPost, handler.AddHouseholdMember, `{"email":"member@example.com"}`).Code; code != http.StatusNotFound {
		t.Fatalf("expected adding without a household to fail, got %d", code)
	}
	if code := call(http.MethodPost, handler.CreateHousehold, `{"name":"Family","transit_card_count":2}`).Code; code != http.StatusCreated {
		t.Fatalf("create: expected status %d, got %d", http.StatusCreated, code)
	}
	if code := call(http.MethodPost, handler.CreateHousehold, `{"name":"Second","transit_card_count":1}`).Code; code != http.StatusConflict {
		t.Fatalf("expected a second household to be rejected, got %d", code)
	}
	if code := call(http.MethodPost, handler.AddHouseholdMember, `{"email":"nobody@example.com"}`).Code; code != http.StatusNotFound {
		t.Fatalf("expected unknown email to be rejected, got %d", code)
	}
	if code := call(http.MethodPost, handler.AddHouseholdMember, `{"email":" Member@Example.com "}`).Code; code != http.StatusNoContent {
		t.Fatalf("add member: expected status %d, got %d", http.StatusNoContent, code)
	}
	if code := call(http.MethodPost, handler.AddHouseholdMember, `{"email":"member@example.com"}`).Code; code != http.StatusConflict {
		t.Fatalf("expected adding an existing member to be rejected, got %d", code)
	}
	if code := call(http.MethodPatch, handler.UpdateHousehold, `{"name":"Home","transit_card_count":3}`).Code; code != http.StatusNoContent {
		t.Fatalf("update: expected status %d, got %d", http.StatusNoContent, code)
	}
	household := get()
	if household.Name != "Home" || household.TransitCardCount != 3 || len(household.Members) != 2 ||
		household.Members[0].UserID != ownerID || household.Members[1].DisplayName != "member" {
		t.Fatalf("unexpected household %+v", household)
	}

	// The outsider's own household must not be able to pull in a member of another.
	outsider, err := store.FindUserByEmail(ctx, "outsider@example.com")
	if err != nil {
		t.Fatal(err)
	}
	currentUser = outsider.ID
	if code := call(http.MethodPost, handler.CreateHousehold, `{"name":"Other","transit_card_count":1}`).Code; code != http.StatusCreated {
		t.Fatalf("outsider create: got %d", code)
	}
	if code := call(http.MethodPost, handler.AddHouseholdMember, `{"email":"member@example.com"}`).Code; code != http.StatusConflict {
		t.Fatalf("expected member of another household to be rejected, got %d", code)
	}

	currentUser = memberID
	if code := call(http.MethodDelete, handler.LeaveHousehold, "").Code; code != http.StatusNoContent {
		t.Fatalf("leave: expected status %d, got %d", http.StatusNoContent, code)
	}
	if household := get(); household != nil {
		t.Fatalf("expected member to have left, got %+v", household)
	}
	currentUser = ownerID
	if household := get(); household == nil || len(household.Members) != 1 {
		t.Fatalf("expected owner to remain alone, got %+v", household)
	}
	if code := call(http.MethodDelete, handler.LeaveHousehold, "").Code; code != http.StatusNoContent {
		t.Fatalf("leave: expected status %d, got %d", http.StatusNoContent, code)
	}
	if code := call(http.MethodDelete, handler.LeaveHousehold, "").Code; code != http.StatusNotFound {
		t.Fatalf("expected leaving twice to fail, got %d", code)
	}
}

func TestTransitCardConflicts(t *testing.T) {
	store, firstID := setupTestStore(t)
	ctx := context.Background()
	register := func(email string) uuid.UUID {
		id, err := store.Register(ctx, email, "hash")
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	secondID := register("second@example.com")
	thirdID := register("third@example.com")
	outsiderID := register("outsider@example.com")

	if _, err := store.CreateHousehold(ctx, firstID, "Family", 2); err != nil {
		t.Fatal(err)
	}
	for _, id := range []uuid.UUID{secondID, thirdID} {
		if err := store.AddHouseholdMember(ctx, firstID, id); err != nil {
			t.Fatal(err)
		}
	}

	day := time.Now().UTC().AddDate(0, 0, 2).Truncate(24 * time.Hour)
	declare := func(userID uuid.UUID, need string) {
		var needed, maybe []time.Time
		switch need {
		case "yes":
			needed = []time.Time{day}
		case "maybe":
			maybe = []time.Time{day}
		}
		if err := store.SetTransitCardDays(ctx, userID, day, day, needed, maybe); err != nil {
			t.Fatal(err)
		}
	}
	list := func(userID uuid.UUID) map[uuid.UUID]transitCardDayResponse {
		handler := New(store, testutil.SignedInAs(userID))
		date := day.Format(dateLayout)
		response := httptest.NewRecorder()
		handler.ListTransitCardDays(response, httptest.NewRequest(http.MethodGet, "/api/transit-card/days?from="+date+"&to="+date, nil))
		if response.Code != http.StatusOK {
			t.Fatalf("list: expected status %d, got %d", http.StatusOK, response.Code)
		}
		var days []transitCardDayResponse
		if err := json.NewDecoder(response.Body).Decode(&days); err != nil {
			t.Fatal(err)
		}
		byUser := map[uuid.UUID]transitCardDayResponse{}
		for _, day := range days {
			byUser[day.UserID] = day
		}
		return byUser
	}

	declare(firstID, "yes")
	declare(secondID, "yes")
	declare(thirdID, "maybe")
	declare(outsiderID, "yes")
	days := list(firstID)
	if len(days) != 3 {
		t.Fatalf("expected the three household members' days, got %+v", days)
	}
	if _, ok := days[outsiderID]; ok {
		t.Fatal("household member saw an outsider's declaration")
	}
	for id, day := range days {
		if day.Status != transitCardResolved {
			t.Fatalf("expected no conflict with two yes for two cards, %s got %s", id, day.Status)
		}
	}

	declare(thirdID, "yes")
	for id, day := range list(secondID) {
		if day.Status != transitCardConflict {
			t.Fatalf("expected conflict with three yes for two cards, %s got %s", id, day.Status)
		}
	}

	declare(thirdID, "maybe")
	declare(secondID, "maybe")
	declare(firstID, "maybe")
	for id, day := range list(firstID) {
		if day.Status != transitCardResolved {
			t.Fatalf("maybe must never conflict, %s got %s", id, day.Status)
		}
	}

	outsiderDays := list(outsiderID)
	if len(outsiderDays) != 1 || outsiderDays[outsiderID].Status != transitCardResolved {
		t.Fatalf("outsider should only see their own resolved day, got %+v", outsiderDays)
	}
}
