package web_test

import (
	"net/http"
	"strings"
	"testing"
)

func TestDashboardPageRendersStats(t *testing.T) {
	h := newTestRouter(newHandlerTestDB(t))

	doJSON(t, h, http.MethodPost, "/admin/api/posts/", map[string]string{"title": "Shown Here"})
	rec := doJSON(t, h, http.MethodGet, "/admin/dashboard/", nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Shown Here") {
		t.Errorf("dashboard does not contain recent post title; body:\n%s", body)
	}
	if !strings.Contains(body, "Total Posts") {
		t.Errorf("dashboard does not contain stats; body:\n%s", body)
	}
}
