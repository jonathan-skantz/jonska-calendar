package middleware

import (
	"net/http/httptest"
	"testing"
)

func TestClientIP(t *testing.T) {
	tests := []struct {
		name      string
		forwarded string
		want      string
	}{
		{name: "connection address", want: "192.0.2.1"},
		{name: "forwarded by proxy", forwarded: "203.0.113.7", want: "203.0.113.7"},
		{name: "last hop wins", forwarded: "10.0.0.1, 203.0.113.7", want: "203.0.113.7"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest("GET", "/", nil)
			if test.forwarded != "" {
				request.Header.Set("X-Forwarded-For", test.forwarded)
			}
			if got := ClientIP(request); got != test.want {
				t.Fatalf("expected %q, got %q", test.want, got)
			}
		})
	}
}
