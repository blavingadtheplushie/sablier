package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/espcaa/sablier/internal/server/db"
	"github.com/espcaa/sablier/internal/server/db/sqlc"
	"github.com/espcaa/sablier/internal/server/hcauth"
	"github.com/espcaa/sablier/internal/server/views"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pinger interface {
	Ping(ctx context.Context) error
}

type Server struct {
	cfg  Config
	pool pinger
	q    *sqlc.Queries
	auth *hcauth.Client
}

func New(cfg Config, pool pinger, q *sqlc.Queries, auth *hcauth.Client) http.Handler {
	s := &Server{cfg: cfg, pool: pool, q: q, auth: auth}

	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.RealIP, middleware.Logger, middleware.Recoverer)

	r.Get("/health", s.health)
	r.Get("/", s.home)
	r.Get("/auth/login", s.login)
	r.Get("/auth/callback", s.callback)
	r.Post("/auth/logout", s.logout)

	r.Route("/v1", func(r chi.Router) {
		r.Use(s.requireKey)
		r.Get("/me", s.me)
	})
	return r
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := s.pool.Ping(ctx); err != nil {
		http.Error(w, "db unreachable", http.StatusServiceUnavailable)
		return
	}
	w.Write([]byte("ok\n"))
}

func (s *Server) home(w http.ResponseWriter, r *http.Request) {
	var me *views.Me
	if u := s.currentUser(r); u != nil {
		me = &views.Me{Name: u.Name, APIKey: u.ApiKey}
	}
	views.Home(me).Render(r.Context(), w)
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

	var auth *hcauth.Client
	if c.HackClubClientID != "" {
		auth = hcauth.New(c.HackClubClientID, c.HackClubClientSecret, c.BaseURL+"/auth/callback")
	} else {
		log.Print("HACKCLUB_CLIENT_ID not set, sign in is off")
	}

	srv := &http.Server{Addr: c.Addr, Handler: New(c, pool, sqlc.New(pool), auth), ReadHeaderTimeout: 10 * time.Second}
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
