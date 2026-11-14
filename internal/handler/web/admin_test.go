package web_test

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"gofolio/internal/handler/web"
)

func TestCreatePostRequiresTitle(t *testing.T) {
	h := newTestRouter(newHandlerTestDB(t))

	rec := doJSON(t, h, http.MethodPost, "/api/posts/", map[string]string{"title": ""})
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestCreatePostDefaultsSlugAndStatus(t *testing.T) {
	h := newTestRouter(newHandlerTestDB(t))

	rec := doJSON(t, h, http.MethodPost, "/api/posts/", map[string]string{"title": "Hello World"})
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	var got struct {
		Slug   string `json:"slug"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decoding: %v", err)
	}
	if got.Slug != "hello-world" {
		t.Errorf("slug = %q, want hello-world", got.Slug)
	}
	if got.Status != "draft" {
		t.Errorf("status = %q, want draft", got.Status)
	}
}

func TestUpdatePostInvalidID(t *testing.T) {
	h := newTestRouter(newHandlerTestDB(t))

	rec := doJSON(t, h, http.MethodPut, "/api/posts/notanumber/", map[string]string{"title": "x"})
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestDeletePostRoundTrip(t *testing.T) {
	h := newTestRouter(newHandlerTestDB(t))

	create := doJSON(t, h, http.MethodPost, "/api/posts/", map[string]string{"title": "Doomed"})
	var created struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(create.Body.Bytes(), &created); err != nil {
		t.Fatalf("decoding created post: %v", err)
	}

	del := doJSON(t, h, http.MethodDelete, fmt.Sprintf("/api/posts/%d/", created.ID), nil)
	if del.Code != http.StatusNoContent {
		t.Errorf("delete status = %d, want %d", del.Code, http.StatusNoContent)
	}

	get := doJSON(t, h, http.MethodGet, "/posts/doomed/", nil)
	if get.Code != http.StatusNotFound {
		t.Errorf("deleted post still visible: status = %d, want %d", get.Code, http.StatusNotFound)
	}
}

func TestPostsTableFragment(t *testing.T) {
	h := newTestRouter(newHandlerTestDB(t))

	doJSON(t, h, http.MethodPost, "/api/posts/", map[string]string{"title": "Fragment Me"})
	rec := doJSON(t, h, http.MethodGet, "/admin/posts/table/", nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if body := rec.Body.String(); !strings.Contains(body, "Fragment Me") {
		t.Errorf("fragment does not contain the post title; body:\n%s", body)
	}
}

func TestPagesTableFragment(t *testing.T) {
	h := newTestRouter(newHandlerTestDB(t))

	doJSON(t, h, http.MethodPost, "/admin/api/pages/", map[string]string{"title": "About Fragment"})
	rec := doJSON(t, h, http.MethodGet, "/admin/pages/table/", nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if body := rec.Body.String(); !strings.Contains(body, "About Fragment") {
		t.Errorf("fragment does not contain page title; body:\n%s", body)
	}
}

func newTestRouterWithPages(db *sql.DB) http.Handler {
	a := &web.Admin{DB: db, Tpl: testRenderer, Sess: testSess}
	r := chi.NewRouter()
	r.Post("/admin/api/pages/", a.CreatePage)
	r.Get("/admin/api/pages/", a.ListPages)
	r.Get("/api/pages/{id}/", a.GetPage)
	r.Put("/api/pages/{id}/", a.UpdatePage)
	r.Delete("/api/pages/{id}/", a.DeletePage)
	r.Get("/admin/pages/table/", a.PagesTable)
	return r
}

func TestPageEditorRoundTrip(t *testing.T) {
	h := newTestRouterWithPages(newHandlerTestDB(t))

	create := doJSON(t, h, http.MethodPost, "/admin/api/pages/",
		map[string]string{"title": "About Us", "content": "<p>v1</p>"})
	if create.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body: %s", create.Code, create.Body.String())
	}
	var created struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(create.Body.Bytes(), &created); err != nil {
		t.Fatalf("decoding create: %v", err)
	}

	// The list API is how the editor discovers the id (Create returns only
	// status); fetch it, then exercise get-one, update and get again.
	list := doJSON(t, h, http.MethodGet, "/admin/api/pages/", nil)
	var pages []struct {
		ID      int64  `json:"id"`
		Title   string `json:"title"`
		Slug    string `json:"slug"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(list.Body.Bytes(), &pages); err != nil || len(pages) != 1 {
		t.Fatalf("listing pages: %v (%d pages)", err, len(pages))
	}
	id := pages[0].ID
	if pages[0].Slug != "about-us" {
		t.Errorf("derived slug = %q, want about-us", pages[0].Slug)
	}
	rec := doJSON(t, h, http.MethodGet, "/admin/pages/table/", nil)
	if !strings.Contains(rec.Body.String(), "About Us") {
		t.Error("table fragment does not render created page")
	}

	got := doJSON(t, h, http.MethodGet, fmt.Sprintf("/api/pages/%d/", id), nil)
	if got.Code != http.StatusOK {
		t.Fatalf("get status = %d, want %d", got.Code, http.StatusOK)
	}
	var page struct {
		Content string `json:"content"`
	}
	if err := json.Unmarshal(got.Body.Bytes(), &page); err != nil {
		t.Fatalf("decoding page: %v", err)
	}
	if page.Content != "<p>v1</p>" {
		t.Errorf("content = %q, want <p>v1</p>", page.Content)
	}

	upd := doJSON(t, h, http.MethodPut, fmt.Sprintf("/api/pages/%d/", id),
		map[string]string{"title": "About Us", "content": "<p>v2</p>"})
	if upd.Code != http.StatusOK {
		t.Fatalf("update status = %d, body: %s", upd.Code, upd.Body.String())
	}
	got2 := doJSON(t, h, http.MethodGet, fmt.Sprintf("/api/pages/%d/", id), nil)
	var page2 struct {
		Content string `json:"content"`
	}
	json.Unmarshal(got2.Body.Bytes(), &page2)
	if page2.Content != "<p>v2</p>" {
		t.Errorf("content after update = %q, want <p>v2</p>", page2.Content)
	}

	del := doJSON(t, h, http.MethodDelete, fmt.Sprintf("/api/pages/%d/", id), nil)
	if del.Code != http.StatusNoContent {
		t.Errorf("delete status = %d, want %d", del.Code, http.StatusNoContent)
	}
	final := doJSON(t, h, http.MethodGet, fmt.Sprintf("/api/pages/%d/", id), nil)
	if final.Code != http.StatusNotFound {
		t.Errorf("deleted page still fetchable: status = %d, want %d", final.Code, http.StatusNotFound)
	}
}

func TestUpdatePageInvalidID(t *testing.T) {
	h := newTestRouterWithPages(newHandlerTestDB(t))
	rec := doJSON(t, h, http.MethodPut, "/api/pages/notanumber/",
		map[string]string{"title": "x"})
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Hello":             "hello",
		"Launch Day":        "launch-day",
		"Hello, World!":     "hello-world",
		"  spaced  out ":    "spaced-out",
		"...":               "",
		"Città":             "citta",
		"Perché l'Economia": "perche-l-economia",
		"façade & naïve":    "facade-naive",
		"ÀÈÌÒÙ":             "aeiou",
		"Café-Core":         "cafe-core",
		"under_score.dot":   "under-score-dot",
	}
	for in, want := range cases {
		if got := web.Slugify(in); got != want {
			t.Errorf("Slugify(%q) = %q, want %q", in, got, want)
		}
	}
}
