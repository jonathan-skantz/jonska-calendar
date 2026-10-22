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

func refusePrivate(_, address string, _ syscall.RawConn) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return err
	}
	ip, err := netip.ParseAddr(host)
	if err != nil {
		return err
	}
	if !isPublic(ip) {
		return ErrBlockedAddress
	}
	return nil
}

// sharedAddressSpace is carrier-grade NAT (RFC 6598), which IsPrivate does
// not cover.
var sharedAddressSpace = netip.MustParsePrefix("100.64.0.0/10")

func isPublic(ip netip.Addr) bool {
	ip = ip.Unmap()
	return ip.IsGlobalUnicast() && !ip.IsPrivate() && !sharedAddressSpace.Contains(ip)
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

// Message is what a subscriber is told about a failed fetch. Network
// details stay in the server's log.
func Message(err error) string {
	if _, ok := errors.AsType[statusError](err); ok {
		return err.Error()
	}
	for _, known := range []error{ErrInvalidURL, ErrBlockedAddress, ErrTooLarge, ErrUnreachable} {
		if errors.Is(err, known) {
			return known.Error()
		}
	}
	return err.Error()
}
