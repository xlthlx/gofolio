package integration_test

import (
	"bytes"
	"encoding/json"
	web "gofolio/internal/handler/http"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"gofolio/internal/core"
	"gofolio/internal/handler/middleware"
	"gofolio/internal/service"
	"gofolio/internal/store"
)

// newTestServer boots the full stack (router + middleware + SQLite) and
// returns a client that stores cookies but does not follow redirects, so
// tests can assert exact status codes.
func newTestServer(t *testing.T) (string, *http.Client) {
	t.Helper()

	db, err := store.Open(filepath.Join(t.TempDir(), "integration.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	if err := store.Migrate(db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if err := service.InitUsersTable(db); err != nil {
		t.Fatalf("InitUsersTable: %v", err)
	}
	if err := service.CreateDefaultAdmin(db); err != nil {
		t.Fatalf("CreateDefaultAdmin: %v", err)
	}
	sess := middleware.NewSessions([]byte("integration-test-secret"))

	tpl := web.MustRenderer(web.FindAdminThemeDir())
	ts := httptest.NewServer(core.New(db, tpl, sess))
	t.Cleanup(ts.Close)

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookiejar: %v", err)
	}
	client := &http.Client{Jar: jar}
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return ts.URL, client
}

func call(t *testing.T, client *http.Client, method, url string, body interface{}) *http.Response {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		reader = bytes.NewReader(b)
	} else {
		reader = bytes.NewReader(nil)
	}
	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	t.Cleanup(func() { resp.Body.Close() })
	return resp
}

func decode(t *testing.T, resp *http.Response, v interface{}) {
	t.Helper()
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
}

func TestAdminRequiresAuth(t *testing.T) {
	base, client := newTestServer(t)

	resp := call(t, client, http.MethodGet, base+"/admin/api/posts/", nil)
	if resp.StatusCode != http.StatusSeeOther {
		t.Errorf("unauthenticated admin API: status = %d, want %d", resp.StatusCode, http.StatusSeeOther)
	}
	if loc := resp.Header.Get("Location"); loc != "/admin/login/" {
		t.Errorf("Location = %q, want /admin/login/", loc)
	}
}

func TestFullPostLifecycle(t *testing.T) {
	base, client := newTestServer(t)

	// Login failure
	resp := call(t, client, http.MethodPost, base+"/api/auth/login/",
		map[string]string{"username": "admin", "password": "nope"})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("bad login: status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}

	// Login success (sets session cookie in the jar)
	resp = call(t, client, http.MethodPost, base+"/api/auth/login/",
		map[string]string{"username": "admin", "password": "password"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login: status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	// Auth check
	var check map[string]bool
	decode(t, call(t, client, http.MethodGet, base+"/api/auth/check/", nil), &check)
	if !check["authenticated"] {
		t.Error("auth check after login = false, want true")
	}

	// Create a published post
	var created struct {
		ID     int64  `json:"id"`
		Slug   string `json:"slug"`
		Status string `json:"status"`
	}
	decode(t, call(t, client, http.MethodPost, base+"/admin/api/posts/",
		map[string]string{"title": "Launch Day", "status": "published"}), &created)
	if created.Slug != "launch-day" || created.Status != "published" {
		t.Fatalf("created post = %+v, want slug launch-day, status published", created)
	}

	// Visible on the public homepage
	var home struct {
		Posts []map[string]interface{} `json:"posts"`
	}
	decode(t, call(t, client, http.MethodGet, base+"/", nil), &home)
	if len(home.Posts) != 1 {
		t.Fatalf("homepage posts = %d, want 1", len(home.Posts))
	}

	// Fetchable by slug
	var fetched map[string]interface{}
	decode(t, call(t, client, http.MethodGet, base+"/posts/launch-day/", nil), &fetched)
	if fetched["title"] != "Launch Day" {
		t.Errorf("fetched title = %v, want Launch Day", fetched["title"])
	}

	// Delete it
	resp = call(t, client, http.MethodDelete, base+"/admin/api/posts/"+jsonNum(created.ID)+"/", nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("delete: status = %d, want %d", resp.StatusCode, http.StatusNoContent)
	}

	// Gone from the public API
	resp = call(t, client, http.MethodGet, base+"/posts/launch-day/", nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("after delete: status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func jsonNum(n int64) string {
	return toJSONNumber(n)
}

func toJSONNumber(n int64) string {
	b, _ := json.Marshal(n)
	return string(b)
}
