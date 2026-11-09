// Package web contains the HTTP handlers for goFolio. The folder is
// handler/http per docs/folder-structure.md; the package is named "web"
// because the package identifier must not shadow net/http.
package web

import (
	"encoding/json"
	"io"
	"net/http"
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
