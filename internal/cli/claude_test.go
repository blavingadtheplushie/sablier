package cli

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/espcaa/sablier/internal/event"
	"github.com/espcaa/sablier/internal/launcher"
	"github.com/espcaa/sablier/internal/store"
)

func TestTickRecorder(t *testing.T) {
	home := t.TempDir()
	t.Setenv("SABLIER_HOME", home)
	ctx := context.Background()
	now := time.Now()

	var warn bytes.Buffer
	onTick, closeRec := tickRecorder(ctx, "r1", &warn)
	onTick(launcher.Tick{At: now, Window: 10 * time.Second, Keys: 7, Pasted: true})
	closeRec()
	if warn.Len() != 0 {
		t.Fatalf("unexpected warning: %s", warn.String())
	}

	s, err := store.Open(filepath.Join(home, "sablier.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	events, err := s.Between(ctx, now.Add(-time.Minute), now.Add(time.Minute))
	if err != nil || len(events) != 1 {
		t.Fatalf("got %d events, %v", len(events), err)
	}
	e := events[0]
	if e.Kind != event.InputTick || e.SessionID != "run:r1" || e.Source != event.SourceLauncher || e.ProjectID == "" {
		t.Fatalf("bad envelope: %+v", e)
	}
	p, _ := e.Payload()
	if d := p.(*event.InputTickData); d.Keys != 7 || !d.Pasted || d.WindowMs != 10000 {
		t.Fatalf("bad payload: %+v", d)
	}
	names, _ := s.Projects(ctx)
	if names[e.ProjectID] == "" {
		t.Fatal("project name not recorded")
	}
}
