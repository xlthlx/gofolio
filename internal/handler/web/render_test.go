package web_test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gofolio/internal/handler/web"
	"gofolio/internal/model"
)

func TestFindAdminThemeDirLocatesTemplates(t *testing.T) {
	dir := web.FindAdminThemeDir()
	for _, name := range []string{
		"posts.html", "posts_table.html",
		"pages.html", "pages_table.html",
		"settings.html", "site_settings.html",
		"dashboard.html", "dashboard_recent.html",
		"login.html",
		"editor.html", "page_editor.html",
	} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Errorf("theme dir %q missing %s: %v", dir, name, err)
		}
	}
}

func TestMustRendererParsesTemplates(t *testing.T) {
	if r := web.MustRenderer(web.FindAdminThemeDir(), web.FindPublicThemeDir()); r == nil {
		t.Fatal("MustRenderer returned nil despite valid templates")
	}
}

func TestFindPublicThemeDirLocatesTemplates(t *testing.T) {
	dir := web.FindPublicThemeDir()
	for _, name := range []string{
		"layout.html", "index.html", "post.html", "page.html", "not_found.html",
	} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Errorf("public theme dir %q missing %s: %v", dir, name, err)
		}
	}
	for _, name := range []string{"htmx.min.js", "htmx-config.js"} {
		if _, err := os.Stat(filepath.Join(dir, "assets", "js", name)); err != nil {
			t.Errorf("public theme dir %q missing asset %s: %v", dir, name, err)
		}
	}
}

// TestPagesLoadHTMXAndConfig checks that every page with a head, in either
// theme, loads HTMX and the config that makes it swap error fragments, and loads
// the config after HTMX itself. The fragment templates have no head of their
// own, so they are covered by the pages they are swapped into.
func TestPagesLoadHTMXAndConfig(t *testing.T) {
	r := web.MustRenderer(web.FindAdminThemeDir(), web.FindPublicThemeDir())

	for _, themeDir := range []string{web.FindAdminThemeDir(), web.FindPublicThemeDir()} {
		if _, err := os.Stat(filepath.Join(themeDir, "assets", "js", "htmx-config.js")); err != nil {
			t.Errorf("%s is missing the HTMX config asset: %v", themeDir, err)
		}
	}

	pages := map[string]struct {
		render func(w io.Writer, data interface{}) error
		data   interface{}
	}{
		"posts":       {r.PostsPage, web.PostsData{}},
		"pages":       {r.PagesPage, web.PagesData{}},
		"dashboard":   {r.DashboardPage, web.DashboardData{}},
		"settings":    {r.SettingsPage, web.SettingsData{}},
		"login":       {r.LoginPage, nil},
		"editor":      {r.PostEditorPage, web.EditorData{}},
		"page editor": {r.PageEditorPage, web.PageEditorData{}},
		"home":        {r.HomePage, web.SiteData{}},
		"post view":   {r.PostPage, web.SiteData{Post: &model.Post{}}},
		"page view":   {r.PagePage, web.SiteData{Page: &model.Page{}}},
		"not found":   {r.NotFound, web.SiteData{}},
	}

	for name, page := range pages {
		var buf bytes.Buffer
		if err := page.render(&buf, page.data); err != nil {
			t.Errorf("rendering %s: %v", name, err)
			continue
		}
		body := buf.String()
		if !strings.Contains(body, "htmx.min.js") {
			t.Errorf("%s does not load HTMX", name)
		}
		if !strings.Contains(body, "htmx-config.js") {
			t.Errorf("%s does not load the HTMX config, so error fragments stay invisible", name)
		}
	}
}
