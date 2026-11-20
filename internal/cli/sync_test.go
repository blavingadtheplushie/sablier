package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/espcaa/sablier/internal/api"
	"github.com/espcaa/sablier/internal/event"
	"github.com/espcaa/sablier/internal/store"
)

type fakeServer struct {
	mu      sync.Mutex
	batches []api.Batch
}

func (f *fakeServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Authorization") != "Bearer sab_good" {
		w.WriteHeader(401)
		return
	}
	var b api.Batch
	json.NewDecoder(r.Body).Decode(&b)
	f.mu.Lock()
	f.batches = append(f.batches, b)
	f.mu.Unlock()
	json.NewEncoder(w).Encode(api.Result{Accepted: len(b.Events)})
}

func TestSync(t *testing.T) {
	ctx := context.Background()
	s, err := store.Open(filepath.Join(t.TempDir(), "sablier.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	n := api.MaxBatch + 3 // two batches
	for i := range n {
		e, _ := event.New(event.InputTick, time.Now().Add(time.Duration(i)*time.Second), event.InputTickData{Keys: 1})
		e.ProjectID = "p1"
		if _, err := s.Append(ctx, e); err != nil {
			t.Fatal(err)
		}
	}
	s.Update(ctx, func(tx *store.Tx) error { return tx.PutProject(ctx, "p1", "/Users/alice/projects/sablier") })

	fake := &fakeServer{}
	srv := httptest.NewServer(fake)
	defer srv.Close()
	client := api.New(srv.URL, "sab_good")

	var out bytes.Buffer
	if err := syncEvents(ctx, s, client, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "synced 503 events") {
		t.Fatalf("got %q", out.String())
	}
	if len(fake.batches) != 2 || len(fake.batches[0].Events) != api.MaxBatch || len(fake.batches[1].Events) != 3 {
		t.Fatalf("batches: %d", len(fake.batches))
	}
	b := fake.batches[0]
	if len(b.InstallID) != 36 || len(b.Projects) != 1 || b.Projects[0].Name != "sablier" {
		t.Fatalf("first batch: install %q, projects %+v", b.InstallID, b.Projects)
	}
	if fake.batches[1].InstallID != b.InstallID || len(fake.batches[1].Projects) != 0 {
		t.Fatal("second batch should keep the install and skip projects")
	}

	out.Reset()
	syncEvents(ctx, s, client, &out)
	if strings.TrimSpace(out.String()) != "nothing to sync" {
		t.Fatalf("second sync: %q", out.String())
	}
}

func TestSyncBadKey(t *testing.T) {
	ctx := context.Background()
	s, _ := store.Open(filepath.Join(t.TempDir(), "sablier.db"))
	defer s.Close()
	e, _ := event.New(event.InputTick, time.Now(), event.InputTickData{Keys: 1})
	s.Append(ctx, e)

	srv := httptest.NewServer(&fakeServer{})
	defer srv.Close()
	err := syncEvents(ctx, s, api.New(srv.URL, "sab_bad"), &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "login again") {
		t.Fatalf("got %v", err)
	}
	if p, _ := s.Pending(ctx, 10); len(p) != 1 {
		t.Fatal("event marked uploaded after a failed sync")
	}
}

func TestSyncNotSignedIn(t *testing.T) {
	t.Setenv("SABLIER_CONFIG", filepath.Join(t.TempDir(), "none.cfg"))
	t.Setenv("SABLIER_HOME", t.TempDir())
	root := NewRootCmd()
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"sync"})
	if err := root.Execute(); err == nil || !strings.Contains(err.Error(), "sablier login") {
		t.Fatalf("got %v", err)
	}
}
