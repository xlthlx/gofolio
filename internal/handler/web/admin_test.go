package web_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gofolio/internal/handler/web"
)

func TestCreatePostRequiresTitle(t *testing.T) {
	h := newTestRouter(newHandlerTestDB(t))

	rec := doJSON(t, h, http.MethodPost, "/admin/api/posts/", map[string]string{"title": ""})
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestCreatePostDefaultsSlugAndStatus(t *testing.T) {
	h := newTestRouter(newHandlerTestDB(t))

	rec := doJSON(t, h, http.MethodPost, "/admin/api/posts/", map[string]string{"title": "Hello World"})
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

	rec := doJSON(t, h, http.MethodPut, "/admin/api/posts/notanumber/", map[string]string{"title": "x"})
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestDeletePostRoundTrip(t *testing.T) {
	h := newTestRouter(newHandlerTestDB(t))

	create := doJSON(t, h, http.MethodPost, "/admin/api/posts/", map[string]string{"title": "Doomed"})
	var created struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(create.Body.Bytes(), &created); err != nil {
		t.Fatalf("decoding created post: %v", err)
	}

	del := doJSON(t, h, http.MethodDelete, fmt.Sprintf("/admin/api/posts/%d/", created.ID), nil)
	if del.Code != http.StatusNoContent {
		t.Errorf("delete status = %d, want %d", del.Code, http.StatusNoContent)
	}

	get := doJSON(t, h, http.MethodGet, "/posts/doomed/", nil)
	if get.Code != http.StatusNotFound {
		t.Errorf("deleted post still visible: status = %d, want %d", get.Code, http.StatusNotFound)
	}
}

// HTMX deletes answer 200 with an empty body so hx-swap="outerHTML" can drop
// the table row; plain API clients keep the 204 that TestDeletePostRoundTrip
// covers.
func TestDeletePostHTMXReturnsEmptyOK(t *testing.T) {
	h := newTestRouter(newHandlerTestDB(t))

	create := doJSON(t, h, http.MethodPost, "/admin/api/posts/", map[string]string{"title": "Doomed"})
	var created struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(create.Body.Bytes(), &created); err != nil {
		t.Fatalf("decoding created post: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/admin/api/posts/%d/", created.ID), nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("body = %q, want an empty body", rec.Body.String())
	}
}

func TestDeletePageHTMXReturnsEmptyOK(t *testing.T) {
	h := newTestRouter(newHandlerTestDB(t))

	create := doJSON(t, h, http.MethodPost, "/admin/api/pages/", map[string]string{"title": "Doomed"})
	if create.Code != http.StatusCreated {
		t.Fatalf("seeding page: status = %d", create.Code)
	}
	list := doJSON(t, h, http.MethodGet, "/admin/api/pages/", nil)
	var pages []struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(list.Body.Bytes(), &pages); err != nil || len(pages) != 1 {
		t.Fatalf("listing pages: %v (%d pages)", err, len(pages))
	}

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/admin/api/pages/%d/", pages[0].ID), nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("body = %q, want an empty body", rec.Body.String())
	}
}

func TestPostsTableFragment(t *testing.T) {
	h := newTestRouter(newHandlerTestDB(t))

	doJSON(t, h, http.MethodPost, "/admin/api/posts/", map[string]string{"title": "Fragment Me"})
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

func TestPagesAPIRoundTrip(t *testing.T) {
	h := newTestRouter(newHandlerTestDB(t))

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
		Status  string `json:"status"`
	}
	if err := json.Unmarshal(list.Body.Bytes(), &pages); err != nil || len(pages) != 1 {
		t.Fatalf("listing pages: %v (%d pages)", err, len(pages))
	}
	id := pages[0].ID
	if pages[0].Slug != "about-us" {
		t.Errorf("derived slug = %q, want about-us", pages[0].Slug)
	}
	if pages[0].Status != "draft" {
		t.Errorf("status = %q, want draft", pages[0].Status)
	}
	rec := doJSON(t, h, http.MethodGet, "/admin/pages/table/", nil)
	if !strings.Contains(rec.Body.String(), "About Us") {
		t.Error("table fragment does not render created page")
	}

	got := doJSON(t, h, http.MethodGet, fmt.Sprintf("/admin/api/pages/%d/", id), nil)
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

	upd := doJSON(t, h, http.MethodPut, fmt.Sprintf("/admin/api/pages/%d/", id),
		map[string]string{"title": "About Us", "content": "<p>v2</p>"})
	if upd.Code != http.StatusOK {
		t.Fatalf("update status = %d, body: %s", upd.Code, upd.Body.String())
	}
	got2 := doJSON(t, h, http.MethodGet, fmt.Sprintf("/admin/api/pages/%d/", id), nil)
	var page2 struct {
		Content string `json:"content"`
	}
	json.Unmarshal(got2.Body.Bytes(), &page2)
	if page2.Content != "<p>v2</p>" {
		t.Errorf("content after update = %q, want <p>v2</p>", page2.Content)
	}

	del := doJSON(t, h, http.MethodDelete, fmt.Sprintf("/admin/api/pages/%d/", id), nil)
	if del.Code != http.StatusNoContent {
		t.Errorf("delete status = %d, want %d", del.Code, http.StatusNoContent)
	}
	final := doJSON(t, h, http.MethodGet, fmt.Sprintf("/admin/api/pages/%d/", id), nil)
	if final.Code != http.StatusNotFound {
		t.Errorf("deleted page still fetchable: status = %d, want %d", final.Code, http.StatusNotFound)
	}
}

func TestUpdatePageInvalidID(t *testing.T) {
	h := newTestRouter(newHandlerTestDB(t))
	rec := doJSON(t, h, http.MethodPut, "/admin/api/pages/notanumber/",
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
