package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/espcaa/sablier/internal/api"
	"github.com/espcaa/sablier/internal/event"
	"github.com/espcaa/sablier/internal/server/db/sqlc"
)

const installA = "11111111-1111-4111-8111-111111111111"

func post(h http.Handler, key string, body any) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/v1/events", bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+key)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func ev(seq int64) event.Event {
	e, _ := event.New(event.PromptSent, time.Date(2026, 11, 20, 10, 0, int(seq), 0, time.UTC), event.PromptSentData{Hash: "h"})
	e.Seq, e.SessionID, e.ProjectID, e.Harness, e.Source = seq, "run:x", "p1", "claude-code", event.SourceAdapter
	return e
}

func ingestSetup(t *testing.T) (http.Handler, *sqlc.Queries) {
	pool := testPool(t)
	q := sqlc.New(pool)
	ctx := context.Background()
	q.UpsertUser(ctx, sqlc.UpsertUserParams{ID: "ident!a", ApiKey: "sab_a"})
	q.UpsertUser(ctx, sqlc.UpsertUserParams{ID: "ident!b", ApiKey: "sab_b"})
	return New(Config{}, pool, q, nil), q
}

func TestIngest(t *testing.T) {
	h, q := ingestSetup(t)
	batch := api.Batch{
		InstallID: installA,
		Events:    []event.Event{ev(1), ev(2)},
		Projects:  []api.Project{{ID: "p1", Name: "sablier"}},
	}

	rec := post(h, "sab_a", batch)
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"accepted":2,"duplicates":0`) {
		t.Fatalf("got %d %s", rec.Code, rec.Body)
	}

	// a retry with one new event
	batch.Events = append(batch.Events, ev(3))
	rec = post(h, "sab_a", batch)
	if !strings.Contains(rec.Body.String(), `"accepted":1,"duplicates":2`) {
		t.Fatalf("retry: %s", rec.Body)
	}

	ps, _ := q.UserProjects(context.Background(), "ident!a")
	if len(ps) != 1 || ps[0].Name != "sablier" {
		t.Fatalf("projects %+v", ps)
	}
}

func TestIngestOtherAccountsInstall(t *testing.T) {
	h, _ := ingestSetup(t)
	post(h, "sab_a", api.Batch{InstallID: installA, Events: []event.Event{ev(1)}})

	rec := post(h, "sab_b", api.Batch{InstallID: installA, Events: []event.Event{ev(2)}})
	if rec.Code != http.StatusForbidden {
		t.Fatalf("got %d %s", rec.Code, rec.Body)
	}
}

func TestIngestRejects(t *testing.T) {
	h, _ := ingestSetup(t)
	unknown := ev(1)
	unknown.Kind = "made_up"
	noSeq := ev(1)
	noSeq.Seq = 0
	tooMany := make([]event.Event, api.MaxBatch+1)
	for i := range tooMany {
		tooMany[i] = ev(int64(i + 1))
	}

	cases := map[string]api.Batch{
		"bad install":  {InstallID: "nope", Events: []event.Event{ev(1)}},
		"unknown kind": {InstallID: installA, Events: []event.Event{unknown}},
		"no seq":       {InstallID: installA, Events: []event.Event{noSeq}},
		"too many":     {InstallID: installA, Events: tooMany},
		"path as name": {InstallID: installA, Events: []event.Event{ev(1)}, Projects: []api.Project{{ID: "p1", Name: "/Users/alice/sablier"}}},
	}
	for name, b := range cases {
		if rec := post(h, "sab_a", b); rec.Code != http.StatusBadRequest {
			t.Errorf("%s: got %d %s", name, rec.Code, rec.Body)
		}
	}

	// nothing from a rejected batch is stored, not even the install
	rec := post(h, "sab_b", api.Batch{InstallID: installA, Events: []event.Event{ev(1)}})
	if rec.Code != 200 {
		t.Fatalf("install was claimed by a rejected batch: %d %s", rec.Code, rec.Body)
	}
}

func TestValidProjectName(t *testing.T) {
	for name, want := range map[string]bool{
		"sablier": true, "my project": true, "web-app_2": true, "café": true, ".dotfiles": true,
		"/a/b/sablier": false, "a/b": false, `C:\code\sablier`: false, "": false, ".": false, "..": false,
		"tab\there": false, "new\nline": false, strings.Repeat("a", 201): false,
	} {
		if got := validProjectName(name); got != want {
			t.Errorf("validProjectName(%q) = %v, want %v", name, got, want)
		}
	}
}
