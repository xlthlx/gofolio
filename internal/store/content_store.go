package store

import (
	"database/sql"
	"time"

	"gofolio/internal/model"
)

type PostStore struct{ DB *sql.DB }

const postCols = `id, title, slug, content, status, created_at, updated_at`

func (s PostStore) ListPublished() ([]model.Post, error) {
	return s.list(`SELECT ` + postCols + ` FROM posts WHERE status = 'published' ORDER BY created_at DESC`)
}

func (s PostStore) ListAll() ([]model.Post, error) {
	return s.list(`SELECT ` + postCols + ` FROM posts ORDER BY created_at DESC`)
}

func (s PostStore) list(query string) ([]model.Post, error) {
	rows, err := s.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []model.Post
	for rows.Next() {
		var p model.Post
		if err := rows.Scan(&p.ID, &p.Title, &p.Slug, &p.Content, &p.Status, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		posts = append(posts, p)
	}
	return posts, rows.Err()
}

// GetPublishedBySlug returns sql.ErrNoRows when not found; handlers map that to 404.
func (s PostStore) GetPublishedBySlug(slug string) (*model.Post, error) {
	var p model.Post
	err := s.DB.QueryRow(
		`SELECT `+postCols+` FROM posts WHERE slug = ? AND status = 'published'`, slug,
	).Scan(&p.ID, &p.Title, &p.Slug, &p.Content, &p.Status, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (s PostStore) Create(p *model.Post) error {
	p.Status = draftStatus(p.Status)
	result, err := s.DB.Exec(
		`INSERT INTO posts (title, slug, content, status) VALUES (?, ?, ?, ?)`,
		p.Title, p.Slug, p.Content, p.Status,
	)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	p.ID = id
	p.CreatedAt = time.Now()
	p.UpdatedAt = time.Now()
	return nil
}

func (s PostStore) Update(p *model.Post, id int64) error {
	p.Status = draftStatus(p.Status)
	_, err := s.DB.Exec(
		`UPDATE posts SET title = ?, slug = ?, content = ?, status = ?, updated_at = ? WHERE id = ?`,
		p.Title, p.Slug, p.Content, p.Status, time.Now(), id,
	)
	return err
}

func (s PostStore) Delete(id int64) error {
	_, err := s.DB.Exec(`DELETE FROM posts WHERE id = ?`, id)
	return err
}

type PageStore struct{ DB *sql.DB }

const pageCols = `id, title, slug, content, status, created_at, updated_at`

func (s PageStore) ListAll() ([]model.Page, error) {
	rows, err := s.DB.Query(`SELECT ` + pageCols + ` FROM pages ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pages []model.Page
	for rows.Next() {
		var p model.Page
		if err := rows.Scan(&p.ID, &p.Title, &p.Slug, &p.Content, &p.Status, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		pages = append(pages, p)
	}
	return pages, rows.Err()
}

// GetPublishedBySlug returns a published page and nothing else, so a draft stays
// invisible however its slug is guessed. This mirrors GetPublishedBySlug on posts
// and returns sql.ErrNoRows when no published page matches.
func (s PageStore) GetPublishedBySlug(slug string) (*model.Page, error) {
	var p model.Page
	err := s.DB.QueryRow(
		`SELECT `+pageCols+` FROM pages WHERE slug = ? AND status = 'published'`, slug,
	).Scan(&p.ID, &p.Title, &p.Slug, &p.Content, &p.Status, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (s PageStore) Create(p *model.Page) error {
	p.Status = draftStatus(p.Status)
	result, err := s.DB.Exec(
		`INSERT INTO pages (title, slug, content, status) VALUES (?, ?, ?, ?)`,
		p.Title, p.Slug, p.Content, p.Status,
	)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	p.ID = id
	p.CreatedAt = time.Now()
	p.UpdatedAt = time.Now()
	return nil
}

// GetByID returns the page with the given id, or sql.ErrNoRows.
func (s PageStore) GetByID(id int64) (*model.Page, error) {
	var p model.Page
	row := s.DB.QueryRow(`SELECT `+pageCols+` FROM pages WHERE id = ?`, id)
	err := row.Scan(&p.ID, &p.Title, &p.Slug, &p.Content, &p.Status, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// Update modifies title, slug, content and status of an existing page.
func (s PageStore) Update(p *model.Page, id int64) error {
	p.Status = draftStatus(p.Status)
	_, err := s.DB.Exec(
		`UPDATE pages SET title = ?, slug = ?, content = ?, status = ?, updated_at = ? WHERE id = ?`,
		p.Title, p.Slug, p.Content, p.Status, time.Now(), id)
	return err
}

// Delete removes the page with the given id.
func (s PageStore) Delete(id int64) error {
	_, err := s.DB.Exec(`DELETE FROM pages WHERE id = ?`, id)
	return err
}

func (s PostStore) GetByID(id int64) (*model.Post, error) {
	var p model.Post
	row := s.DB.QueryRow(
		`SELECT `+postCols+` FROM posts WHERE id = ?`, id)
	err := row.Scan(&p.ID, &p.Title, &p.Slug, &p.Content, &p.Status, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}
