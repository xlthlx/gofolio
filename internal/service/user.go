package service

import (
	"database/sql"
	"errors"
	"net/http"

	"gofolio/internal/handler/middleware"
	"gofolio/internal/model"
	"gofolio/pkg/util"
)

func CreateUser(db *sql.DB, username, email, password string, isAdmin bool) (int64, error) {
	hash, err := util.HashPassword(password)
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

func GetUserByUsername(db *sql.DB, username string) (*model.User, error) {
	var u model.User
	var passwordHash string

	err := db.QueryRow(
		`SELECT id, username, email, password_hash, is_admin FROM users WHERE username = ?`,
		username,
	).Scan(&u.ID, &u.Username, &u.Email, &passwordHash, &u.IsAdmin)

	if errors.Is(err, sql.ErrNoRows) {
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

	if !util.CheckPasswordHash(password, user.Password) {
		return errors.New("invalid credentials")
	}

	return middleware.SetSession(w, r, user.Username, user.IsAdmin)
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
	var count int
	db.QueryRow("SELECT COUNT(*) FROM users WHERE is_admin = 1").Scan(&count)

	if count == 0 {
		defaultPassword := "admin123"
		_, err := CreateUser(db, "admin", "admin@local", defaultPassword, true)
		return err
	}
	return nil
}

// ChangePassword replaces the stored password hash for the user with the
// given id. Callers are responsible for session policy; existing cookies
// stay valid after a successful change.
func ChangePassword(db *sql.DB, userID, newPassword string) error {
	hash, err := util.HashPassword(newPassword)
	if err != nil {
		return err
	}
	res, err := db.Exec(`UPDATE users SET password_hash = ? WHERE username = ?`,
		hash, userID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("user not found")
	}
	return nil
}
