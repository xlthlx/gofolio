package core

import (
	"database/sql"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"gofolio/internal/handler/middleware"
	"gofolio/internal/handler/web"
)

// New builds the full application router.
func New(db *sql.DB, tpl *web.Renderer, sess *middleware.Sessions) http.Handler {
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

	pub := &web.Public{DB: db, Tpl: tpl, Sess: sess}
	admin := &web.Admin{DB: db, Tpl: tpl, Sess: sess}
	authH := &web.Auth{DB: db, Sess: sess}

	// Public site, rendered as HTML
	r.Get("/", pub.HomePage)
	r.Get("/healthz/", (&web.Health{DB: db}).Check)
	r.Get("/posts/{slug}/", pub.PostPage)
	r.Get("/pages/{slug}/", pub.PagePage)
	r.Get("/static/*", web.ServeStatic)

	// Public JSON API, the same content in machine-readable form
	r.Get("/api/posts/", pub.ListPosts)
	r.Get("/api/posts/{slug}/", pub.GetPost)
	r.Get("/api/pages/{slug}/", pub.GetPage)

	// Admin routes with authentication
	r.Route("/admin/", func(r chi.Router) {
		r.Use(middleware.RequireAuth(sess))

		r.Get("/", admin.DashboardPage)
		r.Get("/dashboard/", admin.DashboardPage)
		r.Get("/dashboard/recent/", admin.RecentPosts)

		r.Get("/posts/", admin.PostsPage)
		r.Get("/posts/table/", admin.PostsTable)
		r.Get("/posts/new/", admin.PostEditor)
		r.Get("/posts/{id}/edit/", admin.PostEditor)
		r.Post("/posts/save/", admin.SavePost)

		r.Get("/pages/", admin.PagesPage)
		r.Get("/pages/table/", admin.PagesTable)
		r.Get("/pages/new/", admin.PageEditor)
		r.Get("/pages/{id}/edit/", admin.PageEditor)
		r.Post("/pages/save/", admin.SavePage)

		r.Get("/settings/", admin.SettingsPage)
		r.Post("/settings/password/", admin.ChangePassword)
		r.Post("/settings/site/", admin.SiteSettingsPost)

		r.Get("/api/posts/", admin.ListPosts)
		r.Get("/api/posts/{id}/", admin.GetPost)
		r.Post("/api/posts/", admin.CreatePost)
		r.Put("/api/posts/{id}/", admin.UpdatePost)
		r.Delete("/api/posts/{id}/", admin.DeletePost)

		r.Get("/api/pages/", admin.ListPages)
		r.Get("/api/pages/{id}/", admin.GetPage)
		r.Post("/api/pages/", admin.CreatePage)
		r.Put("/api/pages/{id}/", admin.UpdatePage)
		r.Delete("/api/pages/{id}/", admin.DeletePage)
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
