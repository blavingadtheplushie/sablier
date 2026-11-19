package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeDB struct{ err error }

func (f fakeDB) Ping(context.Context) error { return f.err }

func TestHealth(t *testing.T) {
	cases := map[string]struct {
		db   fakeDB
		code int
	}{
		"db up":   {fakeDB{}, http.StatusOK},
		"db down": {fakeDB{errors.New("down")}, http.StatusServiceUnavailable},
	}
	for name, c := range cases {
		rec := httptest.NewRecorder()
		New(Config{}, c.db, nil, nil).ServeHTTP(rec, httptest.NewRequest("GET", "/health", nil))
		if rec.Code != c.code {
			t.Errorf("%s: got %d, want %d", name, rec.Code, c.code)
		}
	}
}

func TestConfigNeedsDatabase(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	if _, err := ConfigFromEnv(); err == nil {
		t.Fatal("expected error")
	}
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("SABLIER_ADDR", "")
	c, err := ConfigFromEnv()
	if err != nil || c.Addr != ":8080" {
		t.Fatalf("got %+v, %v", c, err)
	}
}
