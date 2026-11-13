package store

import (
	"database/sql"
	"errors"
)

// ErrSettingNotFound is returned when a key has no row.
var ErrSettingNotFound = errors.New("setting not found")

// SettingStore persists site settings as key/value rows.
type SettingStore struct{ DB *sql.DB }

// All returns every setting, ordered by key for stable rendering.
// A missing table or empty result yields a nil map, not an error, so
// templates can treat settings as optional.
func (s SettingStore) All() (map[string]string, error) {
	rows, err := s.DB.Query(`SELECT key, value FROM settings ORDER BY key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[string]string{}
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		out[k] = v
	}
	return out, rows.Err()
}

// Get returns one setting's value; ErrSettingNotFound when absent.
func (s SettingStore) Get(key string) (string, error) {
	var v string
	err := s.DB.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrSettingNotFound
	}
	return v, err
}

// Set inserts or updates one key. UPSERT keeps it a single statement.
func (s SettingStore) Set(key, value string) error {
	_, err := s.DB.Exec(
		`INSERT INTO settings (key, value) VALUES (?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
	return err
}
