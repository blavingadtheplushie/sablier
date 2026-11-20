package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/espcaa/sablier/internal/event"
	"github.com/espcaa/sablier/internal/server/db/sqlc"
	"github.com/jackc/pgx/v5/pgtype"
)

type seed struct {
	q   *sqlc.Queries
	seq map[string]int64
}

func (sd *seed) add(t *testing.T, install string, kind event.Kind, at time.Time, project string, data any) {
	t.Helper()
	var id pgtype.UUID
	id.Scan(install)
	sd.q.ClaimInstall(context.Background(), sqlc.ClaimInstallParams{ID: id, UserID: "ident!a"})
	b, _ := json.Marshal(data)
	sd.seq[install]++
	_, err := sd.q.InsertEvent(context.Background(), sqlc.InsertEventParams{
		InstallID: id, Seq: sd.seq[install], Ts: pgtype.Timestamptz{Time: at, Valid: true},
		Kind: string(kind), SessionID: "run:" + install, ProjectID: project,
		Harness: "claude-code", Source: "adapter", Data: b,
	})
	if err != nil {
		t.Fatal(err)
	}
}

const installB = "22222222-2222-4222-8222-222222222222"

func TestToday(t *testing.T) {
	pool := testPool(t)
	q := sqlc.New(pool)
	ctx := context.Background()
	q.UpsertUser(ctx, sqlc.UpsertUserParams{ID: "ident!a", ApiKey: "sab_a"})
	s := &Server{q: q}
	sd := &seed{q: q, seq: map[string]int64{}}
	noon := time.Date(2026, 11, 20, 12, 0, 0, 0, time.Local)

	// laptop: 6m on sablier (1m reading back to session start + 5m after the prompt)
	sd.add(t, installA, event.SessionStart, noon, "pa1", event.SessionStartData{})
	sd.add(t, installA, event.PromptSent, noon.Add(time.Minute), "pa1", event.PromptSentData{})
	// desktop: the same folder, different id, 5m later in the day
	sd.add(t, installB, event.PromptSent, noon.Add(time.Hour), "pb1", event.PromptSentData{})
	// yesterday doesn't count
	sd.add(t, installB, event.PromptSent, noon.Add(-24*time.Hour), "pb1", event.PromptSentData{})

	q.UpsertProject(ctx, sqlc.UpsertProjectParams{InstallID: uuidOf(installA), ProjectID: "pa1", Name: "sablier"})
	q.UpsertProject(ctx, sqlc.UpsertProjectParams{InstallID: uuidOf(installB), ProjectID: "pb1", Name: "sablier"})

	total, projects, err := s.today(ctx, "ident!a", noon.Add(3*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if total != 11*time.Minute || len(projects) != 1 || projects[0].Name != "sablier" || projects[0].Time != 11*time.Minute {
		t.Fatalf("got %s, %+v", total, projects)
	}
}

func TestHomeSignedIn(t *testing.T) {
	pool := testPool(t)
	q := sqlc.New(pool)
	ctx := context.Background()
	q.UpsertUser(ctx, sqlc.UpsertUserParams{ID: "ident!a", Name: "Alice B", ApiKey: "sab_a"})
	q.CreateSession(ctx, sqlc.CreateSessionParams{
		TokenHash: hash("tok"), UserID: "ident!a",
		ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(time.Hour), Valid: true},
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: "tok"})
	rec := httptest.NewRecorder()
	New(Config{}, pool, q, nil).ServeHTTP(rec, req)

	body := rec.Body.String()
	for _, want := range []string{"signed in as Alice B", "today: 0m", "nothing tracked today yet", "sab_a"} {
		if !strings.Contains(body, want) {
			t.Errorf("page is missing %q:\n%s", want, body)
		}
	}
}

func uuidOf(s string) pgtype.UUID {
	var id pgtype.UUID
	id.Scan(s)
	return id
}
