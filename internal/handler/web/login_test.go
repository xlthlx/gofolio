package web_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"gofolio/internal/handler/web"
	"gofolio/internal/service"
)

func TestLoginGetRendersPage(t *testing.T) {
	h := newTestRouter(newHandlerTestDB(t))

	rec := doJSON(t, h, http.MethodGet, "/admin/login/", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if body := rec.Body.String(); !strings.Contains(body, `hx-post="/admin/login/"`) {
		t.Errorf("login page does not contain hx-post form; body:\n%s", body)
	}
}

func TestLoginPostHTMXSuccess(t *testing.T) {
	db := newHandlerTestDB(t)
	if _, err := service.CreateUser(db, "admin", "admin@localhost", "password", true); err != nil {
		t.Fatalf("seeding admin: %v", err)
	}

	a := &web.Admin{DB: db, Tpl: testRenderer, Sess: testSess}
	r := chi.NewRouter()
	r.Post("/admin/login/", a.LoginPost)

	req := httptest.NewRequest(http.MethodPost, "/admin/login/",
		strings.NewReader("username=admin&password=password"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if loc := rec.Header().Get("HX-Redirect"); loc != "/admin/" {
		t.Errorf("HX-Redirect = %q, want /admin/", loc)
	}
	if len(rec.Result().Cookies()) == 0 {
		t.Error("successful HTMX login did not set a session cookie")
	}
}

func TestLoginPostHTMXFailureReturnsFragment(t *testing.T) {
	db := newHandlerTestDB(t)
	if _, err := service.CreateUser(db, "admin", "admin@localhost", "password", true); err != nil {
		t.Fatalf("seeding admin: %v", err)
	}

	a := &web.Admin{DB: db, Tpl: testRenderer, Sess: testSess}
	r := chi.NewRouter()
	r.Post("/admin/login/", a.LoginPost)

	req := httptest.NewRequest(http.MethodPost, "/admin/login/",
		strings.NewReader("username=admin&password=wrong"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if body := rec.Body.String(); !strings.Contains(body, "Invalid username or password.") {
		t.Errorf("error fragment not rendered; body:\n%s", body)
	}
}
