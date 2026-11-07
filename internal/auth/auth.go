package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	. "net/http"

	"github.com/gorilla/sessions"
)

var store *sessions.CookieStore

const (
	SessionName    = "gopress_session"
	SessionMaxAge  = 86400 // 24 hours
	PasswordMinLen = 8
	TokenLength    = 32
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

func GetSession(r *Request) *sessions.Session {
	session, _ := store.Get(r, SessionName)
	return session
}

func SetSession(w ResponseWriter, r *Request, userID string, isAdmin bool) error {
	session := GetSession(r)
	session.Values["user_id"] = userID
	session.Values["is_admin"] = isAdmin
	session.Options.MaxAge = SessionMaxAge
	return session.Save(r, w)
}

func ClearSession(w ResponseWriter, r *Request) error {
	session := GetSession(r)
	session.Values = make(map[interface{}]interface{})
	session.Options.MaxAge = -1
	return session.Save(r, w)
}

func IsAuthenticated(r *Request) bool {
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

func HashPassword(password string) (string, error) {
	// Simple hash for demo - use bcrypt in production
	h := make([]byte, 32)
	hash := sha256.Sum256([]byte(password))
	copy(h, hash[:])
	return hex.EncodeToString(h), nil
}

func CheckPasswordHash(password, hash string) bool {
	// Simple check for demo - use bcrypt in production
	h, _ := HashPassword(password)
	return h == hash
}
