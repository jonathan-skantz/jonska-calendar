package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/mail"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/alexedwards/argon2id"
	"github.com/google/uuid"

	"github.com/jonathan-skantz/calendar/backend/db"
	"github.com/jonathan-skantz/calendar/backend/internal/middleware"
	"github.com/jonathan-skantz/calendar/backend/internal/ratelimit"
)

var (
	ErrInvalidRequestBody    = errors.New("invalid request body")
	ErrEmailCannotBeEmpty    = errors.New("email cannot be empty")
	ErrPasswordCannotBeEmpty = errors.New("password cannot be empty")
	ErrEmailTooLong          = errors.New("email is too long")
	ErrInvalidEmail          = errors.New("email is not a valid address")
	ErrPasswordTooShort      = errors.New("password must be at least 8 characters")
	ErrPasswordTooLong       = errors.New("password must be at most 256 bytes")
	ErrInvalidCredentials    = errors.New("invalid credentials")
	ErrSessionTokenMissing   = errors.New("session token missing")
	ErrTooManyAttempts       = errors.New("too many attempts, try again later")
)

const (
	sessionCookieName = "calendar_session"
	sessionDuration   = 30 * 24 * time.Hour
)

// Login is limited per client, not per email, so a stranger cannot lock
// someone out by failing logins with their email.
const (
	loginsPerClient        = 20
	loginWindow            = 15 * time.Minute
	registrationsPerClient = 5
	registrationWindow     = time.Hour
)

type Handler struct {
	Store *db.Store
	// secureCookies sends the session cookie over HTTPS only.
	secureCookies bool

	loginsByClient        *ratelimit.Limiter[string]
	registrationsByClient *ratelimit.Limiter[string]
}

func New(store *db.Store, secureCookies bool) *Handler {
	return &Handler{
		Store:                 store,
		secureCookies:         secureCookies,
		loginsByClient:        ratelimit.New[string](loginsPerClient, loginWindow),
		registrationsByClient: ratelimit.New[string](registrationsPerClient, registrationWindow),
	}
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type registerResponse struct {
	ID uuid.UUID `json:"id"`
}

type currentUserResponse struct {
	ID          uuid.UUID `json:"id"`
	Email       string    `json:"email"`
	DisplayName *string   `json:"display_name,omitempty"`
}

type updateCurrentUserRequest struct {
	DisplayName string `json:"display_name"`
}

const (
	maxDisplayNameLength = 60
	maxEmailLength       = 254
	minPasswordLength    = 8
	// Hashing cost grows with the password, so its length is capped.
	maxPasswordBytes = 256
)

// ParseRegisterRequest reads the credentials of a register or login request.
// Register checks them further with validateNewAccount.
func ParseRegisterRequest(body io.Reader) (*registerRequest, error) {
	var req registerRequest
	if err := json.NewDecoder(body).Decode(&req); err != nil {
		return nil, ErrInvalidRequestBody
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if req.Email == "" {
		return nil, ErrEmailCannotBeEmpty
	}
	if len(req.Email) > maxEmailLength {
		return nil, ErrEmailTooLong
	}
	if req.Password == "" {
		return nil, ErrPasswordCannotBeEmpty
	}
	if len(req.Password) > maxPasswordBytes {
		return nil, ErrPasswordTooLong
	}
	return &req, nil
}

// validateNewAccount applies the rules for new accounts only, so accounts
// created before them can still log in.
func validateNewAccount(req *registerRequest) error {
	address, err := mail.ParseAddress(req.Email)
	if err != nil || address.Address != req.Email {
		return ErrInvalidEmail
	}
	if utf8.RuneCountInString(req.Password) < minPasswordLength {
		return ErrPasswordTooShort
	}
	return nil
}

// Each Argon2 hash takes 64 MiB of memory, so only a few run at once. This
// keeps a burst of logins from exhausting the server's memory.
var hashSlots = make(chan struct{}, 4)

func withHashSlot[T any](r *http.Request, hash func() (T, error)) (T, error) {
	select {
	case hashSlots <- struct{}{}:
		defer func() { <-hashSlots }()
		return hash()
	case <-r.Context().Done():
		var zero T
		return zero, r.Context().Err()
	}
}

// dummyHash is compared against when a login names an unknown email, so the
// response takes as long as for a real account and does not reveal which
// emails are registered.
var dummyHash = sync.OnceValue(func() string {
	hash, err := argon2id.CreateHash("not-a-real-password", argon2id.DefaultParams)
	if err != nil {
		panic(err)
	}
	return hash
})

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	if !h.registrationsByClient.Allow(middleware.ClientIP(r)) {
		http.Error(w, ErrTooManyAttempts.Error(), http.StatusTooManyRequests)
		return
	}

	req, err := ParseRegisterRequest(r.Body)

	if err == nil {
		err = validateNewAccount(req)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	hash, err := withHashSlot(r, func() (string, error) {
		return argon2id.CreateHash(req.Password, argon2id.DefaultParams)
	})
	if err != nil {
		log.Printf("Failed to create hash: %v", err)
		http.Error(w, "Failed to create hash", http.StatusInternalServerError)
		return
	}

	userID, err := h.Store.Register(r.Context(), req.Email, hash)
	if err != nil {
		log.Printf("Failed to register user: %v", err)

		if errors.Is(err, db.ErrEmailAlreadyExists) {
			http.Error(w, "Email already exists", http.StatusConflict)
			return
		}
		http.Error(w, "Failed to register user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	response := registerResponse{ID: userID}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode response (but user created): %v", err)
		return
	}

}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	if !h.loginsByClient.Allow(middleware.ClientIP(r)) {
		http.Error(w, ErrTooManyAttempts.Error(), http.StatusTooManyRequests)
		return
	}
	req, err := ParseRegisterRequest(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	user, err := h.Store.FindUserByEmail(r.Context(), req.Email)
	known := err == nil
	if errors.Is(err, db.ErrUserNotFound) {
		user.PasswordHash = dummyHash()
	} else if err != nil {
		log.Printf("Failed to find user: %v", err)
		http.Error(w, "Failed to log in", http.StatusInternalServerError)
		return
	}

	match, err := withHashSlot(r, func() (bool, error) {
		return argon2id.ComparePasswordAndHash(req.Password, user.PasswordHash)
	})
	if err != nil || !match || !known {
		http.Error(w, ErrInvalidCredentials.Error(), http.StatusUnauthorized)
		return
	}

	token := make([]byte, 32)
	if _, err := rand.Read(token); err != nil {
		log.Printf("Failed to create session token: %v", err)
		http.Error(w, "Failed to log in", http.StatusInternalServerError)
		return
	}
	expiresAt := time.Now().Add(sessionDuration)
	tokenHash := sha256.Sum256(token)
	if err := h.Store.CreateSession(r.Context(), user.ID, tokenHash[:], expiresAt); err != nil {
		log.Printf("Failed to create session: %v", err)
		http.Error(w, "Failed to log in", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    base64.RawURLEncoding.EncodeToString(token),
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   h.secureCookies,
		SameSite: http.SameSiteLaxMode,
	})
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(registerResponse{ID: user.ID})
}

// CurrentUser resolves the signed-in user's ID for a request. Handlers in
// other packages take one, so tests can stub authentication.
type CurrentUser func(*http.Request) (uuid.UUID, error)

// RequireUser returns the signed-in user's ID, or answers 401 and returns
// false when there is none.
func RequireUser(w http.ResponseWriter, r *http.Request, currentUser CurrentUser) (uuid.UUID, bool) {
	userID, err := currentUser(r)
	if err != nil {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return uuid.Nil, false
	}
	return userID, true
}

// CurrentUserID is the CurrentUser for real requests, read from the session
// cookie.
func (h *Handler) CurrentUserID(r *http.Request) (uuid.UUID, error) {
	user, err := h.UserFromRequest(r)
	return user.ID, err
}

func (h *Handler) UserFromRequest(r *http.Request) (db.User, error) {
	tokenHash, err := sessionTokenHash(r)
	if err != nil {
		return db.User{}, err
	}
	return h.Store.FindUserBySession(r.Context(), tokenHash)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	tokenHash, err := sessionTokenHash(r)
	if err == nil {
		if err := h.Store.RevokeSession(r.Context(), tokenHash); err != nil {
			log.Printf("Failed to revoke session: %v", err)
			http.Error(w, "Failed to log out", http.StatusInternalServerError)
			return
		}
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.secureCookies,
		SameSite: http.SameSiteLaxMode,
	})
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) CurrentUser(w http.ResponseWriter, r *http.Request) {
	user, err := h.UserFromRequest(r)
	if err != nil {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(currentUserResponse{ID: user.ID, Email: user.Email, DisplayName: user.DisplayName})
}

func (h *Handler) UpdateCurrentUser(w http.ResponseWriter, r *http.Request) {
	user, err := h.UserFromRequest(r)
	if err != nil {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	var req updateCurrentUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, ErrInvalidRequestBody.Error(), http.StatusBadRequest)
		return
	}
	var displayName *string
	if name := strings.TrimSpace(req.DisplayName); name != "" {
		if utf8.RuneCountInString(name) > maxDisplayNameLength {
			http.Error(w, "display name is too long", http.StatusBadRequest)
			return
		}
		displayName = &name
	}
	if err := h.Store.SetDisplayName(r.Context(), user.ID, displayName); err != nil {
		log.Printf("Failed to update display name: %v", err)
		http.Error(w, "Failed to update user", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func sessionTokenHash(r *http.Request) ([]byte, error) {
	// The browser holds the raw token; only its SHA-256 digest is stored in
	// PostgreSQL, so a database leak does not directly expose active cookies.
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return nil, ErrSessionTokenMissing
	}
	token, err := base64.RawURLEncoding.DecodeString(cookie.Value)
	if err != nil || len(token) != 32 {
		return nil, ErrInvalidCredentials
	}
	tokenHash := sha256.Sum256(token)
	return tokenHash[:], nil
}
