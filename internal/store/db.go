package store

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", path+"?_foreign_keys=on")
	if err != nil {
		return nil, err
	}
	// SQLite: a single connection avoids "database is locked" under concurrency.
	db.SetMaxOpenConns(1)
	return db, nil
}

const migrations = `
CREATE TABLE IF NOT EXISTS posts (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	title TEXT NOT NULL,
	slug TEXT UNIQUE NOT NULL,
	content TEXT,
	status TEXT DEFAULT 'draft',
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS pages (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	title TEXT NOT NULL,
	slug TEXT UNIQUE NOT NULL,
	content TEXT,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
`

func Migrate(db *sql.DB) error {
	_, err := db.Exec(migrations)
	return err
}
