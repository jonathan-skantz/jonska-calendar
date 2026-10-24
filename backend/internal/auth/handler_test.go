package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/alexedwards/argon2id"
	"github.com/google/uuid"
	"github.com/jonathan-skantz/calendar/backend/db"
	"github.com/jonathan-skantz/calendar/backend/internal/testutil"
)

func TestRegister(t *testing.T) {
	store := testutil.NewStore(t)
	handler := New(store, false)

	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBufferString(`{"email":" TEST@example.com ","password":"password123"}`))
	response := httptest.NewRecorder()

	handler.Register(response, req)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, response.Code, response.Body.String())
	}

	var payload registerResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.ID == uuid.Nil {
		t.Fatal("expected a user UUID")
	}

	user, err := store.FindUserByEmail(context.Background(), "test@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if user.ID != payload.ID {
		t.Fatalf("response ID %s does not match stored ID %s", payload.ID, user.ID)
	}
	if user.Email != "test@example.com" {
		t.Fatalf("expected normalized email, got %q", user.Email)
	}
	if user.PasswordHash == "password123" {
		t.Fatal("password was stored in plaintext")
	}
	if match, err := argon2id.ComparePasswordAndHash("password123", user.PasswordHash); err != nil || !match {
		t.Fatalf("stored password hash did not verify: %v", err)
	}
}

func TestRegisterRejectsDuplicateEmail(t *testing.T) {
	store := testutil.NewStore(t)
	handler := New(store, false)

	register := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBufferString(`{"email":"test@example.com","password":"password123"}`))
		response := httptest.NewRecorder()
		handler.Register(response, req)
		return response
	}

	if response := register(); response.Code != http.StatusCreated {
		t.Fatalf("first registration: expected status %d, got %d", http.StatusCreated, response.Code)
	}
	if response := register(); response.Code != http.StatusConflict {
		t.Fatalf("duplicate registration: expected status %d, got %d", http.StatusConflict, response.Code)
	}
}

func TestLoginCreatesSession(t *testing.T) {
	store := testutil.NewStore(t)
	handler := New(store, false)

	registerRequest := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBufferString(`{"email":"test@example.com","password":"password123"}`))
	registerResponse := httptest.NewRecorder()
	handler.Register(registerResponse, registerRequest)
	if registerResponse.Code != http.StatusCreated {
		t.Fatalf("registration: expected status %d, got %d", http.StatusCreated, registerResponse.Code)
	}

	loginRequest := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(`{"email":"TEST@example.com","password":"password123"}`))
	loginResponse := httptest.NewRecorder()
	handler.Login(loginResponse, loginRequest)
	if loginResponse.Code != http.StatusOK {
		t.Fatalf("login: expected status %d, got %d: %s", http.StatusOK, loginResponse.Code, loginResponse.Body.String())
	}

	cookie := loginResponse.Result().Cookies()
	if len(cookie) != 1 || cookie[0].HttpOnly == false {
		t.Fatal("expected an HTTP-only session cookie")
	}
	if cookie[0].Secure {
		t.Fatal("expected an insecure cookie when secure cookies are off")
	}

	currentRequest := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	currentRequest.AddCookie(cookie[0])
	user, err := handler.UserFromRequest(currentRequest)
	if err != nil {
		t.Fatal(err)
	}
	if user.Email != "test@example.com" {
		t.Fatalf("expected logged-in user email, got %q", user.Email)
	}
}

func TestLoginRejectsInvalidCredentials(t *testing.T) {
	store := testutil.NewStore(t)
	handler := New(store, false)

	request := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(`{"email":"missing@example.com","password":"password123"}`))
	response := httptest.NewRecorder()
	handler.Login(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, response.Code)
	}
}

func TestLogoutRevokesSession(t *testing.T) {
	store := testutil.NewStore(t)
	handler := New(store, false)

	registerRequest := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBufferString(`{"email":"test@example.com","password":"password123"}`))
	registerResponse := httptest.NewRecorder()
	handler.Register(registerResponse, registerRequest)
	if registerResponse.Code != http.StatusCreated {
		t.Fatalf("registration: expected status %d, got %d", http.StatusCreated, registerResponse.Code)
	}

	loginRequest := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(`{"email":"test@example.com","password":"password123"}`))
	loginResponse := httptest.NewRecorder()
	handler.Login(loginResponse, loginRequest)
	if loginResponse.Code != http.StatusOK {
		t.Fatalf("login: expected status %d, got %d", http.StatusOK, loginResponse.Code)
	}
	cookies := loginResponse.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatal("expected session cookie")
	}

	logoutRequest := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	logoutRequest.AddCookie(cookies[0])
	logoutResponse := httptest.NewRecorder()
	handler.Logout(logoutResponse, logoutRequest)
	if logoutResponse.Code != http.StatusNoContent {
		t.Fatalf("logout: expected status %d, got %d", http.StatusNoContent, logoutResponse.Code)
	}
	logoutCookies := logoutResponse.Result().Cookies()
	if len(logoutCookies) != 1 || logoutCookies[0].MaxAge != -1 {
		t.Fatal("expected expired session cookie")
	}

	currentRequest := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	currentRequest.AddCookie(cookies[0])
	if _, err := handler.UserFromRequest(currentRequest); !errors.Is(err, db.ErrSessionNotFound) {
		t.Fatalf("expected revoked session, got %v", err)
	}
}

func TestSessionCookieIsSecureWhenConfigured(t *testing.T) {
	store := testutil.NewStore(t)
	handler := New(store, true)

	registerRequest := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBufferString(`{"email":"test@example.com","password":"password123"}`))
	registerResponse := httptest.NewRecorder()
	handler.Register(registerResponse, registerRequest)
	if registerResponse.Code != http.StatusCreated {
		t.Fatalf("registration: expected status %d, got %d", http.StatusCreated, registerResponse.Code)
	}

	loginRequest := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(`{"email":"test@example.com","password":"password123"}`))
	loginResponse := httptest.NewRecorder()
	handler.Login(loginResponse, loginRequest)
	cookies := loginResponse.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].Secure {
		t.Fatal("expected a secure session cookie")
	}

	logoutRequest := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	logoutRequest.AddCookie(cookies[0])
	logoutResponse := httptest.NewRecorder()
	handler.Logout(logoutResponse, logoutRequest)
	logoutCookies := logoutResponse.Result().Cookies()
	if len(logoutCookies) != 1 || !logoutCookies[0].Secure {
		t.Fatal("expected a secure expired session cookie")
	}
}

func TestLoginIsRateLimited(t *testing.T) {
	handler := New(testutil.NewStore(t), false)
	login := func(email, clientIP string) int {
		request := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(`{"email":"`+email+`","password":"password123"}`))
		request.Header.Set("X-Forwarded-For", clientIP)
		response := httptest.NewRecorder()
		handler.Login(response, request)
		return response.Code
	}

	for range loginsPerClient {
		if code := login("someone@example.com", "203.0.113.1"); code != http.StatusUnauthorized {
			t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, code)
		}
	}
	if code := login("third@example.com", "203.0.113.1"); code != http.StatusTooManyRequests {
		t.Fatalf("same client: expected status %d, got %d", http.StatusTooManyRequests, code)
	}
	if code := login("third@example.com", "203.0.113.3"); code != http.StatusUnauthorized {
		t.Fatalf("another client: expected status %d, got %d", http.StatusUnauthorized, code)
	}
}

func TestRegisterIsRateLimited(t *testing.T) {
	handler := New(testutil.NewStore(t), false)
	for i := range registrationsPerClient + 1 {
		request := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBufferString(`{"email":"user`+strconv.Itoa(i)+`@example.com","password":"password123"}`))
		response := httptest.NewRecorder()
		handler.Register(response, request)
		want := http.StatusCreated
		if i == registrationsPerClient {
			want = http.StatusTooManyRequests
		}
		if response.Code != want {
			t.Fatalf("registration %d: expected status %d, got %d", i+1, want, response.Code)
		}
	}
}

func TestCurrentUser(t *testing.T) {
	store := testutil.NewStore(t)
	handler := New(store, false)

	registerRequest := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBufferString(`{"email":"test@example.com","password":"password123"}`))
	registerResponse := httptest.NewRecorder()
	handler.Register(registerResponse, registerRequest)

	loginRequest := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(`{"email":"test@example.com","password":"password123"}`))
	loginResponse := httptest.NewRecorder()
	handler.Login(loginResponse, loginRequest)
	cookies := loginResponse.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatal("expected session cookie")
	}

	request := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	request.AddCookie(cookies[0])
	response := httptest.NewRecorder()
	handler.CurrentUser(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("current user: expected status %d, got %d", http.StatusOK, response.Code)
	}
	var user currentUserResponse
	if err := json.NewDecoder(response.Body).Decode(&user); err != nil {
		t.Fatal(err)
	}
	if user.Email != "test@example.com" || user.ID == uuid.Nil {
		t.Fatalf("unexpected current user: %+v", user)
	}
	if strings.Contains(response.Body.String(), "password") {
		t.Fatal("current user response exposed password data")
	}

	unauthenticatedResponse := httptest.NewRecorder()
	handler.CurrentUser(unauthenticatedResponse, httptest.NewRequest(http.MethodGet, "/api/auth/me", nil))
	if unauthenticatedResponse.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated current user: expected status %d, got %d", http.StatusUnauthorized, unauthenticatedResponse.Code)
	}
}

func TestParseRegisterRequestValidation(t *testing.T) {
	tests := []struct {
		name string
		body string
		want error
	}{
		{name: "invalid json", body: "{", want: ErrInvalidRequestBody},
		{name: "empty email", body: `{"email":"","password":"password123"}`, want: ErrEmailCannotBeEmpty},
		{name: "empty password", body: `{"email":"test@example.com","password":""}`, want: ErrPasswordCannotBeEmpty},
		{name: "email too long", body: `{"email":"` + strings.Repeat("a", 250) + `@b.se","password":"password123"}`, want: ErrEmailTooLong},
		{name: "password too long", body: `{"email":"test@example.com","password":"` + strings.Repeat("a", 257) + `"}`, want: ErrPasswordTooLong},
		{name: "short password still parses for login", body: `{"email":"test@example.com","password":"short"}`, want: nil},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := ParseRegisterRequest(bytes.NewBufferString(test.body))
			if !errors.Is(err, test.want) {
				t.Fatalf("expected %v, got %v", test.want, err)
			}
		})
	}
}

func TestValidateNewAccount(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		password string
		want     error
	}{
		{name: "valid", email: "test@example.com", password: "password123", want: nil},
		{name: "no at sign", email: "test.example.com", password: "password123", want: ErrInvalidEmail},
		{name: "display name form", email: "test <test@example.com>", password: "password123", want: ErrInvalidEmail},
		{name: "short password", email: "test@example.com", password: "1234567", want: ErrPasswordTooShort},
		{name: "eight characters", email: "test@example.com", password: "åäö12345", want: nil},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateNewAccount(&registerRequest{Email: test.email, Password: test.password})
			if !errors.Is(err, test.want) {
				t.Fatalf("expected %v, got %v", test.want, err)
			}
		})
	}
}

func TestRegisterRejectsShortPassword(t *testing.T) {
	handler := New(testutil.NewStore(t), false)
	response := httptest.NewRecorder()
	handler.Register(response, httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBufferString(`{"email":"test@example.com","password":"short"}`)))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}
}

func TestUpdateCurrentUserDisplayName(t *testing.T) {
	store := testutil.NewStore(t)
	handler := New(store, false)

	registerResponse := httptest.NewRecorder()
	handler.Register(registerResponse, httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBufferString(`{"email":"test@example.com","password":"password123"}`)))
	loginResponse := httptest.NewRecorder()
	handler.Login(loginResponse, httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(`{"email":"test@example.com","password":"password123"}`)))
	cookies := loginResponse.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatal("expected session cookie")
	}

	patch := func(body string) int {
		request := httptest.NewRequest(http.MethodPatch, "/api/auth/me", bytes.NewBufferString(body))
		request.AddCookie(cookies[0])
		response := httptest.NewRecorder()
		handler.UpdateCurrentUser(response, request)
		return response.Code
	}
	currentDisplayName := func() *string {
		request := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
		request.AddCookie(cookies[0])
		response := httptest.NewRecorder()
		handler.CurrentUser(response, request)
		var body currentUserResponse
		if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		return body.DisplayName
	}

	if code := patch(`{"display_name":"  Jonathan  "}`); code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, code)
	}
	if name := currentDisplayName(); name == nil || *name != "Jonathan" {
		t.Fatalf("expected trimmed display name, got %v", name)
	}
	if code := patch(`{"display_name":"` + strings.Repeat("x", maxDisplayNameLength+1) + `"}`); code != http.StatusBadRequest {
		t.Fatalf("expected status %d for long name, got %d", http.StatusBadRequest, code)
	}
	if code := patch(`{"display_name":""}`); code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, code)
	}
	if name := currentDisplayName(); name != nil {
		t.Fatalf("expected cleared display name, got %q", *name)
	}

	unauthenticated := httptest.NewRecorder()
	handler.UpdateCurrentUser(unauthenticated, httptest.NewRequest(http.MethodPatch, "/api/auth/me", bytes.NewBufferString(`{}`)))
	if unauthenticated.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, unauthenticated.Code)
	}
}
