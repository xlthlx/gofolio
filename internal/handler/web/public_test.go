package web_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// public_test covers the JSON API the public site also publishes, at the /api
// paths. The rendered views are covered in site_test.go.

func TestPublicListPostsReturnsPublishedOnly(t *testing.T) {
	h := newTestRouter(newHandlerTestDB(t))

	seedPost(t, h, "Live Post", "published", "<p>live</p>")
	seedPost(t, h, "Draft Post", "draft", "<p>draft</p>")

	rec := doJSON(t, h, http.MethodGet, "/api/posts/", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var feed struct {
		Posts []map[string]interface{} `json:"posts"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &feed); err != nil {
		t.Fatalf("decoding feed: %v", err)
	}
	if len(feed.Posts) != 1 || feed.Posts[0]["title"] != "Live Post" {
		t.Fatalf("feed = %+v, want only the published post", feed.Posts)
	}
}

func TestPublicGetPostVisibility(t *testing.T) {
	h := newTestRouter(newHandlerTestDB(t))

	seedPost(t, h, "Hidden Draft", "draft", "<p>hidden</p>")

	rec := doJSON(t, h, http.MethodGet, "/api/posts/hidden-draft/", nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("draft visible publicly: status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestPublicGetPostBySlug(t *testing.T) {
	h := newTestRouter(newHandlerTestDB(t))
	slug, _ := seedPost(t, h, "Readable Post", "published", "<p>readable</p>")

	rec := doJSON(t, h, http.MethodGet, "/api/posts/"+slug+"/", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("Content-Type = %q, want JSON", ct)
	}
	var post map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &post); err != nil {
		t.Fatalf("decoding post: %v", err)
	}
	if post["title"] != "Readable Post" {
		t.Errorf("title = %v, want Readable Post", post["title"])
	}
}

func TestPublicGetPageBySlug(t *testing.T) {
	h := newTestRouter(newHandlerTestDB(t))

	rec := doJSON(t, h, http.MethodPost, "/admin/api/pages/", map[string]string{
		"title":   "About Us",
		"content": "<p>about</p>",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("seeding page: status = %d", rec.Code)
	}

	rec = doJSON(t, h, http.MethodGet, "/api/pages/about-us/", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var page map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
		t.Fatalf("decoding page: %v", err)
	}
	if page["title"] != "About Us" {
		t.Errorf("title = %v, want About Us", page["title"])
	}

	rec = doJSON(t, h, http.MethodGet, "/api/pages/ghost/", nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("unknown page: status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
