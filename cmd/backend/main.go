package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/r7rainz/auramail/internal/auth"
	"github.com/r7rainz/auramail/internal/server"
	"github.com/r7rainz/auramail/internal/user"

	authgoogle "github.com/r7rainz/auramail/internal/auth/google"
)

func main() {
	ctx := context.Background()

	_ = godotenv.Load()

	dbURL := os.Getenv("DATABASE_URL")
	dsn := os.Getenv("GOOSE_DBSTRING")
	if dsn == "" {
		dsn = dbURL
	}
	db, err := pgxpool.New(ctx, dsn)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	if error := db.Ping(ctx); error != nil {
		panic(error)
	}

	mux := http.NewServeMux()
	googleCfg := authgoogle.NewOAuthConfig()

	userRepo := user.NewPostgresRepository(db)
	googleHandler := authgoogle.NewHandler(googleCfg, userRepo)
	authHandler := auth.NewHandler(googleCfg, userRepo)

	log.Printf("Google OAuth RedirectURL: %s", googleCfg.RedirectURL)

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	mux.HandleFunc("/auth/google", googleHandler.GoogleAuth)
	mux.HandleFunc("/auth/google/callback", googleHandler.GoogleCallback)
	mux.HandleFunc("POST /auth/refresh", authHandler.Refresh)
	mux.HandleFunc("POST /auth/logout", authHandler.Logout)

	srv := server.New(":8080", mux)

	go func() {
		log.Printf("Server starting on :8080")
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutdown signal received, closing gracefully...")
	srv.Shutdown(context.Background())
}
