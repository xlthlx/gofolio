package web

import (
	"database/sql"
	"errors"
	"net/http"

	"gofolio/internal/model"
)

// EditorData feeds the post editor page and its error fragment.
type EditorData struct {
	Post  model.Post
	Error string
}

// PostEditor serves the post editor for both /admin/posts/new/ and
// /admin/posts/{id}/edit/. The values arrive already rendered into the form, so
// the editor needs no JSON round trip to load: TinyMCE initialises over a
// textarea that the server has filled in.
func (h Admin) PostEditor(w http.ResponseWriter, r *http.Request) {
	data := EditorData{}

	id, err := routeID(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if id > 0 {
		post, err := h.posts().GetByID(id)
		if errors.Is(err, sql.ErrNoRows) {
			http.NotFound(w, r)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		data.Post = *post
	}

	h.renderPostEditor(w, data, http.StatusOK)
}

// SavePost handles the editor's form post, creating when the form carries no
// id and updating when it does. HTMX requests get an error fragment or an
// HX-Redirect; plain form posts get a redirect or the re-rendered editor, so
// saving works with JavaScript disabled.
func (h Admin) SavePost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	post := model.Post{
		Title:   r.PostFormValue("title"),
		Slug:    r.PostFormValue("slug"),
		Content: r.PostFormValue("content"),
		Status:  r.PostFormValue("status"),
	}
	if post.Slug == "" && post.Title != "" {
		post.Slug = Slugify(post.Title)
	}

	id, err := formID(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	if post.Title == "" {
		h.postEditorFailure(w, r, post, "Title is required.", http.StatusBadRequest)
		return
	}

	if id > 0 {
		err = h.posts().Update(&post, id)
	} else {
		err = h.posts().Create(&post)
	}
	if err != nil {
		h.postEditorFailure(w, r, post, "Could not save the post: "+err.Error(),
			http.StatusInternalServerError)
		return
	}

	redirectAfterSave(w, r, "/admin/posts/")
}

// postEditorFailure answers with the error fragment for HTMX and with the
// editor page itself otherwise, keeping what the user typed in both cases.
func (h Admin) postEditorFailure(w http.ResponseWriter, r *http.Request, post model.Post, msg string, code int) {
	data := EditorData{Post: post, Error: msg}
	if isHTMX(r) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(code)
		if err := h.Tpl.PostEditorError(w, data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	h.renderPostEditor(w, data, code)
}

func (h Admin) renderPostEditor(w http.ResponseWriter, data EditorData, code int) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(code)
	if err := h.Tpl.PostEditorPage(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
