package cli

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/espcaa/sablier/internal/config"
	"github.com/espcaa/sablier/internal/event"
	"github.com/espcaa/sablier/internal/store"
)

func autoSyncSetup(t *testing.T, apiURL, key string) *store.Store {
	t.Helper()
	t.Setenv("SABLIER_CONFIG", filepath.Join(t.TempDir(), "sablier.cfg"))
	if key != "" {
		config.Save(config.Config{APIKey: key, APIURL: apiURL})
	}
	s, err := store.Open(filepath.Join(t.TempDir(), "sablier.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func addTick(t *testing.T, s *store.Store) {
	t.Helper()
	e, _ := event.New(event.InputTick, time.Now(), event.InputTickData{Keys: 1})
	if _, err := s.Append(context.Background(), e); err != nil {
		t.Fatal(err)
	}
}

func pendingCount(s *store.Store) int {
	p, _ := s.Pending(context.Background(), 1000)
	return len(p)
}

func TestAutoSyncOffline(t *testing.T) {
	s := autoSyncSetup(t, "http://127.0.0.1:1", "sab_good")
	addTick(t, s)

	var warn bytes.Buffer
	startAutoSync(s, time.Hour, &warn)()
	if pendingCount(s) != 1 {
		t.Fatal("events lost while offline")
	}
	if !strings.Contains(warn.String(), "will retry next time") {
		t.Fatalf("got %q", warn.String())
	}
}
