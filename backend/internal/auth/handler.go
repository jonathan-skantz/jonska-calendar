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
	"strings"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/jonathan-skantz/calendar/backend/db"
)

var (
	ErrInvalidRequestBody    = errors.New("invalid request body")
	ErrEmailCannotBeEmpty    = errors.New("email cannot be empty")
	ErrPasswordCannotBeEmpty = errors.New("password cannot be empty")
	ErrInvalidCredentials    = errors.New("invalid credentials")
	ErrSessionTokenMissing   = errors.New("session token missing")
)

const (
	sessionCookieName = "calendar_session"
	sessionDuration   = 30 * 24 * time.Hour
)

type Handler struct {
	Store *db.Store
}

func New(store *db.Store) *Handler {
	return &Handler{Store: store}
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type registerResponse struct {
	ID uuid.UUID `json:"id"`
}

type currentUserResponse struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
}

func ParseRegisterRequest(body io.Reader) (*registerRequest, error) {
	var req registerRequest
	if err := json.NewDecoder(body).Decode(&req); err != nil {
		return nil, ErrInvalidRequestBody
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if req.Email == "" {
		return nil, ErrEmailCannotBeEmpty
	}
	if req.Password == "" {
		return nil, ErrPasswordCannotBeEmpty
	}
	return &req, nil
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {

	req, err := ParseRegisterRequest(r.Body)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	hash, err := argon2id.CreateHash(req.Password, argon2id.DefaultParams)
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
	req, err := ParseRegisterRequest(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	user, err := h.Store.FindUserByEmail(r.Context(), req.Email)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		log.Printf("Failed to find user: %v", err)
		http.Error(w, "Failed to log in", http.StatusInternalServerError)
		return
	}
	if errors.Is(err, pgx.ErrNoRows) {
		http.Error(w, ErrInvalidCredentials.Error(), http.StatusUnauthorized)
		return
	}

	match, err := argon2id.ComparePasswordAndHash(req.Password, user.PasswordHash)
	if err != nil || !match {
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
		SameSite: http.SameSiteLaxMode,
	})
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(registerResponse{ID: user.ID})
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
	_ = json.NewEncoder(w).Encode(currentUserResponse{ID: user.ID, Email: user.Email})
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
