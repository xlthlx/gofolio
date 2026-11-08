package model

type User struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"-"` // Never expose password
	IsAdmin  bool   `json:"is_admin"`
}
