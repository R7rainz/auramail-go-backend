package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/r7rainz/auramail/internal/auth"
	"github.com/r7rainz/auramail/internal/gmail"
	"github.com/r7rainz/auramail/internal/server"
	"github.com/r7rainz/auramail/internal/user"

	authgoogle "github.com/r7rainz/auramail/internal/auth/google"
)

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == "OPTION" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_ = godotenv.Load()

	//Database connection
	dbURL := os.Getenv("DATABASE_URL")
	dsn := os.Getenv("GOOSE_DBSTRING")
	if dsn == "" {
		dsn = dbURL
	}
	db, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("Unable to connect to database : %v", err)
	}

	defer db.Close()

	if error := db.Ping(ctx); error != nil {
		panic(error)
	}

	//dependency injections
	mux := http.NewServeMux()
	googleCfg := authgoogle.NewOAuthConfig()

	userRepo := user.NewPostgresRepository(db)
	googleHandler := authgoogle.NewHandler(googleCfg, userRepo)
	authHandler := auth.NewHandler(googleCfg, userRepo)

	gmailHandler := gmail.NewHandler(userRepo)

	log.Printf("Google OAuth RedirectURL: %s", googleCfg.RedirectURL)

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	mux.HandleFunc("/auth/google", googleHandler.GoogleAuth)
	mux.HandleFunc("/auth/google/callback", googleHandler.GoogleCallback)
	mux.HandleFunc("POST /auth/refresh", authHandler.Refresh)
	mux.Handle("POST /auth/logout", auth.AuthMiddleware(http.HandlerFunc(authHandler.Logout)))
	mux.Handle("GET /emails/sync", auth.AuthMiddleware(http.HandlerFunc(gmailHandler.SyncPlacementEmails)))
	mux.Handle("GET /emails/stream", auth.AuthMiddleware((http.HandlerFunc(gmailHandler.StreamPlacementEmails))))

	handlerWithCORS := corsMiddleware(mux)
	srv := server.New(":8080", handlerWithCORS)

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

	//timeout for shutdown process
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10 * time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err  != nil {
		log.Fatalf("GraceFul shutdown failed: %v", err)
	}

	log.Println("Server exited cleanly")
}
