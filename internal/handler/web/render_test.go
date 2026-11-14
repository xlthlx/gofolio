package web_test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gofolio/internal/handler/web"
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
	if r := web.MustRenderer(web.FindAdminThemeDir()); r == nil {
		t.Fatal("MustRenderer returned nil despite valid templates")
	}
}

// TestAdminPagesLoadHTMXConfig checks that every page carrying a head loads the
// config that makes HTMX swap error fragments, and loads it after HTMX itself.
// The fragment templates have no head of their own, so they are covered by the
// pages they are swapped into.
func TestAdminPagesLoadHTMXConfig(t *testing.T) {
	r := web.MustRenderer(web.FindAdminThemeDir())

	if _, err := os.Stat(filepath.Join(web.FindAdminThemeDir(), "assets", "js", "htmx-config.js")); err != nil {
		t.Errorf("HTMX config asset is missing: %v", err)
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
	}

	for name, page := range pages {
		var buf bytes.Buffer
		if err := page.render(&buf, page.data); err != nil {
			t.Errorf("rendering %s: %v", name, err)
			continue
		}
		body := buf.String()
		if !strings.Contains(body, "/admin/static/assets/js/htmx.min.js") {
			t.Errorf("%s does not load HTMX", name)
		}
		if !strings.Contains(body, "/admin/static/assets/js/htmx-config.js") {
			t.Errorf("%s does not load the HTMX config, so error fragments stay invisible", name)
		}
	}
}
