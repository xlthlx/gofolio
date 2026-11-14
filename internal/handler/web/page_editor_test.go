package web_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// createPage seeds a page through the JSON API and returns its id.
func createPage(t *testing.T, h http.Handler, fields map[string]string) int64 {
	t.Helper()
	rec := doJSON(t, h, http.MethodPost, "/admin/api/pages/", fields)
	if rec.Code != http.StatusCreated {
		t.Fatalf("seeding page: status = %d, body: %s", rec.Code, rec.Body.String())
	}
	list := doJSON(t, h, http.MethodGet, "/admin/api/pages/", nil)
	var pages []struct {
		ID    int64  `json:"id"`
		Title string `json:"title"`
	}
	if err := json.Unmarshal(list.Body.Bytes(), &pages); err != nil {
		t.Fatalf("listing seeded pages: %v", err)
	}
	if len(pages) != 1 {
		t.Fatalf("seeded %d pages, want 1", len(pages))
	}
	return pages[0].ID
}

// fetchPage reads a page back through the admin API.
func fetchPage(t *testing.T, h http.Handler, id int64) map[string]interface{} {
	t.Helper()
	rec := doJSON(t, h, http.MethodGet, fmt.Sprintf("/admin/api/pages/%d/", id), nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("reading page %d: status = %d", id, rec.Code)
	}
	var page map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
		t.Fatalf("decoding page %d: %v", id, err)
	}
	return page
}

// onlyPageID returns the id of the only page stored, for the cases where a
// page was created through the editor form rather than through createPage.
func onlyPageID(t *testing.T, h http.Handler) int64 {
	t.Helper()
	list := doJSON(t, h, http.MethodGet, "/admin/api/pages/", nil)
	var pages []struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(list.Body.Bytes(), &pages); err != nil {
		t.Fatalf("listing pages: %v", err)
	}
	if len(pages) != 1 {
		t.Fatalf("stored %d pages, want 1", len(pages))
	}
	return pages[0].ID
}

func TestPageEditorRendersEmptyForm(t *testing.T) {
	h := newTestRouter(newHandlerTestDB(t))

	rec := doJSON(t, h, http.MethodGet, "/admin/pages/new/", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	for _, want := range []string{
		`action="/admin/pages/save/"`,
		`hx-post="/admin/pages/save/"`,
		`name="id"`,
		`name="title"`,
		`name="slug"`,
		`name="content"`,
		`name="status"`,
		`tinymce.min.js`,
	} {
		if !strings.Contains(rec.Body.String(), want) {
			t.Errorf("new page editor is missing %s", want)
		}
	}
}

func TestPageEditorRendersStoredValues(t *testing.T) {
	h := newTestRouter(newHandlerTestDB(t))
	id := createPage(t, h, map[string]string{
		"title":   "About Us",
		"content": "<p>version one</p>",
		"status":  "published",
	})

	rec := doJSON(t, h, http.MethodGet, fmt.Sprintf("/admin/pages/%d/edit/", id), nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	for _, want := range []string{
		`value="About Us"`,
		`&lt;p&gt;version one&lt;/p&gt;`,
		`value="published" selected`,
	} {
		if !strings.Contains(rec.Body.String(), want) {
			t.Errorf("page editor does not render %s", want)
		}
	}
}

func TestPageEditorUnknownIDIsNotFound(t *testing.T) {
	h := newTestRouter(newHandlerTestDB(t))

	rec := doJSON(t, h, http.MethodGet, "/admin/pages/4242/edit/", nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestSavePageStoresStatusAndSlug(t *testing.T) {
	h := newTestRouter(newHandlerTestDB(t))

	rec := doForm(t, h, "/admin/pages/save/", url.Values{
		"title":   {"About Us"},
		"content": {"<p>version one</p>"},
		"status":  {"published"},
	}, true)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if loc := rec.Header().Get("HX-Redirect"); loc != "/admin/pages/" {
		t.Errorf("HX-Redirect = %q, want /admin/pages/", loc)
	}

	page := fetchPage(t, h, onlyPageID(t, h))
	if page["status"] != "published" {
		t.Errorf("status = %v, want published; the editor's status select was ignored", page["status"])
	}
	if page["slug"] != "about-us" {
		t.Errorf("slug = %v, want about-us", page["slug"])
	}
	if updatedAt, ok := page["updated_at"].(string); !ok || updatedAt == "" {
		t.Errorf("updated_at = %v, want a timestamp", page["updated_at"])
	}
}

func TestSavePageUpdatesExistingPage(t *testing.T) {
	h := newTestRouter(newHandlerTestDB(t))
	id := createPage(t, h, map[string]string{"title": "Draft Page"})

	rec := doForm(t, h, "/admin/pages/save/", url.Values{
		"id":      {fmt.Sprintf("%d", id)},
		"title":   {"Draft Page"},
		"slug":    {"draft-page"},
		"content": {"<p>version two</p>"},
		"status":  {"published"},
	}, true)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	page := fetchPage(t, h, id)
	if page["content"] != "<p>version two</p>" {
		t.Errorf("content = %v, want <p>version two</p>", page["content"])
	}
	if page["status"] != "published" {
		t.Errorf("status = %v, want published", page["status"])
	}
}

func TestSavePageRequiresTitle(t *testing.T) {
	h := newTestRouter(newHandlerTestDB(t))

	rec := doForm(t, h, "/admin/pages/save/", url.Values{"title": {""}}, true)
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

func TestSavePagePlainFormRedirects(t *testing.T) {
	h := newTestRouter(newHandlerTestDB(t))

	rec := doForm(t, h, "/admin/pages/save/", url.Values{"title": {"No JavaScript"}}, false)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if loc := rec.Header().Get("Location"); loc != "/admin/pages/" {
		t.Errorf("Location = %q, want /admin/pages/", loc)
	}

	page := fetchPage(t, h, onlyPageID(t, h))
	if page["title"] != "No JavaScript" {
		t.Errorf("title = %v, want No JavaScript", page["title"])
	}
	if page["status"] != "draft" {
		t.Errorf("status = %v, want draft for a form that sent no status", page["status"])
	}
}

func TestPagesTableShowsStatus(t *testing.T) {
	h := newTestRouter(newHandlerTestDB(t))
	createPage(t, h, map[string]string{"title": "Listed Page", "status": "published"})

	rec := doJSON(t, h, http.MethodGet, "/admin/pages/table/", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if body := rec.Body.String(); !strings.Contains(body, "published") {
		t.Errorf("pages table does not show the stored status; body:\n%s", body)
	}
}
