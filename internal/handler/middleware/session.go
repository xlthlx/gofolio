package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"github.com/gorilla/sessions"
)

// Sessions owns cookie-backed session state for the application.
type Sessions struct {
	store *sessions.CookieStore
}

const (
	SessionName   = "gofolio_session"
	SessionMaxAge = 86400 // 24 hours
	TokenLength   = 32
)

// NewSessions builds the cookie store with the given secret key.
func NewSessions(secretKey []byte) *Sessions {
	store := sessions.NewCookieStore(secretKey)
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   SessionMaxAge,
		HttpOnly: true,
		Secure:   false, // Set true in production with HTTPS
	}
	return &Sessions{store: store}
}

func (s *Sessions) GetSession(r *http.Request) *sessions.Session {
	session, _ := s.store.Get(r, SessionName)
	return session
}

func (s *Sessions) SetSession(w http.ResponseWriter, r *http.Request, userID string, isAdmin bool) error {
	session := s.GetSession(r)
	session.Values["user_id"] = userID
	session.Values["is_admin"] = isAdmin
	session.Options.MaxAge = SessionMaxAge
	return session.Save(r, w)
}

func (s *Sessions) ClearSession(w http.ResponseWriter, r *http.Request) error {
	session := s.GetSession(r)
	session.Values = make(map[interface{}]interface{})
	session.Options.MaxAge = -1
	return session.Save(r, w)
}

func (s *Sessions) IsAuthenticated(r *http.Request) bool {
	session := s.GetSession(r)
	isAdmin, ok := session.Values["is_admin"].(bool)
	return ok && isAdmin
}

// GetUserID returns the session identity (the username), or an empty string.
func (s *Sessions) GetUserID(r *http.Request) string {
	session := s.GetSession(r)
	id, ok := session.Values["user_id"].(string)
	if !ok {
		return ""
	}
	return id
}

func GenerateSecureToken() (string, error) {
	b := make([]byte, TokenLength)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
