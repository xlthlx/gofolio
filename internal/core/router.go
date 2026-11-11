package core

import (
	"database/sql"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	// The package is named "web"; the folder is handler/http per the docs.
	web "gofolio/internal/handler/http"
	"gofolio/internal/handler/middleware"
)

// New builds the full application router.
func New(db *sql.DB, tpl *web.Renderer) http.Handler {
	r := chi.NewRouter()

	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.RealIP)

	// Trailing slash enforcement
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if req.Method == http.MethodGet && !strings.HasSuffix(req.URL.Path, "/") {
				redirect := req.URL.Path + "/"
				if req.URL.RawQuery != "" {
					redirect += "?" + req.URL.RawQuery
				}
				http.Redirect(w, req, redirect, http.StatusMovedPermanently)
				return
			}
			next.ServeHTTP(w, req)
		})
	})

	pub := &web.Public{DB: db}
	admin := &web.Admin{DB: db, Tpl: tpl}
	authH := &web.Auth{DB: db}

	// Public routes
	r.Get("/", pub.ListPosts)
	r.Get("/healthz/", (&web.Health{DB: db}).Check)
	r.Get("/posts/{slug}/", pub.GetPost)
	r.Get("/pages/{slug}/", pub.GetPage)
	r.Get("/static/*", web.ServeStatic)

	// Admin routes with authentication
	r.Route("/admin/", func(r chi.Router) {
		r.Use(middleware.RequireAuth)

		r.Get("/", admin.DashboardPage)
		r.Get("/dashboard/", admin.DashboardPage)
		r.Get("/dashboard/recent/", admin.RecentPosts)
		r.Get("/posts/", admin.PostsPage)
		r.Get("/posts/table/", admin.PostsTable)
		r.Get("/posts/new/", admin.View("editor.html"))
		r.Get("/posts/{id}/edit/", admin.View("editor.html"))
		r.Get("/pages/", admin.PagesPage)
		r.Get("/pages/table/", admin.PagesTable)
		r.Get("/settings/", admin.SettingsPage)
		r.Post("/settings/password/", admin.ChangePassword)

		r.Get("/api/posts/", admin.ListPosts)
		r.Post("/api/posts/", admin.CreatePost)
		r.Put("/api/posts/{id}/", admin.UpdatePost)
		r.Delete("/api/posts/{id}/", admin.DeletePost)
		r.Get("/api/pages/", admin.ListPages)
		r.Post("/api/pages/", admin.CreatePage)
	})

	r.Get("/admin/login/", admin.LoginGet)
	r.Post("/admin/login/", admin.LoginPost)
	r.Get("/admin/static/*", web.ServeAdminStatic)

	// Auth routes (no auth required)
	r.Post("/api/auth/login/", authH.Login)
	r.Get("/api/auth/logout/", authH.Logout)
	r.Get("/api/auth/check/", authH.Check)

	return r
}
