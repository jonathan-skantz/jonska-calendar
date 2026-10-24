// Package subscription follows calendars by URL: it fetches iCalendar feeds,
// keeps their events as read-only projections, and refreshes them on a
// schedule.
package subscription

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"syscall"
	"time"

	"github.com/jonathan-skantz/calendar/backend/internal/ical"
)

const (
	// maxFeedBytes bounds one fetch, so a feed cannot fill the server's
	// memory.
	maxFeedBytes  = 10 << 20
	fetchTimeout  = 30 * time.Second
	maxRedirects  = 5
	maxURLLength  = 2048
	userAgentName = "jonska-calendar"
)

// Errors a subscriber sees. They say what went wrong without details of the
// server's network.
var (
	ErrInvalidURL     = errors.New("the address must be an http, https or webcal URL")
	ErrBlockedAddress = errors.New("the address points to a private network, which is not allowed")
	ErrTooLarge       = errors.New("the calendar is larger than 10 MB")
	ErrUnreachable    = errors.New("the calendar could not be reached")
)

// statusError is a fetch the server answered with an error status.
type statusError int

func (status statusError) Error() string {
	return fmt.Sprintf("the calendar's server answered %d %s", int(status), http.StatusText(int(status)))
}

// NormalizeURL checks a subscription URL and turns webcal:// into https://,
// which is what calendar apps fetch it with.
func NormalizeURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || len(raw) > maxURLLength {
		return "", ErrInvalidURL
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" {
		return "", ErrInvalidURL
	}
	switch strings.ToLower(parsed.Scheme) {
	case "webcal", "webcals", "https":
		parsed.Scheme = "https"
	case "http":
		parsed.Scheme = "http"
	default:
		return "", ErrInvalidURL
	}
	return parsed.String(), nil
}

// Client fetches feeds. Unless private addresses are allowed, it refuses to
// connect to loopback, private, link-local and other non-public addresses.
// The check runs on the address actually dialed, so neither DNS nor a
// redirect can lead it into the server's own network.
type Client struct {
	HTTP *http.Client
}

func NewClient(allowPrivate bool) *Client {
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	if !allowPrivate {
		dialer.Control = refusePrivate
	}
	transport := &http.Transport{
		// A proxy from the environment would be dialed instead of the feed.
		Proxy:                 nil,
		DialContext:           dialer.DialContext,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: fetchTimeout,
		MaxIdleConns:          10,
		IdleConnTimeout:       90 * time.Second,
	}
	return &Client{HTTP: &http.Client{
		Transport: transport,
		Timeout:   fetchTimeout,
		CheckRedirect: func(request *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirects {
				return errors.New("too many redirects")
			}
			if request.URL.Scheme != "http" && request.URL.Scheme != "https" {
				return ErrInvalidURL
			}
			return nil
		},
	}}
}

// refusePrivate allows only public addresses on the web's own ports, so a
// feed URL cannot probe other services.
func refusePrivate(_, address string, _ syscall.RawConn) error {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return err
	}
	ip, err := netip.ParseAddr(host)
	if err != nil {
		return err
	}
	if !isPublic(ip) || (port != "80" && port != "443") {
		return ErrBlockedAddress
	}
	return nil
}

// nonPublic are ranges that IsGlobalUnicast and IsPrivate let through but
// that are not ordinary public hosts: carrier-grade NAT, benchmarking and
// IETF protocol ranges, and the IPv6 ranges that embed an IPv4 address,
// which a NAT64 or 6to4 gateway could route to a private one.
var nonPublic = []netip.Prefix{
	netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("::/96"),
	netip.MustParsePrefix("64:ff9b::/96"),
	netip.MustParsePrefix("64:ff9b:1::/48"),
	netip.MustParsePrefix("2001::/32"),
	netip.MustParsePrefix("2002::/16"),
}

func isPublic(ip netip.Addr) bool {
	ip = ip.Unmap()
	if !ip.IsGlobalUnicast() || ip.IsPrivate() {
		return false
	}
	for _, prefix := range nonPublic {
		if prefix.Contains(ip) {
			return false
		}
	}
	return true
}

// Fetch downloads a feed at a URL that NormalizeURL accepted.
func (c *Client) Fetch(ctx context.Context, feedURL string) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, nil)
	if err != nil {
		return nil, ErrInvalidURL
	}
	request.Header.Set("Accept", "text/calendar, */*;q=0.5")
	request.Header.Set("User-Agent", userAgentName)
	response, err := c.HTTP.Do(request)
	if err != nil {
		if errors.Is(err, ErrBlockedAddress) {
			return nil, ErrBlockedAddress
		}
		if errors.Is(err, ErrInvalidURL) {
			return nil, ErrInvalidURL
		}
		// The URL can hold a secret, as private calendar links do, so it
		// is kept out of the error and the server's log.
		if urlErr, ok := errors.AsType[*url.Error](err); ok {
			err = urlErr.Err
		}
		return nil, fmt.Errorf("%w: %v", ErrUnreachable, err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, statusError(response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxFeedBytes+1))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnreachable, err)
	}
	if len(body) > maxFeedBytes {
		return nil, ErrTooLarge
	}
	return body, nil
}

// ErrUnreadable is what a subscriber is told about any other failure.
var ErrUnreadable = errors.New("the calendar could not be read")

// Message is what a subscriber is told about a failed fetch. Network
// details stay in the server's log.
func Message(err error) string {
	if _, ok := errors.AsType[statusError](err); ok {
		return err.Error()
	}
	for _, known := range []error{
		ErrInvalidURL, ErrBlockedAddress, ErrTooLarge, ErrUnreachable, ical.ErrInvalidFile, ical.ErrTooManyEvents,
	} {
		if errors.Is(err, known) {
			return known.Error()
		}
	}
	return ErrUnreadable.Error()
}
