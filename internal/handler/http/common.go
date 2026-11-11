package web

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

const (
	AdminThemeDir  = "web/themes/admin/default"
	PublicThemeDir = "web/themes/public/default"
)

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func decodeJSON(r *http.Request, v interface{}) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	defer r.Body.Close()
	return json.Unmarshal(body, v)
}

// ServeStatic serves public theme assets under /static/.
func ServeStatic(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/static/")
	http.ServeFile(w, r, PublicThemeDir+"/"+path)
}

// ServeAdminStatic serves admin theme assets under /admin/static/.
func ServeAdminStatic(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/admin/static/")
	http.ServeFile(w, r, AdminThemeDir+"/"+path)
}
