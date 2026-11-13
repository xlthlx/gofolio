package store

import (
	"errors"
	"testing"
)

func TestSettingsSetGetRoundTrip(t *testing.T) {
	db := newTestDB(t)
	ss := SettingStore{DB: db}

	if err := ss.Set("site_title", "My Portfolio"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got, err := ss.Get("site_title")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != "My Portfolio" {
		t.Errorf("Get = %q, want %q", got, "My Portfolio")
	}
}

func TestSettingsGetMissing(t *testing.T) {
	db := newTestDB(t)
	ss := SettingStore{DB: db}

	if _, err := ss.Get("nope"); !errors.Is(err, ErrSettingNotFound) {
		t.Errorf("Get on missing key = %v, want ErrSettingNotFound", err)
	}
}

func TestSettingsSetOverwrites(t *testing.T) {
	db := newTestDB(t)
	ss := SettingStore{DB: db}

	if err := ss.Set("tagline", "first"); err != nil {
		t.Fatalf("first Set: %v", err)
	}
	if err := ss.Set("tagline", "second"); err != nil {
		t.Fatalf("second Set: %v", err)
	}
	got, err := ss.Get("tagline")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != "second" {
		t.Errorf("Get = %q, want overwritten value %q", got, "second")
	}
}

func TestSettingsAllEmptyAndPopulated(t *testing.T) {
	db := newTestDB(t)
	ss := SettingStore{DB: db}

	all, err := ss.All()
	if err != nil {
		t.Fatalf("All on empty table: %v", err)
	}
	if len(all) != 0 {
		t.Errorf("All on empty table returned %d entries, want 0", len(all))
	}

	want := map[string]string{"site_title": "T1", "tagline": "T2"}
	for k, v := range want {
		if err := ss.Set(k, v); err != nil {
			t.Fatalf("Set %s: %v", k, err)
		}
	}
	all, err = ss.All()
	if err != nil {
		t.Fatalf("All: %v", err)
	}
	for k, v := range want {
		if all[k] != v {
			t.Errorf("All[%s] = %q, want %q", k, all[k], v)
		}
	}
}
