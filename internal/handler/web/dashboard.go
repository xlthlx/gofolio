package web

import (
	"net/http"

	"gofolio/internal/model"
)

// DashboardData feeds the dashboard page and its recent-posts fragment.
type DashboardData struct {
	TotalPosts  int
	Published   int
	Drafts      int
	TotalPages  int
	RecentPosts []model.Post
}

func (h Admin) dashboardData() DashboardData {
	posts, _ := h.posts().ListAll()
	pages, _ := h.pages().ListAll()

	d := DashboardData{TotalPosts: len(posts), TotalPages: len(pages)}
	for _, p := range posts {
		switch p.Status {
		case "published":
			d.Published++
		case "draft":
			d.Drafts++
		}
	}
	if len(posts) > 5 {
		d.RecentPosts = posts[:5]
	} else {
		d.RecentPosts = posts
	}
	return d
}

// DashboardPage renders the full dashboard with stats and recent posts
// computed server-side; no client fetch on load.
func (h Admin) DashboardPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.Tpl.DashboardPage(w, h.dashboardData()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// RecentPosts renders only the recent-posts rows, for HTMX polling or
// targeted refreshes.
func (h Admin) RecentPosts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.Tpl.RecentRows(w, h.dashboardData()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
