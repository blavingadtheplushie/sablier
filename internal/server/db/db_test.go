package db

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/espcaa/sablier/internal/server/db/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

func testDB(t *testing.T) *sqlc.Queries {
	t.Helper()
	url := os.Getenv("SABLIER_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("SABLIER_TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if _, err := pool.Exec(ctx, "DROP SCHEMA public CASCADE; CREATE SCHEMA public"); err != nil {
		t.Fatal(err)
	}
	if err := Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	return sqlc.New(pool)
}

func uuid(b byte) pgtype.UUID { return pgtype.UUID{Bytes: [16]byte{b}, Valid: true} }

func TestUsersAndInstalls(t *testing.T) {
	q, ctx := testDB(t), context.Background()

	u, err := q.UpsertUser(ctx, sqlc.UpsertUserParams{ID: "ident!a", Name: "alice", ApiKey: "key-1"})
	if err != nil {
		t.Fatal(err)
	}
	// signing in again updates the name but keeps the key
	u, err = q.UpsertUser(ctx, sqlc.UpsertUserParams{ID: "ident!a", Name: "alice b", ApiKey: "key-2"})
	if err != nil || u.Name != "alice b" || u.ApiKey != "key-1" {
		t.Fatalf("got %+v, %v", u, err)
	}
	if got, err := q.UserByAPIKey(ctx, "key-1"); err != nil || got.ID != "ident!a" {
		t.Fatalf("got %+v, %v", got, err)
	}

	if _, err := q.ClaimInstall(ctx, sqlc.ClaimInstallParams{ID: uuid(1), UserID: "ident!a"}); err != nil {
		t.Fatal(err)
	}
	if _, err := q.ClaimInstall(ctx, sqlc.ClaimInstallParams{ID: uuid(1), UserID: "ident!a"}); err != nil {
		t.Fatalf("reclaiming own install: %v", err)
	}

	q.UpsertUser(ctx, sqlc.UpsertUserParams{ID: "ident!b", ApiKey: "key-b"})
	_, err = q.ClaimInstall(ctx, sqlc.ClaimInstallParams{ID: uuid(1), UserID: "ident!b"})
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("someone else's install was claimed: %v", err)
	}
}

func TestEventsAndProjects(t *testing.T) {
	q, ctx := testDB(t), context.Background()
	q.UpsertUser(ctx, sqlc.UpsertUserParams{ID: "ident!a", ApiKey: "k"})
	q.ClaimInstall(ctx, sqlc.ClaimInstallParams{ID: uuid(1), UserID: "ident!a"})

	ts := time.Date(2026, 11, 20, 10, 0, 0, 0, time.UTC)
	ev := sqlc.InsertEventParams{
		InstallID: uuid(1), Seq: 1, Ts: pgtype.Timestamptz{Time: ts, Valid: true},
		Kind: "prompt_sent", SessionID: "run:x", ProjectID: "p1", Harness: "claude-code", Source: "adapter",
		Data: []byte(`{"hash":"h"}`),
	}
	if n, err := q.InsertEvent(ctx, ev); err != nil || n != 1 {
		t.Fatalf("insert: %d, %v", n, err)
	}
	if n, err := q.InsertEvent(ctx, ev); err != nil || n != 0 {
		t.Fatalf("retry should be skipped: %d, %v", n, err)
	}

	got, err := q.UserEventsBetween(ctx, sqlc.UserEventsBetweenParams{
		UserID: "ident!a",
		FromTs: pgtype.Timestamptz{Time: ts.Add(-time.Hour), Valid: true},
		ToTs:   pgtype.Timestamptz{Time: ts.Add(time.Hour), Valid: true},
	})
	if err != nil || len(got) != 1 || got[0].Kind != "prompt_sent" {
		t.Fatalf("got %+v, %v", got, err)
	}

	q.UpsertProject(ctx, sqlc.UpsertProjectParams{InstallID: uuid(1), ProjectID: "p1", Name: "sablier"})
	q.UpsertProject(ctx, sqlc.UpsertProjectParams{InstallID: uuid(1), ProjectID: "p1", Name: "sablier2"})
	ps, err := q.UserProjects(ctx, "ident!a")
	if err != nil || len(ps) != 1 || ps[0].Name != "sablier2" {
		t.Fatalf("got %+v, %v", ps, err)
	}
}
