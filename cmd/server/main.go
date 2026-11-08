package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"gopress/internal/auth"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

type Post struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Slug      string    `json:"slug"`
	Content   string    `json:"content"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Page struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Slug      string    `json:"slug"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

func initDB() error {
	var err error
	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		dbPath = "./gopress.db"
	}

	db, err = sql.Open("sqlite3", dbPath+"?_foreign_keys=on")
	if err != nil {
		return err
	}

	db.SetMaxOpenConns(1)

	migrations := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		email TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		is_admin INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

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

	_, err = db.Exec(migrations)
	if err != nil {
		return err
	}

	// Initialize auth users table
	if err := auth.InitUsersTable(db); err != nil {
		return err
	}

	// Create default admin if none exists
	if err := auth.CreateDefaultAdmin(db); err != nil {
		log.Printf("Warning: Could not create default admin: %v", err)
	}

	return nil
}

func main() {
	if err := initDB(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize auth with a secure key (use env var in production)
	authSecret := []byte(os.Getenv("AUTH_SECRET"))
	if len(authSecret) == 0 {
		secret, err := auth.GenerateSecureToken()
		if err != nil {
			log.Fatalf("Failed to generate auth secret: %v", err)
		}
		authSecret = []byte(secret)
	}
	auth.Init(authSecret)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)

	// Trailing slash enforcement
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet && !strings.HasSuffix(r.URL.Path, "/") {
				redirectURL := r.URL.Path + "/"
				if r.URL.RawQuery != "" {
					redirectURL += "?" + r.URL.RawQuery
				}
				http.Redirect(w, r, redirectURL, http.StatusMovedPermanently)
				return
			}
			next.ServeHTTP(w, r)
		})
	})

	// Public routes
	r.Get("/", listPostsHandler)
	r.Get("/posts/{slug}/", getPostHandler)
	r.Get("/pages/{slug}/", getPageHandler)
	r.Get("/static/*", serveStatic)

	// Admin routes with authentication
	r.Route("/admin/", func(r chi.Router) {
		r.Use(auth.RequireAuth)

		r.Get("/", adminDashboard)
		r.Get("/dashboard/", adminDashboard)

		// Posts API
		r.Get("/api/posts/", adminListPosts)
		r.Post("/api/posts/", createPost)
		r.Put("/api/posts/{id}/", updatePost)
		r.Delete("/api/posts/{id}/", deletePost)

		// Pages API
		r.Get("/api/pages/", adminListPages)
		r.Post("/api/pages/", createPage)

		// Admin UI
		r.Get("/posts/", adminPostsView)
		r.Get("/posts/new/", newPostEditor)
		r.Get("/posts/{id}/edit/", editPostEditor)
		r.Get("/pages/", adminPagesView)
		r.Get("/settings/", adminSettings)
	})

	// Auth routes (no auth required)
	r.Post("/api/auth/login/", loginHandler)
	r.Get("/api/auth/logout/", logoutHandler)
	r.Get("/api/auth/check/", checkAuthHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	domain := os.Getenv("DOMAIN")
	if domain == "" {
		domain = "localhost"
	}

	addr := ":" + port
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	fmt.Printf("🚀 gopress Go running at http://%s:%s\n", domain, port)
	fmt.Println("ℹ️  Default admin: username=admin, password=admin123")
	fmt.Println("ℹ️  Change password after first login!")

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe(): %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	fmt.Println("\n🛑 Shutting down server...")
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown: %v", err)
	}
	fmt.Println("✅ Server stopped")
}

func serveStatic(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/static/")
	http.ServeFile(w, r, "frontend/"+path)
}

// === Auth Handlers ===

func loginHandler(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var creds struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.Unmarshal(body, &creds); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := auth.Login(db, w, r, creds.Username, creds.Password); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid credentials"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"success": "logged in"})
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	if err := auth.ClearSession(w, r); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"success": "logged out"})
}

func checkAuthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"authenticated": auth.IsAuthenticated(r)})
}

// === Post Handlers ===

func listPostsHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`SELECT id, title, slug, content, status, created_at, updated_at 
		FROM posts WHERE status = 'published' ORDER BY created_at DESC`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		var p Post
		rows.Scan(&p.ID, &p.Title, &p.Slug, &p.Content, &p.Status, &p.CreatedAt, &p.UpdatedAt)
		posts = append(posts, p)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"posts": posts})
}

func getPostHandler(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	row := db.QueryRow(`SELECT id, title, slug, content, status, created_at, updated_at 
		FROM posts WHERE slug = ? AND status = 'published'`, slug)

	var p Post
	err := row.Scan(&p.ID, &p.Title, &p.Slug, &p.Content, &p.Status, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(p)
}

func adminListPosts(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`SELECT id, title, slug, content, status, created_at, updated_at FROM posts ORDER BY created_at DESC`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		var p Post
		rows.Scan(&p.ID, &p.Title, &p.Slug, &p.Content, &p.Status, &p.CreatedAt, &p.UpdatedAt)
		posts = append(posts, p)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(posts)
}

func createPost(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var post Post
	if err := json.Unmarshal(body, &post); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if post.Title == "" {
		http.Error(w, "Title is required", http.StatusBadRequest)
		return
	}

	if post.Slug == "" {
		post.Slug = strings.ToLower(strings.ReplaceAll(post.Title, " ", "-"))
	}
	if post.Status == "" {
		post.Status = "draft"
	}

	result, err := db.Exec(`INSERT INTO posts (title, slug, content, status) VALUES (?, ?, ?, ?)`,
		post.Title, post.Slug, post.Content, post.Status)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	id, _ := result.LastInsertId()
	post.ID = id
	post.CreatedAt = time.Now()
	post.UpdatedAt = time.Now()

	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(post)
}

func updatePost(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	var id int
	fmt.Sscanf(idStr, "%d", &id)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var post Post
	if err := json.Unmarshal(body, &post); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_, err = db.Exec(`UPDATE posts SET title = ?, slug = ?, content = ?, status = ?, updated_at = ? WHERE id = ?`,
		post.Title, post.Slug, post.Content, post.Status, time.Now(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "updated", "id": fmt.Sprintf("%d", id)})
}

func deletePost(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	var id int
	fmt.Sscanf(idStr, "%d", &id)

	_, err := db.Exec("DELETE FROM posts WHERE id = ?", id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func adminPostsView(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "frontend/admin/posts.html")
}

func newPostEditor(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "frontend/admin/editor.html")
}

func editPostEditor(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "frontend/admin/editor.html")
}

func adminDashboard(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "frontend/admin/dashboard.html")
}

// === Page Handlers ===

func getPageHandler(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	row := db.QueryRow(`SELECT id, title, slug, content, created_at FROM pages WHERE slug = ?`, slug)

	var p Page
	err := row.Scan(&p.ID, &p.Title, &p.Slug, &p.Content, &p.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(p)
}

func adminListPages(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`SELECT id, title, slug, content, created_at FROM pages ORDER BY created_at DESC`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var pages []Page
	for rows.Next() {
		var p Page
		rows.Scan(&p.ID, &p.Title, &p.Slug, &p.Content, &p.CreatedAt)
		pages = append(pages, p)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pages)
}

func createPage(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var page Page
	if err := json.Unmarshal(body, &page); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if page.Title == "" {
		http.Error(w, "Title is required", http.StatusBadRequest)
		return
	}

	if page.Slug == "" {
		page.Slug = strings.ToLower(strings.ReplaceAll(page.Title, " ", "-"))
	}

	_, err = db.Exec(`INSERT INTO pages (title, slug, content) VALUES (?, ?, ?)`,
		page.Title, page.Slug, page.Content)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "created"})
}

func adminPagesView(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "frontend/admin/pages.html")
}

func adminSettings(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "frontend/admin/settings.html")
}
