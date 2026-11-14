package web

import (
	"database/sql"
	"encoding/json"
	"io"
	"net/http"

	"gofolio/internal/handler/middleware"
	"gofolio/internal/service"
)

// Auth handles the public authentication endpoints.
type Auth struct {
	DB   *sql.DB
	Sess *middleware.Sessions
}

func (h Auth) Login(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var creds struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.Unmarshal(body, &creds); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := service.Login(h.DB, h.Sess, w, r, creds.Username, creds.Password); err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Invalid credentials"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"success": "logged in"})
}

func (h Auth) Logout(w http.ResponseWriter, r *http.Request) {
	if err := h.Sess.ClearSession(w, r); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"success": "logged out"})
}

func (h Auth) Check(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"authenticated": h.Sess.IsAuthenticated(r)})
}
