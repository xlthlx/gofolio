package web_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"gofolio/internal/handler/web"
	"gofolio/internal/store"
)

func TestSiteSettingsRoundTrip(t *testing.T) {
	db := newHandlerTestDB(t)
	a := &web.Admin{DB: db, Tpl: testRenderer, Sess: testSess}
	r := chi.NewRouter()
	r.Get("/admin/settings/site/", a.SiteSettingsGet)
	r.Post("/admin/settings/site/", a.SiteSettingsPost)

	req := httptest.NewRequest(http.MethodPost, "/admin/settings/site/",
		strings.NewReader("site_title=Gopher+Works&tagline=Build+things"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("POST settings: status = %d", rec.Code)
	}

	// Persisted?
	ss := store.SettingStore{DB: db}
	title, err := ss.Get("site_title")
	if err != nil || title != "Gopher Works" {
		t.Errorf("site_title = %q, err %v; want Gopher Works", title, err)
	}

	// Rendered back with values?
	rec = doJSON(t, r, http.MethodGet, "/admin/settings/site/", nil)
	if !strings.Contains(rec.Body.String(), "Gopher Works") {
		t.Error("settings page does not render saved site_title")
	}
}

func TestSiteSettingsRejectsUnknownKeys(t *testing.T) {
	db := newHandlerTestDB(t)
	a := &web.Admin{DB: db, Tpl: testRenderer, Sess: testSess}
	r := chi.NewRouter()
	r.Post("/admin/settings/site/", a.SiteSettingsPost)

	req := httptest.NewRequest(http.MethodPost, "/admin/settings/site/",
		strings.NewReader("evil_key=pwned"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	ss := store.SettingStore{DB: db}
	all, err := ss.All()
	if err != nil {
		t.Fatalf("All: %v", err)
	}
	if _, exists := all["evil_key"]; exists {
		t.Error("unknown key persisted despite whitelist")
	}
}
