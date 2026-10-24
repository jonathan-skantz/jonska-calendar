// Package demo signs visitors in to throwaway accounts with sample data, so
// anyone can try the app without an invitation.
package demo

import (
	"crypto/rand"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/jonathan-skantz/calendar/backend/db"
	"github.com/jonathan-skantz/calendar/backend/internal/auth"
	"github.com/jonathan-skantz/calendar/backend/internal/middleware"
	"github.com/jonathan-skantz/calendar/backend/internal/ratelimit"
	"github.com/jonathan-skantz/calendar/backend/internal/timeline"
)

const (
	// lifetime is how long a demo account and its session last. The cleanup
	// job deletes the account afterwards.
	lifetime        = 24 * time.Hour
	startsPerClient = 10
	startWindow     = time.Hour
	// maxLiveUsers bounds how many demo accounts exist at once, whatever
	// addresses the visitors come from.
	maxLiveUsers = 200
	displayName  = "Demo visitor"
)

type Handler struct {
	Store *db.Store
	Auth  *auth.Handler
	// Now is replaceable so tests can fix the seeded week.
	Now     func() time.Time
	limiter *ratelimit.Limiter[string]
}

func New(store *db.Store, authHandler *auth.Handler) *Handler {
	return &Handler{
		Store:   store,
		Auth:    authHandler,
		Now:     time.Now,
		limiter: ratelimit.New[string](startsPerClient, startWindow),
	}
}

type startRequest struct {
	// TimeZone is the browser's IANA zone, so the sample events fall on the
	// visitor's own days. It falls back to UTC.
	TimeZone string `json:"time_zone"`
}

type startResponse struct {
	ID uuid.UUID `json:"id"`
}

// Start creates a demo account with sample data and signs the visitor in.
func (h *Handler) Start(w http.ResponseWriter, r *http.Request) {
	if !h.limiter.Allow(middleware.ClientIP(r)) {
		http.Error(w, auth.ErrTooManyAttempts.Error(), http.StatusTooManyRequests)
		return
	}
	var request startRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, auth.ErrInvalidRequestBody.Error(), http.StatusBadRequest)
		return
	}
	zone, err := timeline.ParseTimeZone(request.TimeZone)
	if err != nil {
		zone = time.UTC
	}

	live, err := h.Store.CountLiveDemoUsers(r.Context())
	if err != nil {
		h.serverError(w, "counting demo users", err)
		return
	}
	if live >= maxLiveUsers {
		http.Error(w, "the demo is full, try again later", http.StatusServiceUnavailable)
		return
	}

	now := h.Now()
	expiresAt := now.Add(lifetime)
	userID, err := h.Store.CreateDemoUser(r.Context(), newEmail(), displayName, expiresAt)
	if err != nil {
		h.serverError(w, "creating a demo user", err)
		return
	}
	// A failed seed leaves a partial account, which expires like any other.
	if err := seed(r.Context(), h.Store, userID, now.In(zone)); err != nil {
		h.serverError(w, "seeding a demo user", err)
		return
	}
	if err := h.Auth.StartSession(w, r, userID, expiresAt); err != nil {
		h.serverError(w, "starting a demo session", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(startResponse{ID: userID})
}

func (h *Handler) serverError(w http.ResponseWriter, action string, err error) {
	log.Printf("%s: %v", action, err)
	http.Error(w, "failed to start the demo", http.StatusInternalServerError)
}

// newEmail returns an address no one can guess or receive mail at, so no
// real user can share with or add a demo user.
func newEmail() string {
	return "demo-" + strings.ToLower(rand.Text()) + "@demo.invalid"
}
