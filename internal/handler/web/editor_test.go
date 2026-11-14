package web_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// createPost seeds a post through the JSON API and returns its id.
func createPost(t *testing.T, h http.Handler, fields map[string]string) int64 {
	t.Helper()
	rec := doJSON(t, h, http.MethodPost, "/api/posts/", fields)
	if rec.Code != http.StatusCreated {
		t.Fatalf("seeding post: status = %d, body: %s", rec.Code, rec.Body.String())
	}
	var created struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decoding seeded post: %v", err)
	}
	return created.ID
}

// fetchPost reads a post back through the admin API.
func fetchPost(t *testing.T, h http.Handler, id int64) map[string]interface{} {
	t.Helper()
	rec := doJSON(t, h, http.MethodGet, fmt.Sprintf("/admin/api/posts/%d/", id), nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("reading post %d: status = %d", id, rec.Code)
	}
	var post map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &post); err != nil {
		t.Fatalf("decoding post %d: %v", id, err)
	}
	return post
}

func TestPostEditorRendersEmptyForm(t *testing.T) {
	h := newTestRouter(newHandlerTestDB(t))

	rec := doJSON(t, h, http.MethodGet, "/admin/posts/new/", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	for _, want := range []string{
		`action="/admin/posts/save/"`,
		`hx-post="/admin/posts/save/"`,
		`name="id"`,
		`name="title"`,
		`name="slug"`,
		`name="content"`,
		`name="status"`,
		`tinymce.min.js`,
	} {
		if !strings.Contains(rec.Body.String(), want) {
			t.Errorf("new post editor is missing %s", want)
		}
	}
}

func TestPostEditorRendersStoredValues(t *testing.T) {
	h := newTestRouter(newHandlerTestDB(t))
	id := createPost(t, h, map[string]string{
		"title":   "Existing Post",
		"content": "<p>body</p>",
		"status":  "published",
	})

	rec := doJSON(t, h, http.MethodGet, fmt.Sprintf("/admin/posts/%d/edit/", id), nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	for _, want := range []string{
		`value="Existing Post"`,
		`&lt;p&gt;body&lt;/p&gt;`,
		`value="published" selected`,
	} {
		if !strings.Contains(rec.Body.String(), want) {
			t.Errorf("editor does not render %s", want)
		}
	}
}

func TestPostEditorUnknownIDIsNotFound(t *testing.T) {
	h := newTestRouter(newHandlerTestDB(t))

	rec := doJSON(t, h, http.MethodGet, "/admin/posts/4242/edit/", nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestPostEditorInvalidIDIsBadRequest(t *testing.T) {
	h := newTestRouter(newHandlerTestDB(t))

	rec := doJSON(t, h, http.MethodGet, "/admin/posts/notanumber/edit/", nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestSavePostCreatesAndRedirectsHTMX(t *testing.T) {
	h := newTestRouter(newHandlerTestDB(t))

	// The rendered form always carries an id field, which is 0 for a new post.
	rec := doForm(t, h, "/admin/posts/save/", url.Values{
		"id":     {"0"},
		"title":  {"Launch Day"},
		"status": {"published"},
	}, true)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if loc := rec.Header().Get("HX-Redirect"); loc != "/admin/posts/" {
		t.Errorf("HX-Redirect = %q, want /admin/posts/", loc)
	}

	rec = doJSON(t, h, http.MethodGet, "/posts/launch-day/", nil)
	if rec.Code != http.StatusOK {
		t.Errorf("saved post not reachable by its derived slug: status = %d", rec.Code)
	}
}

func TestSavePostPlainFormRedirects(t *testing.T) {
	h := newTestRouter(newHandlerTestDB(t))

	rec := doForm(t, h, "/admin/posts/save/", url.Values{
		"title":  {"No JavaScript"},
		"status": {"published"},
	}, false)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if loc := rec.Header().Get("Location"); loc != "/admin/posts/" {
		t.Errorf("Location = %q, want /admin/posts/", loc)
	}

	rec = doJSON(t, h, http.MethodGet, "/posts/no-javascript/", nil)
	if rec.Code != http.StatusOK {
		t.Errorf("post saved without HTMX is not published: status = %d", rec.Code)
	}
}

func TestSavePostUpdatesExistingPost(t *testing.T) {
	h := newTestRouter(newHandlerTestDB(t))
	id := createPost(t, h, map[string]string{"title": "Before"})

	rec := doForm(t, h, "/admin/posts/save/", url.Values{
		"id":      {fmt.Sprintf("%d", id)},
		"title":   {"After"},
		"content": {"<p>edited</p>"},
		"status":  {"published"},
	}, true)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	post := fetchPost(t, h, id)
	if post["title"] != "After" {
		t.Errorf("title = %v, want After", post["title"])
	}
	if post["status"] != "published" {
		t.Errorf("status = %v, want published", post["status"])
	}
	if post["content"] != "<p>edited</p>" {
		t.Errorf("content = %v, want <p>edited</p>", post["content"])
	}
	// The rule matches creation: an omitted slug is derived from the title.
	if post["slug"] != "after" {
		t.Errorf("slug = %v, want after", post["slug"])
	}
}

func TestSavePostRequiresTitle(t *testing.T) {
	h := newTestRouter(newHandlerTestDB(t))

	rec := doForm(t, h, "/admin/posts/save/", url.Values{"title": {""}}, true)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Title is required.") {
		t.Errorf("error fragment missing; body: %s", body)
	}
	if strings.Contains(body, "<!DOCTYPE html>") {
		t.Error("HTMX failure returned a whole page instead of a fragment")
	}
}

func TestSavePostPlainFormFailureRendersEditor(t *testing.T) {
	h := newTestRouter(newHandlerTestDB(t))

	rec := doForm(t, h, "/admin/posts/save/", url.Values{"title": {""}}, false)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Title is required.") {
		t.Error("editor page does not show the validation error")
	}
	if !strings.Contains(body, `action="/admin/posts/save/"`) {
		t.Error("failure did not re-render the editor form")
	}
}

func TestSavePostInvalidFormIDIsBadRequest(t *testing.T) {
	h := newTestRouter(newHandlerTestDB(t))

	rec := doForm(t, h, "/admin/posts/save/", url.Values{
		"id":    {"notanumber"},
		"title": {"Broken"},
	}, true)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}
