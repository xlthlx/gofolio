package web_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"gofolio/internal/handler/middleware"
	"gofolio/internal/handler/web"
	"gofolio/internal/service"
	"gofolio/internal/store"
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
	authH := &web.Auth{DB: db, Sess: testSess}

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

	// Auth API (outside any auth middleware, as in production)
	r.Post("/api/auth/login/", authH.Login)
	r.Get("/api/auth/logout/", authH.Logout)
	r.Get("/api/auth/check/", authH.Check)

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
