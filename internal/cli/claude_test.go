package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/espcaa/sablier/internal/event"
	"github.com/espcaa/sablier/internal/launcher"
	"github.com/espcaa/sablier/internal/project"
	"github.com/espcaa/sablier/internal/store"
)

func TestRecorder(t *testing.T) {
	home := t.TempDir()
	t.Setenv("SABLIER_HOME", home)
	ctx := context.Background()
	now := time.Now()

	var warn bytes.Buffer
	rec := newRecorder(ctx, "r1", &warn)
	rec.tick(launcher.Tick{At: now, Window: 10 * time.Second, Keys: 7, Mouse: 3, Pasted: true})
	rec.focus(false, now.Add(time.Second))
	rec.close()
	if warn.Len() != 0 {
		t.Fatalf("unexpected warning: %s", warn.String())
	}

	s, err := store.Open(filepath.Join(home, "sablier.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	events, err := s.Between(ctx, now.Add(-time.Minute), now.Add(time.Minute))
	if err != nil || len(events) != 2 {
		t.Fatalf("got %d events, %v", len(events), err)
	}
	e := events[0]
	if e.Kind != event.InputTick || e.SessionID != "run:r1" || e.Source != event.SourceLauncher || e.ProjectID == "" {
		t.Fatalf("bad envelope: %+v", e)
	}
	p, _ := e.Payload()
	if d := p.(*event.InputTickData); d.Keys != 7 || d.Mouse != 3 || !d.Pasted || d.WindowMs != 10000 {
		t.Fatalf("bad payload: %+v", d)
	}
	f := events[1]
	if fp, _ := f.Payload(); f.Kind != event.Focus || fp.(*event.FocusData).In || f.SessionID != "run:r1" {
		t.Fatalf("bad focus event: %+v", f)
	}

	cwd, _ := os.Getwd()
	if e.ProjectID != project.Detect(cwd) {
		t.Fatalf("project %q, want the launch folder's", e.ProjectID)
	}
}
