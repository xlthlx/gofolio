package web_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"gofolio/internal/store"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	web "gofolio/internal/handler/http"
	"gofolio/internal/handler/middleware"
	"gofolio/internal/service"
	"gofolio/pkg/util"
)

// testRenderer is shared: the admin templates are parsed once, from the repo
// root (the finder walks up from this package's directory).
var testRenderer = web.MustRenderer(web.FindAdminThemeDir())

var testSess = middleware.NewSessions([]byte("unit-test-secret"))

// newHandlerTestDB creates a throwaway database with ALL tables the handler
// layer can touch — content and users — so any test can seed anything.
func newHandlerTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := store.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("opening test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	if err := store.Migrate(db); err != nil {
		t.Fatalf("migrating: %v", err)
	}
	if err := service.InitUsersTable(db); err != nil {
		t.Fatalf("creating users table: %v", err)
	}
	return db
}

// newTestRouter is the minimal-but-complete route surface the unit tests
// exercise, mirroring the production router.
func newTestRouter(db *sql.DB) http.Handler {
	a := &web.Admin{DB: db, Tpl: testRenderer, Sess: testSess}
	p := &web.Public{DB: db}

	r := chi.NewRouter()

	// Public
	r.Get("/healthz/", (&web.Health{DB: db}).Check)
	r.Get("/posts/{slug}/", p.GetPost)

	// Posts API
	r.Get("/api/posts/", a.ListPosts)
	r.Post("/api/posts/", a.CreatePost)
	r.Put("/api/posts/{id}/", a.UpdatePost)
	r.Delete("/api/posts/{id}/", a.DeletePost)

	// Admin pages and fragments
	r.Get("/admin/dashboard/", a.DashboardPage)
	r.Get("/admin/dashboard/recent/", a.RecentPosts)
	r.Get("/admin/posts/", a.PostsPage)
	r.Get("/admin/posts/table/", a.PostsTable)
	r.Get("/admin/pages/", a.PagesPage)
	r.Get("/admin/pages/table/", a.PagesTable)
	r.Get("/admin/settings/", a.SettingsPage)

	// Pages API
	r.Post("/admin/api/pages/", a.CreatePage)

	// Login (outside any auth middleware, as in production)
	r.Get("/admin/login/", a.LoginGet)
	r.Post("/admin/login/", a.LoginPost)

	return r
}

func doJSON(t *testing.T, h http.Handler, method, path string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshalling request body: %v", err)
		}
		rd = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, rd)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// --------------------------------------------------------------------------
// Store-backed handler tests (steps 1 and earlier)
// --------------------------------------------------------------------------

func TestCreatePostRequiresTitle(t *testing.T) {
	h := newTestRouter(newHandlerTestDB(t))

	rec := doJSON(t, h, http.MethodPost, "/api/posts/", map[string]string{"title": ""})
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
	var got struct {
		Slug   string `json:"slug"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decoding: %v", err)
	}
	if got.Slug != "hello-world" {
		t.Errorf("slug = %q, want hello-world", got.Slug)
	}
	if got.Status != "draft" {
		t.Errorf("status = %q, want draft", got.Status)
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
	h := newTestRouter(newHandlerTestDB(t))

	create := doJSON(t, h, http.MethodPost, "/api/posts/", map[string]string{"title": "Doomed"})
	var created struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(create.Body.Bytes(), &created); err != nil {
		t.Fatalf("decoding created post: %v", err)
	}

	del := doJSON(t, h, http.MethodDelete, fmt.Sprintf("/api/posts/%d/", created.ID), nil)
	if del.Code != http.StatusNoContent {
		t.Errorf("delete status = %d, want %d", del.Code, http.StatusNoContent)
	}

	get := doJSON(t, h, http.MethodGet, "/posts/doomed/", nil)
	if get.Code != http.StatusNotFound {
		t.Errorf("deleted post still visible: status = %d, want %d", get.Code, http.StatusNotFound)
	}
}

func TestPublicGetPostVisibility(t *testing.T) {
	h := newTestRouter(newHandlerTestDB(t))

	doJSON(t, h, http.MethodPost, "/api/posts/",
		map[string]string{"title": "Hidden Draft", "status": "draft"})
	rec := doJSON(t, h, http.MethodGet, "/posts/hidden-draft/", nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("draft visible publicly: status = %d, want %d", rec.Code, http.StatusNotFound)
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

// --------------------------------------------------------------------------
// HTMX fragments (steps 1-3)
// --------------------------------------------------------------------------

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

func TestPagesTableFragment(t *testing.T) {
	h := newTestRouter(newHandlerTestDB(t))

	doJSON(t, h, http.MethodPost, "/admin/api/pages/", map[string]string{"title": "About Fragment"})
	rec := doJSON(t, h, http.MethodGet, "/admin/pages/table/", nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if body := rec.Body.String(); !strings.Contains(body, "About Fragment") {
		t.Errorf("fragment does not contain page title; body:\n%s", body)
	}
}

// --------------------------------------------------------------------------
// Dashboard (step 2)
// --------------------------------------------------------------------------

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

// --------------------------------------------------------------------------
// Login (step 2)
// --------------------------------------------------------------------------

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
	if _, err := service.CreateUser(db, "admin", "admin@localhost", "admin123", true); err != nil {
		t.Fatalf("seeding admin: %v", err)
	}

	a := &web.Admin{DB: db, Tpl: testRenderer, Sess: testSess}
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
	if _, err := service.CreateUser(db, "admin", "admin@localhost", "admin123", true); err != nil {
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

// --------------------------------------------------------------------------
// Password change (step 3)
// --------------------------------------------------------------------------

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

func newTestRouterWithPages(db *sql.DB) http.Handler {
	a := &web.Admin{DB: db, Tpl: testRenderer, Sess: testSess}
	r := chi.NewRouter()
	r.Post("/admin/api/pages/", a.CreatePage)
	r.Get("/admin/api/pages/", a.ListPages) // <-- the missing line
	r.Get("/api/pages/{id}/", a.GetPageAPI)
	r.Put("/api/pages/{id}/", a.UpdatePage)
	r.Delete("/api/pages/{id}/", a.DeletePage)
	r.Get("/admin/pages/table/", a.PagesTable)
	return r
}

func TestPageEditorRoundTrip(t *testing.T) {
	h := newTestRouterWithPages(newHandlerTestDB(t))

	create := doJSON(t, h, http.MethodPost, "/admin/api/pages/",
		map[string]string{"title": "About Us", "content": "<p>v1</p>"})
	if create.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body: %s", create.Code, create.Body.String())
	}
	var created struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(create.Body.Bytes(), &created); err != nil {
		t.Fatalf("decoding create: %v", err)
	}

	// The list API is how the editor discovers the id (Create returns only
	// status); fetch it, then exercise get-one, update and get again.
	list := doJSON(t, h, http.MethodGet, "/admin/api/pages/", nil)
	var pages []struct {
		ID      int64  `json:"id"`
		Title   string `json:"title"`
		Slug    string `json:"slug"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(list.Body.Bytes(), &pages); err != nil || len(pages) != 1 {
		t.Fatalf("listing pages: %v (%d pages)", err, len(pages))
	}
	id := pages[0].ID
	if pages[0].Slug != "about-us" {
		t.Errorf("derived slug = %q, want about-us", pages[0].Slug)
	}
	rec := doJSON(t, h, http.MethodGet, "/admin/pages/table/", nil)
	if !strings.Contains(rec.Body.String(), "About Us") {
		t.Error("table fragment does not render created page")
	}

	got := doJSON(t, h, http.MethodGet, fmt.Sprintf("/api/pages/%d/", id), nil)
	if got.Code != http.StatusOK {
		t.Fatalf("get status = %d, want %d", got.Code, http.StatusOK)
	}
	var page struct {
		Content string `json:"content"`
	}
	if err := json.Unmarshal(got.Body.Bytes(), &page); err != nil {
		t.Fatalf("decoding page: %v", err)
	}
	if page.Content != "<p>v1</p>" {
		t.Errorf("content = %q, want <p>v1</p>", page.Content)
	}

	upd := doJSON(t, h, http.MethodPut, fmt.Sprintf("/api/pages/%d/", id),
		map[string]string{"title": "About Us", "content": "<p>v2</p>"})
	if upd.Code != http.StatusOK {
		t.Fatalf("update status = %d, body: %s", upd.Code, upd.Body.String())
	}
	got2 := doJSON(t, h, http.MethodGet, fmt.Sprintf("/api/pages/%d/", id), nil)
	var page2 struct {
		Content string `json:"content"`
	}
	json.Unmarshal(got2.Body.Bytes(), &page2)
	if page2.Content != "<p>v2</p>" {
		t.Errorf("content after update = %q, want <p>v2</p>", page2.Content)
	}

	del := doJSON(t, h, http.MethodDelete, fmt.Sprintf("/api/pages/%d/", id), nil)
	if del.Code != http.StatusNoContent {
		t.Errorf("delete status = %d, want %d", del.Code, http.StatusNoContent)
	}
	final := doJSON(t, h, http.MethodGet, fmt.Sprintf("/api/pages/%d/", id), nil)
	if final.Code != http.StatusNotFound {
		t.Errorf("deleted page still fetchable: status = %d, want %d", final.Code, http.StatusNotFound)
	}
}

func TestUpdatePageInvalidID(t *testing.T) {
	h := newTestRouterWithPages(newHandlerTestDB(t))
	rec := doJSON(t, h, http.MethodPut, "/api/pages/notanumber/",
		map[string]string{"title": "x"})
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

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
