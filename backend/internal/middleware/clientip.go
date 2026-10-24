package middleware

import (
	"net"
	"net/http"
	"strings"
)

// ClientIP is the address of the client that sent r. In production the API
// is reachable only through Caddy, which replaces X-Forwarded-For with the
// address it saw, so its last entry can be trusted. Without the header, the
// connection's own address is used.
func ClientIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		hops := strings.Split(forwarded, ",")
		if ip := strings.TrimSpace(hops[len(hops)-1]); ip != "" {
			return ip
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
