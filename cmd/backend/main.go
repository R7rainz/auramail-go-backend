package main

import (
	"context"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/r7rainz/auramail/internal/server"

	authgoogle "github.com/r7rainz/auramail/internal/auth/google"
)

func main() {
	ctx := context.Background()

	_ = godotenv.Load()

	db_url := os.Getenv("DATABASE_URL")
	dsn := os.Getenv("GOOSE_DBSTRING")
	if dsn == "" {
		dsn = db_url
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
	googleHandler := authgoogle.NewHandler(googleCfg)

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	mux.HandleFunc("/auth/google", googleHandler.GoogleAuth)

	srv := server.New(":8080", mux)

	go func() {
		if err := http.ListenAndServe(":8080", mux); err != nil {
			panic(err)
		}
	}()

	//wait for shutdown signal
	<-ctx.Done()
	srv.Shutdown(context.Background())
}
