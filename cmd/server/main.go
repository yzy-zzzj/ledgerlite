package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yzy-zzzj/ledgerlite/internal/http"
	"github.com/yzy-zzzj/ledgerlite/internal/ledger"
	"github.com/yzy-zzzj/ledgerlite/internal/observability"
)

func main() {
	addr := envOr("ADDR", ":8080")
	dsn := envOr("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/ledgerlite?sslmode=disable")

	ctx := context.Background()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("pgxpool.New: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("db ping: %v", err)
	}

	logger := observability.New()

	store := ledger.NewPostgresStore(pool)
	svc := ledger.NewService(store, logger)

	mux := httpserver.Router(svc, logger)

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	logger.Infof("listening on %s", addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Errorf("server error: %v", err)
	}
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// Basic JSON helper
func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
