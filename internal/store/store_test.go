package store

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/espcaa/sablier/internal/event"
)

func open(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "sablier.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func mk(t *testing.T, ts time.Time) event.Event {
	t.Helper()
	e, err := event.New(event.PromptSent, ts, event.PromptSentData{Hash: "h"})
	if err != nil {
		t.Fatal(err)
	}
	e.SessionID, e.ProjectID, e.Harness, e.Source = "s1", "p1", "claude-code", event.SourceAdapter
	return e
}

func TestAppendAndBetween(t *testing.T) {
	s, ctx := open(t), context.Background()
	t0 := time.Date(2026, 11, 19, 17, 0, 0, 0, time.UTC)

	for i := range 3 {
		e, err := s.Append(ctx, mk(t, t0.Add(time.Duration(i)*time.Minute)))
		if err != nil {
			t.Fatal(err)
		}
		if e.Seq != int64(i+1) {
			t.Fatalf("seq = %d, want %d", e.Seq, i+1)
		}
	}

	got, err := s.Between(ctx, t0, t0.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d events, want 2", len(got))
	}
	p, err := got[0].Payload()
	if err != nil {
		t.Fatal(err)
	}
	if p.(*event.PromptSentData).Hash != "h" || !got[0].TS.Equal(t0) {
		t.Fatalf("bad round trip: %+v", got[0])
	}
}

func TestPendingAndMarkUploaded(t *testing.T) {
	s, ctx := open(t), context.Background()
	now := time.Now()
	for range 3 {
		if _, err := s.Append(ctx, mk(t, now)); err != nil {
			t.Fatal(err)
		}
	}

	if err := s.MarkUploaded(ctx, 2, now); err != nil {
		t.Fatal(err)
	}
	pending, err := s.Pending(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 1 || pending[0].Seq != 3 {
		t.Fatalf("pending = %+v, want only seq 3", pending)
	}
}

// separate connections, like separate hook processes
func TestConcurrentWriters(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sablier.db")
	ctx := context.Background()

	var wg sync.WaitGroup
	for range 5 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s, err := Open(path)
			if err != nil {
				t.Error(err)
				return
			}
			defer s.Close()
			for range 20 {
				if _, err := s.Append(ctx, mk(t, time.Now())); err != nil {
					t.Error(err)
					return
				}
			}
		}()
	}
	wg.Wait()

	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	all, err := s.Pending(ctx, 1000)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 100 {
		t.Fatalf("got %d events, want 100", len(all))
	}
}

func TestSessionState(t *testing.T) {
	s, ctx := open(t), context.Background()
	now := time.Now()

	err := s.Update(ctx, func(tx *Tx) error {
		got, err := tx.SessionState(ctx, "s1")
		if err != nil || got != nil {
			t.Fatalf("new session: got %q, %v", got, err)
		}
		return tx.PutSessionState(ctx, "s1", []byte(`{"open_turn":"p1"}`), now)
	})
	if err != nil {
		t.Fatal(err)
	}

	err = s.Update(ctx, func(tx *Tx) error {
		got, err := tx.SessionState(ctx, "s1")
		if string(got) != `{"open_turn":"p1"}` {
			t.Fatalf("got %q, %v", got, err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestUpdateRollsBack(t *testing.T) {
	s, ctx := open(t), context.Background()
	boom := errors.New("boom")

	err := s.Update(ctx, func(tx *Tx) error {
		if _, err := tx.Append(ctx, mk(t, time.Now())); err != nil {
			t.Fatal(err)
		}
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("got %v", err)
	}
	if all, _ := s.Pending(ctx, 10); len(all) != 0 {
		t.Fatalf("event kept after rollback: %+v", all)
	}
}

func TestSaltIsStable(t *testing.T) {
	s, ctx := open(t), context.Background()
	a, err := s.Salt(ctx)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := s.Salt(ctx)
	if len(a) != 32 || string(a) != string(b) {
		t.Fatalf("salt changed or wrong size: %x %x", a, b)
	}
}

func TestProjects(t *testing.T) {
	s, ctx := open(t), context.Background()
	err := s.Update(ctx, func(tx *Tx) error {
		if err := tx.PutProject(ctx, "id1", "/a"); err != nil {
			return err
		}
		return tx.PutProject(ctx, "id1", "/b") // first root wins
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.Projects(ctx)
	if err != nil || got["id1"] != "/a" || len(got) != 1 {
		t.Fatalf("got %v, %v", got, err)
	}
}

func TestWAL(t *testing.T) {
	s := open(t)
	var mode string
	if err := s.db.QueryRow("PRAGMA journal_mode").Scan(&mode); err != nil || mode != "wal" {
		t.Fatalf("journal_mode = %q, %v", mode, err)
	}
}
