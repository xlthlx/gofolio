package web

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"gofolio/internal/model"
	"gofolio/internal/store"
)

// Admin handles the authenticated admin panel: pages, fragments and CRUD.
type Admin struct {
	DB  *sql.DB
	Tpl *Renderer
}

// View serves a static admin template (used by pages not yet migrated).
func (h Admin) View(file string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, AdminThemeDir+"/"+file)
	}
}

// PostsData is the model for the posts page and the table fragment.
type PostsData struct {
	Posts []model.Post
}

func (h Admin) postsData() PostsData {
	posts, err := h.posts().ListAll()
	if err != nil {
		return PostsData{}
	}
	return PostsData{Posts: posts}
}

// PostsPage renders the full posts management page, rows included.
func (h Admin) PostsPage(w http.ResponseWriter, r *http.Request) {
	data := h.postsData()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.Tpl.PostsPage(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// PostsTable renders only the posts table, for HTMX fragment swaps.
func (h Admin) PostsTable(w http.ResponseWriter, r *http.Request) {
	data := h.postsData()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.Tpl.PostsRows(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h Admin) ListPosts(w http.ResponseWriter, r *http.Request) {
	posts, err := h.posts().ListAll()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, posts)
}

func (h Admin) CreatePost(w http.ResponseWriter, r *http.Request) {
	var post model.Post
	if err := decodeJSON(r, &post); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if post.Title == "" {
		http.Error(w, "Title is required", http.StatusBadRequest)
		return
	}
	if post.Slug == "" {
		post.Slug = Slugify(post.Title)
	}
	if post.Status == "" {
		post.Status = "draft"
	}
	if err := h.posts().Create(&post); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, post)
}

func (h Admin) UpdatePost(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var post model.Post
	if err := decodeJSON(r, &post); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.posts().Update(&post, id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated", "id": strconv.FormatInt(id, 10)})
}

// DeletePost removes a post. HTMX requests get an empty 200 (the caller
// swaps away the table row); plain API clients keep the original 204.
func (h Admin) DeletePost(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := h.posts().Delete(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if r.Header.Get("HX-Request") == "true" {
		w.WriteHeader(http.StatusOK)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h Admin) ListPages(w http.ResponseWriter, r *http.Request) {
	pages, err := h.pages().ListAll()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, pages)
}

func (h Admin) CreatePage(w http.ResponseWriter, r *http.Request) {
	var page model.Page
	if err := decodeJSON(r, &page); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if page.Title == "" {
		http.Error(w, "Title is required", http.StatusBadRequest)
		return
	}
	if page.Slug == "" {
		page.Slug = Slugify(page.Title)
	}
	if err := h.pages().Create(&page); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "created"})
}

// posts and pages hoist the store construction out of statement position;
// composite literals inside if-headers are a Go parsing ambiguity.
func (h Admin) posts() store.PostStore {
	return store.PostStore{DB: h.DB}
}

func (h Admin) pages() store.PageStore {
	return store.PageStore{DB: h.DB}
}

func parseID(r *http.Request) (int64, error) {
	return strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
}

// Slugify converts a title to a URL slug (lowercase, hyphen-separated).
func Slugify(title string) string {
	return strings.ToLower(strings.ReplaceAll(title, " ", "-"))
}
