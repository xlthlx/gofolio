package web

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
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

// isHTMX reports whether the request came from HTMX. Handlers use it to decide
// between a fragment and a whole page, or between HX-Redirect and a redirect.
func isHTMX(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

// redirectAfterSave answers an HTMX request with HX-Redirect and a plain form
// post with a conventional redirect, so a save works with or without
// JavaScript.
func redirectAfterSave(w http.ResponseWriter, r *http.Request, target string) {
	if isHTMX(r) {
		w.Header().Set("HX-Redirect", target)
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}

// parseID reads a required {id} route parameter.
func parseID(r *http.Request) (int64, error) {
	return strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
}

// routeID reads an optional {id} route parameter and reports zero when the
// route has none. The editor routes share one handler between /new/ and
// /{id}/edit/, so absence means "new record" rather than "invalid".
func routeID(r *http.Request) (int64, error) {
	raw := strings.TrimSpace(chi.URLParam(r, "id"))
	if raw == "" {
		return 0, nil
	}
	return strconv.ParseInt(raw, 10, 64)
}

// formID reads the id an editor form carries, and reports zero for a new
// record. Templates render the empty id of an unsaved row as zero.
func formID(r *http.Request) (int64, error) {
	raw := strings.TrimSpace(r.PostFormValue("id"))
	if raw == "" {
		return 0, nil
	}
	return strconv.ParseInt(raw, 10, 64)
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
