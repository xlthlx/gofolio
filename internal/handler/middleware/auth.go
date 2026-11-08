package middleware

import (
	"net/http"
)

func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !IsAuthenticated(r) {
			http.Redirect(w, r, "/admin/login/", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}
