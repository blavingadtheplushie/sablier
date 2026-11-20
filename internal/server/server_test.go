package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5"
)

type fakeDB struct{ err error }

func (f fakeDB) Ping(context.Context) error            { return f.err }
func (f fakeDB) Begin(context.Context) (pgx.Tx, error) { return nil, f.err }

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
