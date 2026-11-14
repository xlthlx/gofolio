package web

import (
	"database/sql"
	"errors"
	"net/http"

	"gofolio/internal/model"
)

// PageEditorData feeds the page editor page and its error fragment.
type PageEditorData struct {
	Page  model.Page
	Error string
}

// PageEditor serves the page editor for both /admin/pages/new/ and
// /admin/pages/{id}/edit/, mirroring PostEditor: the values are rendered into
// the form, and the status the server stores is the status the select sends
// back.
func (h Admin) PageEditor(w http.ResponseWriter, r *http.Request) {
	data := PageEditorData{}

	id, err := routeID(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if id > 0 {
		page, err := h.pages().GetByID(id)
		if errors.Is(err, sql.ErrNoRows) {
			http.NotFound(w, r)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		data.Page = *page
	}

	h.renderPageEditor(w, data, http.StatusOK)
}

// SavePage handles the editor's form post for both creating and updating a
// page, following the same HTMX and plain form conventions as SavePost.
func (h Admin) SavePage(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	page := model.Page{
		Title:   r.PostFormValue("title"),
		Slug:    r.PostFormValue("slug"),
		Content: r.PostFormValue("content"),
		Status:  r.PostFormValue("status"),
	}
	if page.Slug == "" && page.Title != "" {
		page.Slug = Slugify(page.Title)
	}

	id, err := formID(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	if page.Title == "" {
		h.pageEditorFailure(w, r, page, "Title is required.", http.StatusBadRequest)
		return
	}

	if id > 0 {
		err = h.pages().Update(&page, id)
	} else {
		err = h.pages().Create(&page)
	}
	if err != nil {
		h.pageEditorFailure(w, r, page, "Could not save the page: "+err.Error(),
			http.StatusInternalServerError)
		return
	}

	redirectAfterSave(w, r, "/admin/pages/")
}

// pageEditorFailure answers with the error fragment for HTMX and with the
// editor page itself otherwise, keeping what the user typed in both cases.
func (h Admin) pageEditorFailure(w http.ResponseWriter, r *http.Request, page model.Page, msg string, code int) {
	data := PageEditorData{Page: page, Error: msg}
	if isHTMX(r) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(code)
		if err := h.Tpl.PageEditorError(w, data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	h.renderPageEditor(w, data, code)
}

func (h Admin) renderPageEditor(w http.ResponseWriter, data PageEditorData, code int) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(code)
	if err := h.Tpl.PageEditorPage(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
