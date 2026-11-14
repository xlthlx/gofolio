package web_test

import (
	"net/http"
	"testing"
)

func TestPublicGetPostVisibility(t *testing.T) {
	h := newTestRouter(newHandlerTestDB(t))

	doJSON(t, h, http.MethodPost, "/api/posts/",
		map[string]string{"title": "Hidden Draft", "status": "draft"})
	rec := doJSON(t, h, http.MethodGet, "/posts/hidden-draft/", nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("draft visible publicly: status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
