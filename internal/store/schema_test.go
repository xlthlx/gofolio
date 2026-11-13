package store

import "testing"

func TestMigrateCreatesUsersTable(t *testing.T) {
	db := newTestDB(t) // already runs Open + Migrate
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count); err != nil {
		t.Fatalf("users table missing after Migrate: %v", err)
	}
}
