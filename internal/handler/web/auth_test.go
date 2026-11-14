package web_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"gofolio/internal/service"
)

func newRouterWithAdmin(t *testing.T) http.Handler {
	t.Helper()
	db := newHandlerTestDB(t)
	if _, err := service.CreateUser(db, "admin", "admin@localhost", "password", true); err != nil {
		t.Fatalf("seeding admin: %v", err)
	}
	return newTestRouter(db)
}

func TestAuthLoginSuccess(t *testing.T) {
	h := newRouterWithAdmin(t)

	rec := doJSON(t, h, http.MethodPost, "/api/auth/login/",
		map[string]string{"username": "admin", "password": "password"})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if len(rec.Result().Cookies()) == 0 {
		t.Error("successful login did not set a session cookie")
	}
}

func TestAuthLoginBadCredentials(t *testing.T) {
	h := newRouterWithAdmin(t)

	rec := doJSON(t, h, http.MethodPost, "/api/auth/login/",
		map[string]string{"username": "admin", "password": "wrong"})
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestAuthCheckUnauthenticated(t *testing.T) {
	h := newTestRouter(newHandlerTestDB(t))

	rec := doJSON(t, h, http.MethodGet, "/api/auth/check/", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var got map[string]bool
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if got["authenticated"] {
		t.Error("check reports authenticated without a session")
	}
}

func TestAuthLogout(t *testing.T) {
	h := newTestRouter(newHandlerTestDB(t))

	rec := doJSON(t, h, http.MethodGet, "/api/auth/logout/", nil)
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}
