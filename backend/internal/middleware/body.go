// Package middleware holds the HTTP wrappers applied to every API route.
package middleware

import (
	"net/http"
	"slices"
)

// MaxBodyBytes is enough for any JSON request the app makes except an
// iCalendar import, which sets its own larger limit.
const MaxBodyBytes = 64 << 10

// LimitBodies caps request bodies at limit bytes. Paths in exempt are left
// to their handler, which must set its own limit.
func LimitBodies(next http.Handler, limit int64, exempt ...string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !slices.Contains(exempt, r.URL.Path) {
			if r.ContentLength > limit {
				http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, limit)
		}
		next.ServeHTTP(w, r)
	})
}
