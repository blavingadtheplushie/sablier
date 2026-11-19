package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func fakeServer(t *testing.T) *httptest.Server {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Header.Get("Authorization") != "Bearer sab_good":
			w.WriteHeader(401)
		case r.URL.Path == "/v1/me":
			w.Write([]byte(`{"id":"ident!a1","name":"Alice B"}`))
		default:
			w.WriteHeader(500)
			w.Write([]byte(`{"error":"boom"}`))
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestMe(t *testing.T) {
	srv := fakeServer(t)
	u, err := New(srv.URL+"/", "sab_good").Me(context.Background())
	if err != nil || u.Name != "Alice B" {
		t.Fatalf("got %+v, %v", u, err)
	}
	if _, err := New(srv.URL, "sab_bad").Me(context.Background()); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("got %v", err)
	}
}

func TestServerError(t *testing.T) {
	srv := fakeServer(t)
	err := New(srv.URL, "sab_good").do(context.Background(), "GET", "/other", nil, nil)
	if err == nil || err.Error() != "server: boom" {
		t.Fatalf("got %v", err)
	}
}

func TestUnreachable(t *testing.T) {
	if _, err := New("http://127.0.0.1:1", "k").Me(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}
