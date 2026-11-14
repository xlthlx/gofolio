package web_test

import (
	"os"
	"path/filepath"
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
