package subscription

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
)

func TestNormalizeURL(t *testing.T) {
	for raw, want := range map[string]string{
		"webcal://example.com/a.ics":     "https://example.com/a.ics",
		" WEBCALS://example.com/a.ics ":  "https://example.com/a.ics",
		"http://example.com/a.ics?k=v":   "http://example.com/a.ics?k=v",
		"https://example.com:8443/a.ics": "https://example.com:8443/a.ics",
	} {
		if got, err := NormalizeURL(raw); err != nil || got != want {
			t.Errorf("%q: expected %q, got %q, %v", raw, want, got, err)
		}
	}
	for _, raw := range []string{"", "example.com/a.ics", "ftp://example.com/a.ics", "file:///etc/passwd", "https://", "https://" + strings.Repeat("a", maxURLLength)} {
		if _, err := NormalizeURL(raw); !errors.Is(err, ErrInvalidURL) {
			t.Errorf("%q: expected ErrInvalidURL, got %v", raw, err)
		}
	}
}

func TestIsPublic(t *testing.T) {
	for address, want := range map[string]bool{
		"93.184.216.34":    true,
		"2606:4700::1111":  true,
		"127.0.0.1":        false,
		"::1":              false,
		"10.1.2.3":         false,
		"172.16.0.1":       false,
		"192.168.1.10":     false,
		"169.254.169.254":  false,
		"100.64.0.1":       false,
		"0.0.0.0":          false,
		"fd00::1":          false,
		"fe80::1":          false,
		"::ffff:127.0.0.1": false,
		"224.0.0.1":        false,
	} {
		if got := isPublic(netip.MustParseAddr(address)); got != want {
			t.Errorf("%s: expected %v, got %v", address, want, got)
		}
	}
}

func TestFetch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ok.ics":
			_, _ = w.Write([]byte("BEGIN:VCALENDAR\r\nEND:VCALENDAR\r\n"))
		case "/large.ics":
			_, _ = w.Write([]byte(strings.Repeat("x", maxFeedBytes+1)))
		case "/redirect":
			http.Redirect(w, r, "/ok.ics", http.StatusFound)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	ctx := context.Background()
	open := NewClient(true)

	if body, err := open.Fetch(ctx, server.URL+"/redirect"); err != nil || !strings.HasPrefix(string(body), "BEGIN:VCALENDAR") {
		t.Fatalf("expected the calendar after a redirect, got %q, %v", body, err)
	}
	if _, err := open.Fetch(ctx, server.URL+"/large.ics"); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("expected ErrTooLarge, got %v", err)
	}
	_, err := open.Fetch(ctx, server.URL+"/missing.ics")
	if Message(err) != "the calendar's server answered 404 Not Found" {
		t.Fatalf("expected the status in the message, got %q", Message(err))
	}

	// The test server listens on loopback, which the guard refuses.
	if _, err := NewClient(false).Fetch(ctx, server.URL+"/ok.ics"); !errors.Is(err, ErrBlockedAddress) {
		t.Fatalf("expected ErrBlockedAddress, got %v", err)
	}
	if _, err := NewClient(false).Fetch(ctx, "http://127.0.0.1:1/ok.ics"); Message(err) != ErrBlockedAddress.Error() {
		t.Fatalf("expected the blocked message, got %q", Message(err))
	}
}
