package middleware

import "net/http"

// API wraps all API routes. It refuses cross-site requests that change data,
// which the SameSite cookie alone does not stop for login or for same-site
// subdomains, and caps request bodies.
func API(routes http.Handler) http.Handler {
	// The import handler sets its own, larger body limit.
	return LimitBodies(http.NewCrossOriginProtection().Handler(routes), MaxBodyBytes, "/api/import/ics")
}
