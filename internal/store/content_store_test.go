package store

import (
	"database/sql"
	"path/filepath"
	"testing"

	"gofolio/internal/model"
)

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	t.Cleanup(func() {
		err := db.Close()
		if err != nil {
			t.Fatalf("Close failed: %v", err)
		}
	})
	if err := Migrate(db); err != nil {
		t.Fatalf("Migrate failed: %v", err)
	}
	return db
}

func TestPostStoreCreateAndList(t *testing.T) {
	ps := PostStore{DB: newTestDB(t)}

	in := &model.Post{Title: "Hello World", Slug: "hello-world"}
	if err := ps.Create(in); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if in.ID <= 0 {
		t.Errorf("ID = %d, want positive", in.ID)
	}

	all, err := ps.ListAll()
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(all) != 1 || all[0].Title != "Hello World" {
		t.Fatalf("ListAll = %+v, want 1 post titled Hello World", all)
	}
}

func TestPostStoreVisibility(t *testing.T) {
	ps := PostStore{DB: newTestDB(t)}

	if err := ps.Create(&model.Post{Title: "Draft", Slug: "draft"}); err != nil {
		t.Fatalf("Create draft: %v", err)
	}
	if err := ps.Create(&model.Post{Title: "Live", Slug: "live", Status: "published"}); err != nil {
		t.Fatalf("Create published: %v", err)
	}

	pub, err := ps.ListPublished()
	if err != nil {
		t.Fatalf("ListPublished: %v", err)
	}
	if len(pub) != 1 || pub[0].Slug != "live" {
		t.Fatalf("ListPublished = %+v, want only the published post", pub)
	}

	if _, err := ps.GetPublishedBySlug("draft"); err == nil {
		t.Error("draft should not be returned by GetPublishedBySlug")
	}
	p, err := ps.GetPublishedBySlug("live")
	if err != nil {
		t.Fatalf("GetPublishedBySlug(live): %v", err)
	}
	if p.Title != "Live" {
		t.Errorf("Title = %q, want Live", p.Title)
	}
}

func TestPostStoreUpdateAndDelete(t *testing.T) {
	ps := PostStore{DB: newTestDB(t)}

	p := &model.Post{Title: "Old", Slug: "old"}
	if err := ps.Create(p); err != nil {
		t.Fatalf("Create: %v", err)
	}

	p.Title = "New"
	p.Status = "published"
	if err := ps.Update(p, p.ID); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, err := ps.GetPublishedBySlug("old")
	if err != nil {
		t.Fatalf("GetPublishedBySlug after update: %v", err)
	}
	if got.Title != "New" {
		t.Errorf("Title = %q, want New", got.Title)
	}

	if err := ps.Delete(got.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	all, _ := ps.ListAll()
	if len(all) != 0 {
		t.Errorf("after delete, ListAll = %d posts, want 0", len(all))
	}
}

func TestPageStoreCreateAndGet(t *testing.T) {
	pgs := PageStore{DB: newTestDB(t)}

	// Slug must be supplied by the caller (the handler derives it via Slugify).
	in := &model.Page{Title: "About Us", Slug: "about-us"}
	if err := pgs.Create(in); err != nil {
		t.Fatalf("Create: %v", err)
	}

	p, err := pgs.GetBySlug("about-us")
	if err != nil {
		t.Fatalf("GetBySlug: %v", err)
	}
	if p.Title != "About Us" {
		t.Errorf("Title = %q, want About Us", p.Title)
	}

	pages, err := pgs.ListAll()
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(pages) != 1 {
		t.Errorf("ListAll = %d pages, want 1", len(pages))
	}

	if _, err := pgs.GetBySlug("ghost"); err == nil {
		t.Error("missing page should return an error")
	}
}
