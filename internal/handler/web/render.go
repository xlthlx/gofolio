package web

import (
	"html/template"
	"io"
	"os"
	"path/filepath"
)

// Renderer parses the templates of both themes once at startup and renders
// pages and fragments. HTMX needs server-rendered HTML; this is the single
// place that produces it.
type Renderer struct {
	posts      *template.Template
	pages      *template.Template
	settings   *template.Template
	dashboard  *template.Template
	login      *template.Template
	postEditor *template.Template
	pageEditor *template.Template
	site       *template.Template
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

// FindPublicThemeDir locates the public theme directory under the same rules as
// FindAdminThemeDir: THEMES_DIR first, then a walk up from the working
// directory, which is what lets tests find the repository from any package.
func FindPublicThemeDir() string {
	if v := os.Getenv("THEMES_DIR"); v != "" {
		return filepath.Join(v, "public", "default")
	}
	dir, _ := os.Getwd()
	for {
		candidate := filepath.Join(dir, PublicThemeDir, "index.html")
		if _, err := os.Stat(candidate); err == nil {
			return filepath.Join(dir, PublicThemeDir)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return PublicThemeDir // best effort; NewRenderer will fail with a clear error
}

// NewRenderer parses the admin templates from the admin theme directory and the
// public templates from the public one.
func NewRenderer(adminThemeDir, publicThemeDir string) (*Renderer, error) {
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
	postEditor, err := template.ParseFiles(
		filepath.Join(adminThemeDir, "editor.html"),
	)
	if err != nil {
		return nil, err
	}
	pageEditor, err := template.ParseFiles(
		filepath.Join(adminThemeDir, "page_editor.html"),
	)
	if err != nil {
		return nil, err
	}
	// The public pages are parsed as one group so the layout partials defined in
	// layout.html are available to all of them, the way the admin groups share
	// their fragment templates.
	site, err := template.ParseFiles(
		filepath.Join(publicThemeDir, "index.html"),
		filepath.Join(publicThemeDir, "post.html"),
		filepath.Join(publicThemeDir, "page.html"),
		filepath.Join(publicThemeDir, "not_found.html"),
		filepath.Join(publicThemeDir, "layout.html"),
	)
	if err != nil {
		return nil, err
	}
	return &Renderer{posts: posts, pages: pages, settings: settings,
		dashboard: dashboard, login: login, postEditor: postEditor,
		pageEditor: pageEditor, site: site}, nil
}

// MustRenderer is NewRenderer, panicking on failure: a server that cannot
// parse its templates should not start.
func MustRenderer(adminThemeDir, publicThemeDir string) *Renderer {
	r, err := NewRenderer(adminThemeDir, publicThemeDir)
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

// SiteSettings renders only the site settings section, for the form to swap in
// after a save.
func (r *Renderer) SiteSettings(w io.Writer, data interface{}) error {
	return r.settings.ExecuteTemplate(w, "site_settings", data)
}

// HomePage, PostPage, PagePage and NotFound render the public site.
func (r *Renderer) HomePage(w io.Writer, data interface{}) error {
	return r.site.ExecuteTemplate(w, "index_page", data)
}

func (r *Renderer) PostPage(w io.Writer, data interface{}) error {
	return r.site.ExecuteTemplate(w, "post_page", data)
}

func (r *Renderer) PagePage(w io.Writer, data interface{}) error {
	return r.site.ExecuteTemplate(w, "page_page", data)
}

func (r *Renderer) NotFound(w io.Writer, data interface{}) error {
	return r.site.ExecuteTemplate(w, "not_found_page", data)
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

func (r *Renderer) PostEditorPage(w io.Writer, data interface{}) error {
	return r.postEditor.ExecuteTemplate(w, "editor_page", data)
}

func (r *Renderer) PostEditorError(w io.Writer, data interface{}) error {
	return r.postEditor.ExecuteTemplate(w, "editor_error", data)
}

func (r *Renderer) PageEditorPage(w io.Writer, data interface{}) error {
	return r.pageEditor.ExecuteTemplate(w, "page_editor_page", data)
}

func (r *Renderer) PageEditorError(w io.Writer, data interface{}) error {
	return r.pageEditor.ExecuteTemplate(w, "page_editor_error", data)
}
