package web_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"gofolio/internal/handler/middleware"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	web "gofolio/internal/handler/http"
	"gofolio/internal/service"
	"gofolio/internal/store"
)

func TestMain(m *testing.M) {
	middleware.Init([]byte("unit-test-secret"))
	os.Exit(m.Run())
}

func newHandlerTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if err := store.Migrate(db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	return db
}

var testRenderer = web.MustRenderer(web.FindAdminThemeDir())

func newTestRouter(db *sql.DB) http.Handler {
	a := &web.Admin{DB: db, Tpl: testRenderer}
	p := &web.Public{DB: db}
	r := chi.NewRouter()
	r.Get("/api/posts/", a.ListPosts)
	r.Post("/api/posts/", a.CreatePost)
	r.Put("/api/posts/{id}/", a.UpdatePost)
	r.Delete("/api/posts/{id}/", a.DeletePost)
	r.Get("/posts/{slug}/", p.GetPost)
	r.Get("/admin/posts/table/", a.PostsTable)
	r.Get("/healthz/", (&web.Health{DB: db}).Check)
	r.Get("/admin/dashboard/", a.DashboardPage)
	r.Get("/admin/login/", a.LoginGet)
	return r
}

func doJSON(t *testing.T, h http.Handler, method, path string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encoding request body: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func intToStr(n int64) string {
	return strconv.FormatInt(n, 10)
}

func TestCreatePostRequiresTitle(t *testing.T) {
	h := newTestRouter(newHandlerTestDB(t))

	rec := doJSON(t, h, http.MethodPost, "/api/posts/", map[string]string{})
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestCreatePostDefaultsSlugAndStatus(t *testing.T) {
	h := newTestRouter(newHandlerTestDB(t))

	rec := doJSON(t, h, http.MethodPost, "/api/posts/", map[string]string{"title": "Hello World"})
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var got map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if got["slug"] != "hello-world" {
		t.Errorf("slug = %v, want hello-world", got["slug"])
	}
	if got["status"] != "draft" {
		t.Errorf("status = %v, want draft", got["status"])
	}
}

func TestUpdatePostInvalidID(t *testing.T) {
	h := newTestRouter(newHandlerTestDB(t))

	rec := doJSON(t, h, http.MethodPut, "/api/posts/notanumber/", map[string]string{"title": "x"})
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestDeletePostRoundTrip(t *testing.T) {
	db := newHandlerTestDB(t)
	h := newTestRouter(db)

	create := doJSON(t, h, http.MethodPost, "/api/posts/", map[string]string{"title": "Temp"})
	if create.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d", create.Code, http.StatusCreated)
	}
	var created struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(create.Body.Bytes(), &created); err != nil {
		t.Fatalf("decoding created post: %v", err)
	}

	del := doJSON(t, h, http.MethodDelete, "/api/posts/"+intToStr(created.ID)+"/", nil)
	if del.Code != http.StatusNoContent {
		t.Errorf("delete status = %d, want %d", del.Code, http.StatusNoContent)
	}

	all, _ := store.PostStore{DB: db}.ListAll()
	if len(all) != 0 {
		t.Errorf("after delete, ListAll = %d posts, want 0", len(all))
	}
}

func TestPublicGetPostVisibility(t *testing.T) {
	db := newHandlerTestDB(t)
	h := newTestRouter(db)

	doJSON(t, h, http.MethodPost, "/api/posts/", map[string]string{"title": "Hidden One"})

	if rec := doJSON(t, h, http.MethodGet, "/posts/hidden-one/", nil); rec.Code != http.StatusNotFound {
		t.Errorf("draft via public API: status = %d, want %d", rec.Code, http.StatusNotFound)
	}

	list := doJSON(t, h, http.MethodGet, "/api/posts/", nil)
	var posts []struct {
		ID     int64  `json:"id"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(list.Body.Bytes(), &posts); err != nil {
		t.Fatalf("decoding admin list: %v", err)
	}
	if len(posts) != 1 {
		t.Fatalf("admin list = %d posts, want 1", len(posts))
	}

	put := doJSON(t, h, http.MethodPut, "/api/posts/"+intToStr(posts[0].ID)+"/",
		map[string]string{"title": "Hidden One", "slug": "hidden-one", "status": "published"})
	if put.Code != http.StatusOK {
		t.Fatalf("update status = %d, want %d; body: %s", put.Code, http.StatusOK, put.Body.String())
	}

	rec := doJSON(t, h, http.MethodGet, "/posts/hidden-one/", nil)
	if rec.Code != http.StatusOK {
		t.Errorf("published via public API: status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Hello World":      "hello-world",
		"already-a-slug":   "already-a-slug",
		"Mixed CASE Title": "mixed-case-title",
	}
	for in, want := range cases {
		if got := web.Slugify(in); got != want {
			t.Errorf("Slugify(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestHealthCheck(t *testing.T) {
	h := newTestRouter(newHandlerTestDB(t))

	rec := doJSON(t, h, http.MethodGet, "/healthz/", nil)
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var got map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if got["status"] != "ok" {
		t.Errorf("status field = %q, want ok", got["status"])
	}
}

func TestPostsTableFragment(t *testing.T) {
	h := newTestRouter(newHandlerTestDB(t))

	doJSON(t, h, http.MethodPost, "/api/posts/", map[string]string{"title": "Fragment Me"})
	rec := doJSON(t, h, http.MethodGet, "/admin/posts/table/", nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if body := rec.Body.String(); !strings.Contains(body, "Fragment Me") {
		t.Errorf("fragment does not contain the post title; body:\n%s", body)
	}
}

func TestDashboardPageRendersStats(t *testing.T) {
	h := newTestRouter(newHandlerTestDB(t))

	doJSON(t, h, http.MethodPost, "/api/posts/", map[string]string{"title": "Shown Here"})
	rec := doJSON(t, h, http.MethodGet, "/admin/dashboard/", nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Shown Here") {
		t.Errorf("dashboard does not contain recent post title; body:\n%s", body)
	}
	if !strings.Contains(body, "Total Posts") {
		t.Errorf("dashboard does not contain stats; body:\n%s", body)
	}
}

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
	service.InitUsersTable(db)
	service.CreateDefaultAdmin(db)

	a := &web.Admin{DB: db, Tpl: testRenderer}
	r := chi.NewRouter()
	r.Post("/admin/login/", a.LoginPost)

	req := httptest.NewRequest(http.MethodPost, "/admin/login/",
		strings.NewReader("username=admin&password=admin123"))
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
	service.InitUsersTable(db)
	service.CreateDefaultAdmin(db)

	a := &web.Admin{DB: db, Tpl: testRenderer}
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
