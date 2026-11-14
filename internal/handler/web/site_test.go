package web_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// seedPost creates a post through the admin API and returns its slug and the
// creation time as the API reports it.
func seedPost(t *testing.T, h http.Handler, title, status, content string) (string, string) {
	t.Helper()
	rec := doJSON(t, h, http.MethodPost, "/admin/api/posts/", map[string]string{
		"title":   title,
		"status":  status,
		"content": content,
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("seeding post: status = %d, body: %s", rec.Code, rec.Body.String())
	}
	var created struct {
		Slug      string `json:"slug"`
		CreatedAt string `json:"created_at"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decoding seeded post: %v", err)
	}
	return created.Slug, created.CreatedAt
}

func TestHomePageListsPublishedPosts(t *testing.T) {
	h := newTestRouter(newHandlerTestDB(t))

	seedPost(t, h, "Published Story", "published", "<p>visible</p>")
	seedPost(t, h, "Draft Story", "draft", "<p>hidden</p>")

	rec := doJSON(t, h, http.MethodGet, "/", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("Content-Type = %q, want HTML", ct)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Published Story") {
		t.Error("homepage does not list the published post")
	}
	if strings.Contains(body, "Draft Story") {
		t.Error("homepage lists a draft")
	}
	if !strings.Contains(body, `href="/posts/published-story/"`) {
		t.Error("homepage does not link to the published post")
	}
}

func TestHomePageEmptyState(t *testing.T) {
	h := newTestRouter(newHandlerTestDB(t))

	rec := doJSON(t, h, http.MethodGet, "/", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(rec.Body.String(), "Nothing published yet.") {
		t.Error("empty homepage does not say so")
	}
}

func TestPostPageRendersStoredContent(t *testing.T) {
	h := newTestRouter(newHandlerTestDB(t))
	slug, createdAt := seedPost(t, h, "Launch Day", "published", "<p>Hello <strong>world</strong></p>")

	rec := doJSON(t, h, http.MethodGet, "/posts/"+slug+"/", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Launch Day") {
		t.Error("post page does not show the title")
	}
	if !strings.Contains(body, "<p>Hello <strong>world</strong></p>") {
		t.Error("post page does not render the stored HTML as markup")
	}
	if strings.Contains(body, "&lt;strong&gt;") {
		t.Error("post page escaped the stored HTML")
	}
	if !strings.Contains(body, "<title>Launch Day - goFolio</title>") {
		t.Error("post page does not put the post title in the document title")
	}

	created, err := time.Parse(time.RFC3339, createdAt)
	if err != nil {
		t.Fatalf("parsing created_at %q: %v", createdAt, err)
	}
	if !strings.Contains(body, created.Format("2 January 2006")) {
		t.Errorf("post page does not show the publication date %q", created.Format("2 January 2006"))
	}
}

func TestPostPageHidesDraftsAndUnknownSlugs(t *testing.T) {
	h := newTestRouter(newHandlerTestDB(t))
	slug, _ := seedPost(t, h, "Draft Story", "draft", "<p>hidden</p>")

	for _, path := range []string{"/posts/" + slug + "/", "/posts/no-such-post/"} {
		rec := doJSON(t, h, http.MethodGet, path, nil)
		if rec.Code != http.StatusNotFound {
			t.Errorf("%s: status = %d, want %d", path, rec.Code, http.StatusNotFound)
		}
		body := rec.Body.String()
		if !strings.Contains(body, "Not found") {
			t.Errorf("%s: 404 is not the themed page; body: %s", path, body)
		}
		if !strings.Contains(body, "<!DOCTYPE html>") {
			t.Errorf("%s: 404 has no HTML body, so a boosted navigation would swap nothing", path)
		}
	}
}

func TestPagePageRendersStoredContent(t *testing.T) {
	h := newTestRouter(newHandlerTestDB(t))

	rec := doJSON(t, h, http.MethodPost, "/admin/api/pages/", map[string]string{
		"title":   "About Us",
		"status":  "published",
		"content": "<p>We write things</p>",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("seeding page: status = %d", rec.Code)
	}

	rec = doJSON(t, h, http.MethodGet, "/pages/about-us/", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "About Us") {
		t.Error("page view does not show the title")
	}
	if !strings.Contains(body, "<p>We write things</p>") {
		t.Error("page view does not render the stored HTML as markup")
	}
	if !strings.Contains(body, "<title>About Us - goFolio</title>") {
		t.Error("page view does not put the page title in the document title")
	}
}

func TestUnknownPageIsThemedNotFound(t *testing.T) {
	h := newTestRouter(newHandlerTestDB(t))

	rec := doJSON(t, h, http.MethodGet, "/pages/ghost/", nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
	if !strings.Contains(rec.Body.String(), "Not found") {
		t.Error("unknown page did not render the themed 404")
	}
}

func TestPageViewHidesDrafts(t *testing.T) {
	h := newTestRouter(newHandlerTestDB(t))

	rec := doJSON(t, h, http.MethodPost, "/admin/api/pages/", map[string]string{
		"title":   "Hidden Page",
		"status":  "draft",
		"content": "<p>hidden</p>",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("seeding page: status = %d", rec.Code)
	}

	rec = doJSON(t, h, http.MethodGet, "/pages/hidden-page/", nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("draft page rendered publicly: status = %d, want %d", rec.Code, http.StatusNotFound)
	}
	if strings.Contains(rec.Body.String(), "hidden</p>") {
		t.Error("draft page content leaked into the 404")
	}
}

func TestPublicPagesUseStoredSiteIdentity(t *testing.T) {
	h := newTestRouter(newHandlerTestDB(t))

	rec := doJSON(t, h, http.MethodGet, "/", nil)
	if !strings.Contains(rec.Body.String(), "goFolio") {
		t.Error("homepage does not fall back to the default site title")
	}

	rec = doForm(t, h, "/admin/settings/site/", url.Values{
		"site_title": {"Gopher Works"},
		"tagline":    {"Build things"},
	}, false)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("saving site settings: status = %d", rec.Code)
	}

	rec = doJSON(t, h, http.MethodGet, "/", nil)
	body := rec.Body.String()
	if !strings.Contains(body, "Gopher Works") {
		t.Error("homepage does not show the stored site title")
	}
	if !strings.Contains(body, "Build things") {
		t.Error("homepage does not show the stored tagline")
	}
}

func TestPublicNavigationFollowsTheSession(t *testing.T) {
	h := newTestRouter(newHandlerTestDB(t))

	rec := doJSON(t, h, http.MethodGet, "/", nil)
	body := rec.Body.String()
	if !strings.Contains(body, `href="/admin/login/"`) {
		t.Error("signed-out navigation does not offer a login link")
	}
	if strings.Contains(body, `href="/api/auth/logout/"`) {
		t.Error("signed-out navigation offers a logout link")
	}

	resp := authedCookie(t, "admin")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	for _, c := range resp.Cookies() {
		req.AddCookie(c)
	}
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	body = rec.Body.String()
	if !strings.Contains(body, `href="/admin/"`) {
		t.Error("signed-in navigation does not offer the admin link")
	}
	if !strings.Contains(body, `href="/api/auth/logout/"`) {
		t.Error("signed-in navigation does not offer a logout link")
	}
}
