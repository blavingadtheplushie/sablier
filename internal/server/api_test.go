package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/espcaa/sablier/internal/server/db/sqlc"
)

func call(h http.Handler, method, path, key string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestMe(t *testing.T) {
	pool := testPool(t)
	q := sqlc.New(pool)
	q.UpsertUser(context.Background(), sqlc.UpsertUserParams{ID: "ident!a1", Name: "Alice B", ApiKey: "sab_good"})
	h := New(Config{}, pool, q, nil)

	rec := call(h, "GET", "/v1/me", "sab_good")
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"name":"Alice B"`) {
		t.Fatalf("got %d %s", rec.Code, rec.Body)
	}
	if rec := call(h, "GET", "/v1/me", "sab_wrong"); rec.Code != 401 {
		t.Fatalf("wrong key: got %d", rec.Code)
	}
}

func TestMeWithoutKey(t *testing.T) {
	rec := call(New(Config{}, nil, nil, nil), "GET", "/v1/me", "")
	if rec.Code != 401 || !strings.Contains(rec.Body.String(), "missing api key") {
		t.Fatalf("got %d %s", rec.Code, rec.Body)
	}
}
