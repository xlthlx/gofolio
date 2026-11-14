package web

import (
	"html/template"
	"io"
	"os"
	"path/filepath"
)

// Renderer parses the admin templates once at startup and renders pages and
// fragments. HTMX needs server-rendered HTML; this is the single place that
// produces it.
type Renderer struct {
	posts     *template.Template
	pages     *template.Template
	settings  *template.Template
	dashboard *template.Template
	login     *template.Template
}

// FindAdminThemeDir locates the admin theme directory. It honours the
// THEMES_DIR environment variable, then walks up from the current working
// directory so tests running from package directories also find the repo.
func FindAdminThemeDir() string {
	if v := os.Getenv("THEMES_DIR"); v != "" {
		return filepath.Join(v, "admin", "default")
	}
	dir, _ := os.Getwd()
	for {
		candidate := filepath.Join(dir, AdminThemeDir, "posts.html")
		if _, err := os.Stat(candidate); err == nil {
			return filepath.Join(dir, AdminThemeDir)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return AdminThemeDir // best effort; NewRenderer will fail with a clear error
}

// NewRenderer parses the admin templates from the theme directory.
func NewRenderer(adminThemeDir string) (*Renderer, error) {
	posts, err := template.ParseFiles(
		filepath.Join(adminThemeDir, "posts.html"),
		filepath.Join(adminThemeDir, "posts_table.html"),
	)
	if err != nil {
		return nil, err
	}
	pages, err := template.ParseFiles(
		filepath.Join(adminThemeDir, "pages.html"),
		filepath.Join(adminThemeDir, "pages_table.html"),
	)
	if err != nil {
		return nil, err
	}
	settings, err := template.ParseFiles(
		filepath.Join(adminThemeDir, "settings.html"),
		filepath.Join(adminThemeDir, "site_settings.html"),
	)
	if err != nil {
		return nil, err
	}
	dashboard, err := template.ParseFiles(
		filepath.Join(adminThemeDir, "dashboard.html"),
		filepath.Join(adminThemeDir, "dashboard_recent.html"),
	)
	if err != nil {
		return nil, err
	}
	login, err := template.ParseFiles(
		filepath.Join(adminThemeDir, "login.html"),
	)
	if err != nil {
		return nil, err
	}
	return &Renderer{posts: posts, pages: pages, settings: settings,
		dashboard: dashboard, login: login}, nil
}

// MustRenderer is NewRenderer, panicking on failure: a server that cannot
// parse its templates should not start.
func MustRenderer(dir string) *Renderer {
	r, err := NewRenderer(dir)
	if err != nil {
		panic(err)
	}
	return r
}

func (r *Renderer) PostsPage(w io.Writer, data interface{}) error {
	return r.posts.ExecuteTemplate(w, "posts.html", data)
}

func (r *Renderer) PostsRows(w io.Writer, data interface{}) error {
	return r.posts.ExecuteTemplate(w, "posts_rows", data)
}

func (r *Renderer) PagesPage(w io.Writer, data interface{}) error {
	return r.pages.ExecuteTemplate(w, "pages.html", data)
}

func (r *Renderer) PagesRows(w io.Writer, data interface{}) error {
	return r.pages.ExecuteTemplate(w, "pages_rows", data)
}

func (r *Renderer) SettingsPage(w io.Writer, data interface{}) error {
	return r.settings.ExecuteTemplate(w, "settings.html", data)
}

func (r *Renderer) DashboardPage(w io.Writer, data interface{}) error {
	return r.dashboard.ExecuteTemplate(w, "dashboard.html", data)
}

func (r *Renderer) RecentRows(w io.Writer, data interface{}) error {
	return r.dashboard.ExecuteTemplate(w, "recent_rows", data)
}

func (r *Renderer) LoginPage(w io.Writer, data interface{}) error {
	return r.login.ExecuteTemplate(w, "login_page", data)
}

func (r *Renderer) LoginError(w io.Writer, data interface{}) error {
	return r.login.ExecuteTemplate(w, "login_error", data)
}
