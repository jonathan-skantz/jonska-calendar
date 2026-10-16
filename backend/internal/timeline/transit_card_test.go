package timeline

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestValidateTransitCardDaysRequest(t *testing.T) {
	today := time.Date(2026, 10, 16, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name    string
		request transitCardDaysRequest
		wantErr error
	}{
		{name: "bad date", request: transitCardDaysRequest{From: "2026/10/19", To: "2026-10-25"}, wantErr: ErrInvalidDate},
		{name: "reversed range", request: transitCardDaysRequest{From: "2026-10-25", To: "2026-10-19"}, wantErr: ErrInvalidDateRange},
		{name: "range too long", request: transitCardDaysRequest{From: "2026-10-19", To: "2026-12-20"}, wantErr: ErrInvalidDateRange},
		{name: "duplicate needed date", request: transitCardDaysRequest{From: "2026-10-19", To: "2026-10-25", NeededDates: []string{"2026-10-20", "2026-10-20"}}, wantErr: ErrInvalidTransitCardDates},
		{name: "needed date outside range", request: transitCardDaysRequest{From: "2026-10-19", To: "2026-10-25", NeededDates: []string{"2026-10-26"}}, wantErr: ErrInvalidTransitCardDates},
		{name: "day both needed and maybe", request: transitCardDaysRequest{From: "2026-10-19", To: "2026-10-25", NeededDates: []string{"2026-10-20"}, MaybeDates: []string{"2026-10-20"}}, wantErr: ErrInvalidTransitCardDates},
		{name: "maybe date outside range", request: transitCardDaysRequest{From: "2026-10-19", To: "2026-10-25", MaybeDates: []string{"2026-10-18"}}, wantErr: ErrInvalidTransitCardDates},
		{name: "bad needed date", request: transitCardDaysRequest{From: "2026-10-19", To: "2026-10-25", NeededDates: []string{"monday"}}, wantErr: ErrInvalidDate},
		{name: "ended everywhere", request: transitCardDaysRequest{From: "2026-10-14", To: "2026-10-18"}, wantErr: ErrPastTransitCardDay},
		{name: "yesterday in UTC is allowed", request: transitCardDaysRequest{From: "2026-10-15", To: "2026-10-18"}},
		{name: "empty selection", request: transitCardDaysRequest{From: "2026-10-19", To: "2026-10-25"}},
		{name: "longest range", request: transitCardDaysRequest{From: "2026-10-19", To: "2026-12-19"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := validateTransitCardDaysRequest(test.request, today)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("expected error %v, got %v", test.wantErr, err)
			}
		})
	}
}

func TestTransitCardRequiresAuthentication(t *testing.T) {
	handler := New(nil, func(*http.Request) (uuid.UUID, error) {
		return uuid.Nil, errors.New("not authenticated")
	})
	response := httptest.NewRecorder()
	handler.SetTransitCardDays(response, httptest.NewRequest(http.MethodPut, "/api/transit-card/days", bytes.NewBufferString(`{}`)))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, response.Code)
	}
}

func TestTransitCardDays(t *testing.T) {
	store, userID := setupTestStore(t)
	otherUserID, err := store.Register(context.Background(), "other@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	currentUser := userID
	handler := New(store, func(*http.Request) (uuid.UUID, error) { return currentUser, nil })

	from := time.Now().UTC().AddDate(0, 0, 1).Format(dateLayout)
	to := time.Now().UTC().AddDate(0, 0, 7).Format(dateLayout)
	day := func(offset int) string { return time.Now().UTC().AddDate(0, 0, offset).Format(dateLayout) }

	put := func(body any) int {
		payload, _ := json.Marshal(body)
		response := httptest.NewRecorder()
		handler.SetTransitCardDays(response, httptest.NewRequest(http.MethodPut, "/api/transit-card/days", bytes.NewReader(payload)))
		return response.Code
	}
	list := func() []transitCardDayResponse {
		response := httptest.NewRecorder()
		handler.ListTransitCardDays(response, httptest.NewRequest(http.MethodGet, "/api/transit-card/days?from="+from+"&to="+to, nil))
		if response.Code != http.StatusOK {
			t.Fatalf("list: expected status %d, got %d: %s", http.StatusOK, response.Code, response.Body.String())
		}
		var days []transitCardDayResponse
		if err := json.NewDecoder(response.Body).Decode(&days); err != nil {
			t.Fatal(err)
		}
		return days
	}
	datesWithNeed := func(days []transitCardDayResponse, need string) []string {
		dates := []string{}
		for _, day := range days {
			if day.Need == need {
				dates = append(dates, day.Date)
			}
		}
		return dates
	}

	if days := list(); len(days) != 0 {
		t.Fatalf("expected no reported days before submitting, got %d", len(days))
	}

	if code := put(transitCardDaysRequest{From: from, To: to, NeededDates: []string{day(2), day(4)}, MaybeDates: []string{day(5)}}); code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, code)
	}
	days := list()
	if len(days) != 7 {
		t.Fatalf("expected every day in the range to be reported, got %d", len(days))
	}
	if needed := datesWithNeed(days, "yes"); len(needed) != 2 || needed[0] != day(2) || needed[1] != day(4) {
		t.Fatalf("unexpected needed days %v", needed)
	}
	if maybe := datesWithNeed(days, "maybe"); len(maybe) != 1 || maybe[0] != day(5) {
		t.Fatalf("unexpected maybe days %v", maybe)
	}
	if days[0].DisplayName != "test" || days[0].UserID != userID || days[0].Status != transitCardResolved {
		t.Fatalf("unexpected day metadata %+v", days[0])
	}

	// An empty selection is an explicit answer, not a reset to "not reported".
	if code := put(transitCardDaysRequest{From: from, To: to, NeededDates: []string{}}); code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, code)
	}
	days = list()
	if len(days) != 7 || len(datesWithNeed(days, "no")) != 7 {
		t.Fatalf("expected 7 reported days with none needed, got %d days and %v", len(days), datesWithNeed(days, "no"))
	}

	if code := put(transitCardDaysRequest{From: day(-3), To: day(-2)}); code != http.StatusBadRequest {
		t.Fatalf("expected past days to be rejected, got %d", code)
	}

	currentUser = otherUserID
	if days := list(); len(days) != 0 {
		t.Fatalf("expected other user to see no days, got %d", len(days))
	}
}
