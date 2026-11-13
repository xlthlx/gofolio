package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

const testSecret = "test-secret-key-for-unit-tests"

var testSessions = NewSessions([]byte(testSecret))

func newSessionRequest(t *testing.T) (*httptest.ResponseRecorder, *http.Request) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	return rec, req
}

func TestSetSessionRoundTrip(t *testing.T) {
	rec, req := newSessionRequest(t)
	if err := testSessions.SetSession(rec, req, "alice", true); err != nil {
		t.Fatalf("SetSession returned error: %v", err)
	}

	cookies := rec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("SetSession did not set any cookie")
	}

	found := false
	for _, c := range cookies {
		if c.Name == SessionName {
			found = true
			if !c.HttpOnly {
				t.Error("session cookie should be HttpOnly")
			}
			if c.MaxAge != SessionMaxAge {
				t.Errorf("session cookie MaxAge = %d, want %d", c.MaxAge, SessionMaxAge)
			}
		}
	}
	if !found {
		t.Errorf("expected cookie %q, got cookies: %v", SessionName, cookies)
	}
}

func TestIsAuthenticatedWithoutSession(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if testSessions.IsAuthenticated(req) {
		t.Error("IsAuthenticated = true for a request with no session cookie, want false")
	}
}

func TestIsAuthenticatedWithSession(t *testing.T) {
	rec, req := newSessionRequest(t)
	if err := testSessions.SetSession(rec, req, "alice", true); err != nil {
		t.Fatalf("SetSession returned error: %v", err)
	}

	authReq := httptest.NewRequest(http.MethodGet, "/", nil)
	for _, c := range rec.Result().Cookies() {
		authReq.AddCookie(c)
	}

	if !testSessions.IsAuthenticated(authReq) {
		t.Error("IsAuthenticated = false after SetSession, want true")
	}
}

func TestClearSession(t *testing.T) {
	rec, req := newSessionRequest(t)
	if err := testSessions.SetSession(rec, req, "alice", true); err != nil {
		t.Fatalf("SetSession returned error: %v", err)
	}

	clearRec := httptest.NewRecorder()
	if err := testSessions.ClearSession(clearRec, req); err != nil {
		t.Fatalf("ClearSession returned error: %v", err)
	}

	var maxAge int
	found := false
	for _, c := range clearRec.Result().Cookies() {
		if c.Name == SessionName {
			found = true
			maxAge = c.MaxAge
		}
	}
	if !found {
		t.Fatal("ClearSession did not set an expiry cookie")
	}
	if maxAge >= 0 {
		t.Errorf("ClearSession cookie MaxAge = %d, want negative", maxAge)
	}
}

func TestGenerateSecureToken(t *testing.T) {
	token1, err := GenerateSecureToken()
	if err != nil {
		t.Fatalf("GenerateSecureToken returned error: %v", err)
	}
	token2, err := GenerateSecureToken()
	if err != nil {
		t.Fatalf("GenerateSecureToken returned error: %v", err)
	}

	if len(token1) != TokenLength*2 { // hex encoding doubles the length
		t.Errorf("token length = %d, want %d", len(token1), TokenLength*2)
	}
	if token1 == token2 {
		t.Error("two generated tokens are identical")
	}
}

func TestRequireAuthRedirectsUnauthenticated(t *testing.T) {
	handler := RequireAuth(testSessions)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler should not be called for unauthenticated request")
	}))

	req := httptest.NewRequest(http.MethodGet, "/admin/dashboard/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if loc := rec.Header().Get("Location"); loc != "/admin/login/" {
		t.Errorf("redirect Location = %q, want %q", loc, "/admin/login/")
	}
}

func TestRequireAuthPassesAuthenticated(t *testing.T) {
	loginRec, loginReq := newSessionRequest(t)
	if err := testSessions.SetSession(loginRec, loginReq, "alice", true); err != nil {
		t.Fatalf("SetSession returned error: %v", err)
	}

	handler := RequireAuth(testSessions)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/admin/dashboard/", nil)
	for _, c := range loginRec.Result().Cookies() {
		req.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}
