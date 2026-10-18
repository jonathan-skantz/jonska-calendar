package timeline

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

func TestPathID(t *testing.T) {
	id := uuid.New()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.SetPathValue("eventID", id.String())
	response := httptest.NewRecorder()
	if got, ok := pathID(response, request, "eventID"); !ok || got != id {
		t.Fatalf("expected %s, got %s (ok=%v)", id, got, ok)
	}

	request.SetPathValue("eventID", "not-a-uuid")
	response = httptest.NewRecorder()
	if _, ok := pathID(response, request, "eventID"); ok {
		t.Fatal("expected an invalid id to fail")
	}
	if response.Code != http.StatusBadRequest || response.Body.String() != "invalid event id\n" {
		t.Fatalf("expected 400 invalid event id, got %d %q", response.Code, response.Body.String())
	}
}

func TestDecodeJSONRejectsMalformedBody(t *testing.T) {
	var body struct{ Name string }
	request := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"name":`))
	response := httptest.NewRecorder()
	if decodeJSON(response, request, &body) {
		t.Fatal("expected malformed JSON to fail")
	}
	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}
}
