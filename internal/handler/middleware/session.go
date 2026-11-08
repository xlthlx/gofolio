package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"github.com/gorilla/sessions"
)

var store *sessions.CookieStore

const (
	SessionName   = "gofolio_session"
	SessionMaxAge = 86400 // 24 hours
	TokenLength   = 32
)

func Init(secretKey []byte) {
	store = sessions.NewCookieStore(secretKey)
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   SessionMaxAge,
		HttpOnly: true,
		Secure:   false, // Set true in production with HTTPS
	}
}

func GetSession(r *http.Request) *sessions.Session {
	session, _ := store.Get(r, SessionName)
	return session
}

func SetSession(w http.ResponseWriter, r *http.Request, userID string, isAdmin bool) error {
	session := GetSession(r)
	session.Values["user_id"] = userID
	session.Values["is_admin"] = isAdmin
	session.Options.MaxAge = SessionMaxAge
	return session.Save(r, w)
}

func ClearSession(w http.ResponseWriter, r *http.Request) error {
	session := GetSession(r)
	session.Values = make(map[interface{}]interface{})
	session.Options.MaxAge = -1
	return session.Save(r, w)
}

func IsAuthenticated(r *http.Request) bool {
	session := GetSession(r)
	isAdmin, ok := session.Values["is_admin"].(bool)
	return ok && isAdmin
}

func GenerateSecureToken() (string, error) {
	b := make([]byte, TokenLength)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
