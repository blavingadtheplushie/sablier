package store

import (
	"context"
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

// hooks run as separate processes; separate connections is the closest test.
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
