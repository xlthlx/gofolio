package web_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"gofolio/internal/store"
)

func TestSiteSettingsRoundTrip(t *testing.T) {
	db := newHandlerTestDB(t)
	h := newTestRouter(db)

	// A plain form post lands on the settings page, which is where the values
	// have to be visible next.
	rec := doForm(t, h, "/admin/settings/site/", url.Values{
		"site_title": {"Gopher Works"},
		"tagline":    {"Build things"},
	}, false)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("POST site settings: status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if loc := rec.Header().Get("Location"); loc != "/admin/settings/" {
		t.Errorf("Location = %q, want /admin/settings/", loc)
	}

	ss := store.SettingStore{DB: db}
	title, err := ss.Get("site_title")
	if err != nil || title != "Gopher Works" {
		t.Errorf("site_title = %q, err %v; want Gopher Works", title, err)
	}

	rec = doJSON(t, h, http.MethodGet, "/admin/settings/", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET settings: status = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Gopher Works") {
		t.Error("settings page does not render the saved site_title")
	}
}

func TestSiteSettingsHTMXSwapsSection(t *testing.T) {
	db := newHandlerTestDB(t)
	h := newTestRouter(db)

	rec := doForm(t, h, "/admin/settings/site/", url.Values{
		"site_title": {"Gopher Works"},
		"tagline":    {"Build things"},
	}, true)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; HTMX only swaps successful responses", rec.Code, http.StatusOK)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("Content-Type = %q, want HTML", ct)
	}

	body := rec.Body.String()
	if strings.Contains(body, "<!DOCTYPE html>") {
		t.Error("HTMX response is a whole page instead of the section fragment")
	}
	if !strings.Contains(body, `id="site-settings"`) {
		t.Error("fragment lost the id the form targets, so a second save would go nowhere")
	}
	if !strings.Contains(body, `hx-post="/admin/settings/site/"`) {
		t.Error("swapped-in section does not carry the HTMX attributes")
	}
	if !strings.Contains(body, `value="Gopher Works"`) {
		t.Error("fragment does not show the value it just saved")
	}
	if !strings.Contains(body, "Site title: Gopher Works") {
		t.Error("fragment does not show the saved value as text")
	}
}

func TestSiteSettingsRejectsUnknownKeys(t *testing.T) {
	db := newHandlerTestDB(t)
	h := newTestRouter(db)

	rec := doForm(t, h, "/admin/settings/site/", url.Values{"evil_key": {"pwned"}}, true)
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

// TestSiteSettingsHTMXReportsSaveFailure covers the branch that only runs when
// the write itself fails: the fragment has to carry the message, because HTMX
// swaps it in place of the section.
func TestSiteSettingsHTMXReportsSaveFailure(t *testing.T) {
	db := newHandlerTestDB(t)
	h := newTestRouter(db)
	if err := db.Close(); err != nil {
		t.Fatalf("closing test database: %v", err)
	}

	rec := doForm(t, h, "/admin/settings/site/", url.Values{"site_title": {"Gopher"}}, true)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if body := rec.Body.String(); !strings.Contains(body, "Could not save the settings") {
		t.Errorf("failure fragment missing the message; body: %s", body)
	}
}

func TestSettingsPageWiresSiteSettingsToHTMX(t *testing.T) {
	h := newTestRouter(newHandlerTestDB(t))

	rec := doJSON(t, h, http.MethodGet, "/admin/settings/", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	for _, want := range []string{
		`id="site-settings"`,
		`hx-post="/admin/settings/site/"`,
		`hx-target="#site-settings"`,
		`hx-swap="outerHTML"`,
		`action="/admin/settings/site/"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("settings page is missing %s", want)
		}
	}
}
