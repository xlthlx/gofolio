package main

import (
	"context"
	"errors"
	"fmt"
	web "gofolio/internal/handler/http"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gofolio/internal/core"
	"gofolio/internal/handler/middleware"
	"gofolio/internal/service"
	"gofolio/internal/store"
)

func main() {
	seedOnly := len(os.Args) > 1 && os.Args[1] == "-seed-only"

	cfg := LoadConfig()

	db, err := store.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	if err := store.Migrate(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	if err := service.InitUsersTable(db); err != nil {
		log.Fatalf("Failed to init users table: %v", err)
	}
	if err := service.CreateDefaultAdmin(db); err != nil {
		log.Printf("Warning: could not create default admin: %v", err)
	}

	if seedOnly {
		fmt.Println("Database initialized and default admin ensured.")
		return
	}

	authSecret := []byte(cfg.Secret)
	if len(authSecret) == 0 {
		secret, err := middleware.GenerateSecureToken()
		if err != nil {
			log.Fatalf("Failed to generate auth secret: %v", err)
		}
		authSecret = []byte(secret)
	}

	tpl := web.MustRenderer(web.FindAdminThemeDir())
	sess := middleware.NewSessions(authSecret)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      core.New(db, tpl, sess),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	fmt.Printf("gofolio running at http://%s:%s\n", cfg.Domain, cfg.Port)
	fmt.Println("Default admin: username=admin, password=admin123 (change it after first login)")

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("ListenAndServe: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	fmt.Println("\nShutting down server...")
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown: %v", err)
	}
	fmt.Println("Server stopped")
}
