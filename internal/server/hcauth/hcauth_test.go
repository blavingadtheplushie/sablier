package hcauth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// fake auth.hackclub.com
func fakeAuth(t *testing.T, me string) *Client {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /oauth/token", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		if body["code"] != "good-code" || body["client_secret"] != "secret" || body["grant_type"] != "authorization_code" {
			http.Error(w, `{"error":"invalid_grant"}`, http.StatusBadRequest)
			return
		}
		w.Write([]byte(`{"access_token":"idntk.abc","token_type":"Bearer"}`))
	})
	mux.HandleFunc("GET /api/v1/me", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer idntk.abc" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Write([]byte(me))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	c := New("client", "secret", "http://localhost:8080/auth/callback")
	c.BaseURL = srv.URL
	return c
}

func TestAuthorizeURL(t *testing.T) {
	c := New("client", "secret", "http://localhost:8080/auth/callback")
	u, err := url.Parse(c.AuthorizeURL("st4te"))
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	if u.Host != "auth.hackclub.com" || q.Get("client_id") != "client" || q.Get("state") != "st4te" ||
		q.Get("scope") != "openid profile name slack_id" || q.Get("response_type") != "code" {
		t.Fatalf("bad url: %s", u)
	}
}

func TestSignIn(t *testing.T) {
	c := fakeAuth(t, `{"identity":{"id":"ident!a1","first_name":"Alice","last_name":"B","slack_id":"U123"},"scopes":["name","slack_id"]}`)
	ctx := context.Background()

	token, err := c.Exchange(ctx, "good-code")
	if err != nil {
		t.Fatal(err)
	}
	id, err := c.Me(ctx, token)
	if err != nil || id.ID != "ident!a1" || id.SlackID != "U123" || id.Name() != "Alice B" {
		t.Fatalf("got %+v, %v", id, err)
	}
}

func TestNoSlack(t *testing.T) {
	c := fakeAuth(t, `{"identity":{"id":"ident!a1","slack_id":null},"scopes":[]}`)
	id, err := c.Me(context.Background(), "idntk.abc")
	if err != nil || id.SlackID != "" || id.Name() != "ident!a1" {
		t.Fatalf("got %+v, %v", id, err)
	}
}

func TestBadCode(t *testing.T) {
	c := fakeAuth(t, `{}`)
	_, err := c.Exchange(context.Background(), "bad-code")
	if err == nil || !strings.Contains(err.Error(), "400") {
		t.Fatalf("got %v", err)
	}
}

func TestMeWithoutID(t *testing.T) {
	c := fakeAuth(t, `{"identity":{},"scopes":[]}`)
	if _, err := c.Me(context.Background(), "idntk.abc"); err == nil {
		t.Fatal("expected error")
	}
}
