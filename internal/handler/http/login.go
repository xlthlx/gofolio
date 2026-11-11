package web

import (
	"net/http"

	"gofolio/internal/handler/middleware"
	"gofolio/internal/service"
)

// loginData feeds the login page template.
type loginData struct {
	Error string
}

// LoginGet serves the admin login page. Already-authenticated visitors are
// sent straight to the dashboard. This route also fixes a long-standing gap:
// RequireAuth always redirected here, but the page was never routed.
func (h Admin) LoginGet(w http.ResponseWriter, r *http.Request) {
	if middleware.IsAuthenticated(r) {
		http.Redirect(w, r, "/admin/", http.StatusSeeOther)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.Tpl.LoginPage(w, loginData{}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// LoginPost authenticates form submissions. HTMX requests get an error
// fragment on failure and an HX-Redirect on success; plain form posts get
// conventional status codes, so the page also works without JavaScript.
func (h Admin) LoginPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	username := r.PostFormValue("username")
	password := r.PostFormValue("password")

	hx := r.Header.Get("HX-Request") == "true"

	if err := service.Login(h.DB, w, r, username, password); err != nil {
		if hx {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusUnauthorized)
			h.Tpl.LoginError(w, loginData{Error: "Invalid username or password."})
			return
		}
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	if hx {
		w.Header().Set("HX-Redirect", "/admin/")
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/admin/", http.StatusSeeOther)
}
