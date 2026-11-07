package auth

import (
	"database/sql"
	"errors"
	"net/http"
)

type User struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"-"` // Never expose password
	IsAdmin  bool   `json:"is_admin"`
}

func CreateUser(db *sql.DB, username, email, password string, isAdmin bool) (int64, error) {
	hash, err := HashPassword(password)
	if err != nil {
		return 0, err
	}

	result, err := db.Exec(
		`INSERT INTO users (username, email, password_hash, is_admin) VALUES (?, ?, ?, ?)`,
		username, email, hash, isAdmin,
	)
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

func GetUserByUsername(db *sql.DB, username string) (*User, error) {
	var u User
	var passwordHash string

	err := db.QueryRow(
		`SELECT id, username, email, password_hash, is_admin FROM users WHERE username = ?`,
		username,
	).Scan(&u.ID, &u.Username, &u.Email, &passwordHash, &u.IsAdmin)

	if err == sql.ErrNoRows {
		return nil, errors.New("user not found")
	}
	if err != nil {
		return nil, err
	}

	u.Password = passwordHash
	return &u, nil
}

func Login(db *sql.DB, w http.ResponseWriter, r *http.Request, username, password string) error {
	user, err := GetUserByUsername(db, username)
	if err != nil {
		return err
	}

	if !CheckPasswordHash(password, user.Password) {
		return errors.New("invalid credentials")
	}

	return SetSession(w, r, user.Username, user.IsAdmin)
}

func InitUsersTable(db *sql.DB) error {
	_, err := db.Exec(`
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		email TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		is_admin INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	return err
}

func CreateDefaultAdmin(db *sql.DB) error {
	// Check if admin exists
	var count int
	db.QueryRow("SELECT COUNT(*) FROM users WHERE is_admin = 1").Scan(&count)

	if count == 0 {
		defaultPassword := "admin123"
		_, err := CreateUser(db, "admin", "admin@local", defaultPassword, true)
		return err
	}
	return nil
}
