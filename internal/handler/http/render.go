package web

import (
	"html/template"
	"io"
	"os"
	"path/filepath"
)

type Renderer struct {
	posts *template.Template
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

func NewRenderer(adminThemeDir string) (*Renderer, error) {
	posts, err := template.ParseFiles(
		filepath.Join(adminThemeDir, "posts.html"),
		filepath.Join(adminThemeDir, "posts_table.html"),
	)
	if err != nil {
		return nil, err
	}
	return &Renderer{posts: posts}, nil
}

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
