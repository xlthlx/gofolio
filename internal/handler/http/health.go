package web

import (
	"database/sql"
	"net/http"
)

// Health reports liveness; a database ping makes it meaningful, since
// nearly every route in goFolio depends on SQLite being reachable.
type Health struct{ DB *sql.DB }

func (h Health) Check(w http.ResponseWriter, r *http.Request) {
	if err := h.DB.Ping(); err != nil {
		http.Error(w, `{"status":"unhealthy"}`, http.StatusServiceUnavailable)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
