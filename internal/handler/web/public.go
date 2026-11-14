package web

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"gofolio/internal/handler/middleware"
	"gofolio/internal/store"
)

// Public handles the public, unauthenticated site: the HTML views in site.go
// and the JSON API the same content is also published through.
type Public struct {
	DB   *sql.DB
	Tpl  *Renderer
	Sess *middleware.Sessions
}

// posts and pages hoist the store construction, matching the admin handlers.
func (h Public) posts() store.PostStore {
	return store.PostStore{DB: h.DB}
}

func (h Public) pages() store.PageStore {
	return store.PageStore{DB: h.DB}
}

func (h Public) ListPosts(w http.ResponseWriter, r *http.Request) {
	posts, err := h.posts().ListPublished()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"posts": posts})
}

func (h Public) GetPost(w http.ResponseWriter, r *http.Request) {
	post, err := h.posts().GetPublishedBySlug(chi.URLParam(r, "slug"))
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
	page, err := h.pages().GetPublishedBySlug(chi.URLParam(r, "slug"))
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
