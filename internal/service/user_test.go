package service

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"gofolio/internal/handler/middleware"

	_ "github.com/mattn/go-sqlite3"
)

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "test.db")+"?_foreign_keys=on")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })

	if err := InitUsersTable(db); err != nil {
		t.Fatalf("InitUsersTable failed: %v", err)
	}
	return db
}

func TestCreateAndGetUser(t *testing.T) {
	db := newTestDB(t)

	id, err := CreateUser(db, "alice", "alice@example.com", "password123", false)
	if err != nil {
		t.Fatalf("CreateUser returned error: %v", err)
	}
	if id <= 0 {
		t.Errorf("CreateUser returned id %d, want positive", id)
	}

	user, err := GetUserByUsername(db, "alice")
	if err != nil {
		t.Fatalf("GetUserByUsername returned error: %v", err)
	}
	if user.ID != id {
		t.Errorf("user.ID = %d, want %d", user.ID, id)
	}
	if user.Username != "alice" || user.Email != "alice@example.com" || user.IsAdmin {
		t.Errorf("unexpected user: %+v", user)
	}
	if user.Password == "" {
		t.Error("stored password hash should be retrievable for comparison")
	}
}

func TestCreateUserDuplicateUsername(t *testing.T) {
	db := newTestDB(t)

	if _, err := CreateUser(db, "bob", "bob@example.com", "password123", false); err != nil {
		t.Fatalf("first CreateUser returned error: %v", err)
	}
	if _, err := CreateUser(db, "bob", "other@example.com", "password456", false); err == nil {
		t.Error("second CreateUser with duplicate username should fail, got nil error")
	}
}

func TestGetUserByUsernameNotFound(t *testing.T) {
	db := newTestDB(t)

	_, err := GetUserByUsername(db, "ghost")
	if err == nil {
		t.Error("GetUserByUsername for missing user should return error")
	}
}

func TestCreateDefaultAdminOnlyOnce(t *testing.T) {
	db := newTestDB(t)

	if err := CreateDefaultAdmin(db); err != nil {
		t.Fatalf("first CreateDefaultAdmin returned error: %v", err)
	}

	admin, err := GetUserByUsername(db, "admin")
	if err != nil {
		t.Fatalf("default admin not found after CreateDefaultAdmin: %v", err)
	}
	if !admin.IsAdmin {
		t.Error("default admin user should have is_admin = true")
	}

	if err := CreateDefaultAdmin(db); err != nil {
		t.Errorf("second CreateDefaultAdmin should be a no-op, got error: %v", err)
	}
}

func TestLoginSuccessSetsCookie(t *testing.T) {
	db := newTestDB(t)
	middleware.Init([]byte("test-secret-key-for-service-tests"))

	if err := CreateDefaultAdmin(db); err != nil {
		t.Fatalf("CreateDefaultAdmin returned error: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login/", nil)
	rec := httptest.NewRecorder()

	if err := Login(db, rec, req, "admin", "admin123"); err != nil {
		t.Fatalf("Login returned error: %v", err)
	}

	if len(rec.Result().Cookies()) == 0 {
		t.Error("successful Login did not set a session cookie")
	}
}

func TestLoginWrongPassword(t *testing.T) {
	db := newTestDB(t)
	middleware.Init([]byte("test-secret-key-for-service-tests"))

	if err := CreateDefaultAdmin(db); err != nil {
		t.Fatalf("CreateDefaultAdmin returned error: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login/", nil)
	rec := httptest.NewRecorder()

	if err := Login(db, rec, req, "admin", "wrong-password"); err == nil {
		t.Error("Login with wrong password should return error")
	}
	if len(rec.Result().Cookies()) != 0 {
		t.Error("failed Login should not set a session cookie")
	}
}

func TestLoginUnknownUser(t *testing.T) {
	db := newTestDB(t)
	middleware.Init([]byte("test-secret-key-for-service-tests"))

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login/", nil)
	rec := httptest.NewRecorder()

	if err := Login(db, rec, req, "nobody", "whatever"); err == nil {
		t.Error("Login for unknown user should return error")
	}
}
