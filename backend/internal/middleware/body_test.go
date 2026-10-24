package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// readAll answers 200 with the body length, or 400 when reading fails.
var readAll = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	if _, err := io.ReadAll(r.Body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
})

func TestLimitBodies(t *testing.T) {
	handler := LimitBodies(readAll, 10, "/import")
	tests := []struct {
		name    string
		path    string
		body    string
		chunked bool
		want    int
	}{
		{name: "within limit", path: "/api", body: "0123456789", want: http.StatusOK},
		{name: "declared too large", path: "/api", body: "0123456789x", want: http.StatusRequestEntityTooLarge},
		{name: "streamed too large", path: "/api", body: "0123456789x", chunked: true, want: http.StatusBadRequest},
		{name: "exempt path", path: "/import", body: "0123456789x", want: http.StatusOK},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, test.path, strings.NewReader(test.body))
			if test.chunked {
				request.ContentLength = -1
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.want {
				t.Fatalf("expected status %d, got %d", test.want, response.Code)
			}
		})
	}
}
