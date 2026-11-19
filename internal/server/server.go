package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/espcaa/sablier/internal/server/db"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pinger interface {
	Ping(ctx context.Context) error
}

func Handler(pool pinger) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.RealIP, middleware.Logger, middleware.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := pool.Ping(ctx); err != nil {
			http.Error(w, "db unreachable", http.StatusServiceUnavailable)
			return
		}
		w.Write([]byte("ok\n"))
	})

	return r
}

// serves until ctx is done, then shuts down gracefully
func Run(ctx context.Context, c Config) error {
	pool, err := pgxpool.New(ctx, c.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		return err
	}
	if err := db.Migrate(ctx, pool); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	srv := &http.Server{Addr: c.Addr, Handler: Handler(pool), ReadHeaderTimeout: 10 * time.Second}
	errc := make(chan error, 1)
	go func() { errc <- srv.ListenAndServe() }()
	log.Printf("sablier server on %s", c.Addr)

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdown); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	}
}
