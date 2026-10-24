package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAPIRefusesCrossSiteChanges(t *testing.T) {
	handler := API(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	tests := []struct {
		name    string
		method  string
		headers map[string]string
		want    int
	}{
		{name: "same origin", method: http.MethodPost, headers: map[string]string{"Sec-Fetch-Site": "same-origin"}, want: http.StatusOK},
		{name: "cross site", method: http.MethodPost, headers: map[string]string{"Sec-Fetch-Site": "cross-site"}, want: http.StatusForbidden},
		{name: "same site subdomain", method: http.MethodPost, headers: map[string]string{"Sec-Fetch-Site": "same-site"}, want: http.StatusForbidden},
		{name: "foreign origin", method: http.MethodDelete, headers: map[string]string{"Origin": "https://evil.example"}, want: http.StatusForbidden},
		{name: "own origin", method: http.MethodPost, headers: map[string]string{"Origin": "http://cal.example"}, want: http.StatusOK},
		{name: "non-browser client", method: http.MethodPost, want: http.StatusOK},
		{name: "cross-site read", method: http.MethodGet, headers: map[string]string{"Sec-Fetch-Site": "cross-site"}, want: http.StatusOK},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(test.method, "http://cal.example/api/events", nil)
			for name, value := range test.headers {
				request.Header.Set(name, value)
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.want {
				t.Fatalf("expected status %d, got %d", test.want, response.Code)
			}
		})
	}
}
