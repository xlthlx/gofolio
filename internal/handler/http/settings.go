package web

import (
	"net/http"

	"gofolio/internal/handler/middleware"
	"gofolio/internal/service"
)

// SettingsPage renders the settings page. Site title and tagline have no
// database backing yet, so they render as fixed text, not editable inputs.
func (h Admin) SettingsPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.Tpl.SettingsPage(w, nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// ChangePassword handles the HTMX password-change form. On failure, it swaps
// an error message into the page; on success a confirmation. Works as a
// plain form post too.
func (h Admin) ChangePassword(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	newPw := r.PostFormValue("new_password")
	confirm := r.PostFormValue("confirm_password")

	failure := func(msg string, code int) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(code)
		w.Write([]byte(`<p style="color:#b00020;">` + msg + `</p>`))
	}

	if len(newPw) < 8 {
		failure("Password must be at least 8 characters.", http.StatusBadRequest)
		return
	}
	if newPw != confirm {
		failure("Passwords do not match.", http.StatusBadRequest)
		return
	}

	userID := middleware.GetUserID(r)
	if userID == "" {
		failure("No active session.", http.StatusUnauthorized)
		return
	}
	if err := service.ChangePassword(h.DB, userID, newPw); err != nil {
		failure("Could not update password: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(`<p style="color:#2e7d32;">Password updated.</p>`))
}
