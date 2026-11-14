package web_test

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"gofolio/internal/handler/web"
	"gofolio/internal/service"
	"gofolio/pkg/util"
)

// changePasswordRouter seeds a real user through the service layer and
// returns the router plus the user's id (formatted for the session, which
// stores user_id as a string regardless of the column type).
func changePasswordRouter(t *testing.T, db *sql.DB) http.Handler {
	t.Helper()
	if _, err := service.CreateUser(db, "alice", "alice@localhost", "oldpassword", false); err != nil {
		t.Fatalf("seeding user: %v", err)
	}
	a := &web.Admin{DB: db, Tpl: testRenderer, Sess: testSess}
	r := chi.NewRouter()
	r.Post("/admin/settings/password/", a.ChangePassword)
	return r
}

// authedCookie produces a response carrying a valid session cookie for the
// given user id.
func authedCookie(t *testing.T, userID string) *http.Response {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	if err := testSess.SetSession(rec, req, userID, true); err != nil {
		t.Fatalf("SetSession: %v", err)
	}
	return rec.Result()
}

func TestChangePasswordRoundTrip(t *testing.T) {
	db := newHandlerTestDB(t)
	r := changePasswordRouter(t, db)
	resp := authedCookie(t, "alice")

	form := strings.NewReader("new_password=newpassword&confirm_password=newpassword")
	req := httptest.NewRequest(http.MethodPost, "/admin/settings/password/", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	for _, c := range resp.Cookies() {
		req.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if body := rec.Body.String(); !strings.Contains(body, "Password updated.") {
		t.Errorf("success fragment missing; body: %s", body)
	}

	var hash string
	if err := db.QueryRow(`SELECT password_hash FROM users WHERE username = 'alice'`).
		Scan(&hash); err != nil {
		t.Fatalf("reading new hash: %v", err)
	}
	if !util.CheckPasswordHash("newpassword", hash) {
		t.Error("stored hash does not verify against the new password")
	}
	if util.CheckPasswordHash("oldpassword", hash) {
		t.Error("old password still verifies after change")
	}
}

func TestChangePasswordValidation(t *testing.T) {
	db := newHandlerTestDB(t)
	r := changePasswordRouter(t, db)
	resp := authedCookie(t, "alice")

	post := func(body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/admin/settings/password/",
			strings.NewReader(body))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("HX-Request", "true")
		for _, c := range resp.Cookies() {
			req.AddCookie(c)
		}
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		return rec
	}

	if rec := post("new_password=short&confirm_password=short"); rec.Code != http.StatusBadRequest {
		t.Errorf("short password: status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if rec := post("new_password=firstpass&confirm_password=secondpass"); rec.Code != http.StatusBadRequest {
		t.Errorf("mismatched passwords: status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}
