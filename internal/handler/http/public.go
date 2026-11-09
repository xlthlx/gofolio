package web

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"gofolio/internal/store"
)

// Public handles the public, unauthenticated site.
type Public struct{ DB *sql.DB }

func (h Public) ListPosts(w http.ResponseWriter, r *http.Request) {
	posts, err := store.PostStore{DB: h.DB}.ListPublished()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"posts": posts})
}

func (h Public) GetPost(w http.ResponseWriter, r *http.Request) {
	post, err := store.PostStore{DB: h.DB}.GetPublishedBySlug(chi.URLParam(r, "slug"))
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, post)
}

func (h Public) GetPage(w http.ResponseWriter, r *http.Request) {
	page, err := store.PageStore{DB: h.DB}.GetBySlug(chi.URLParam(r, "slug"))
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, page)
}

func ServeStatic(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/static/")
	http.ServeFile(w, r, PublicThemeDir+"/"+path)
}
