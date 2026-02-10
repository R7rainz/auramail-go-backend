package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/r7rainz/auramail/internal/auth"
	authgoogle "github.com/r7rainz/auramail/internal/auth/google"
	"github.com/r7rainz/auramail/internal/calendar"
	"github.com/r7rainz/auramail/internal/config"
	"github.com/r7rainz/auramail/internal/gmail"
	"github.com/r7rainz/auramail/internal/middleware"
	"github.com/r7rainz/auramail/internal/server"
	"github.com/r7rainz/auramail/internal/user"
)

func corsMiddleware(allowedOrigins []string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && isAllowedOrigin(origin, allowedOrigins) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func isAllowedOrigin(origin string, allowed []string) bool {
	for _, a := range allowed {
		if origin == a {
			return true
		}
	}
	return false
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_ = godotenv.Load()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("configuration error", "err", err)
		os.Exit(1)
	}

	dsn := cfg.GooseDBString
	if dsn == "" {
		dsn = cfg.DatabaseURL
	}

	db, err := pgxpool.New(ctx, dsn)
	if err != nil {
		slog.Error("unable to connect to database", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Ping(ctx); err != nil {
		slog.Error("database ping failed", "err", err)
		os.Exit(1)
	}

	mux := http.NewServeMux()
	googleCfg := authgoogle.NewOAuthConfig()

	userRepo := user.NewPostgresRepository(db)
	googleHandler := authgoogle.NewHandler(googleCfg, userRepo, cfg.FrontendURL)
	authHandler := auth.NewHandler(googleCfg, userRepo)
	gmailHandler := gmail.NewHandler(cfg, userRepo)
	calendarHandler := calendar.NewHandler(userRepo)

	slog.Info("google oauth configured", "redirect_url", googleCfg.RedirectURL)

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if err := db.Ping(r.Context()); err != nil {
			http.Error(w, "db not ready", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "healthy"}`))
	})

	// Auth routes
	mux.HandleFunc("/auth/google", googleHandler.GoogleAuth)
	mux.HandleFunc("/auth/google/callback", googleHandler.GoogleCallback)
	mux.HandleFunc("POST /auth/refresh", authHandler.Refresh)
	mux.Handle("GET /auth/me", auth.AuthMiddleware(http.HandlerFunc(authHandler.Me)))
	mux.Handle("POST /auth/logout", auth.AuthMiddleware(http.HandlerFunc(authHandler.Logout)))

	// Email routes (protected)
	mux.Handle("GET /emails", auth.AuthMiddleware(http.HandlerFunc(gmailHandler.GetEmails)))
	mux.Handle("GET /emails/sync", auth.AuthMiddleware(http.HandlerFunc(gmailHandler.SyncPlacementEmails)))
	mux.Handle("GET /emails/stream", auth.AuthMiddleware(http.HandlerFunc(gmailHandler.StreamPlacementEmails)))

	// Calendar routes (protected) - Add placement events to Google Calendar
	mux.Handle("GET /calendar/events", auth.AuthMiddleware(http.HandlerFunc(calendarHandler.GetEvents)))
	mux.Handle("POST /calendar/events", auth.AuthMiddleware(http.HandlerFunc(calendarHandler.AddEvent)))
	mux.Handle("DELETE /calendar/events", auth.AuthMiddleware(http.HandlerFunc(calendarHandler.DeleteEvent)))

	// Apply middleware chain
	handler := corsMiddleware(cfg.AllowedOrigins, middleware.RateLimitMiddleware(mux))

	// Create server using the server package
	srv := server.NewWithDefaults(":"+cfg.ServerPort, handler)

	go func() {
		slog.Info("server starting", "addr", srv.Addr())
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "err", err)
		}
	}()

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan
	slog.Info("shutdown signal received")

	// Graceful shutdown
	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelShutdown()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("server forced to shutdown", "err", err)
	}
	slog.Info("server gracefully stopped")
}
