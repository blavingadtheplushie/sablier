package cli

import (
	"context"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/espcaa/sablier/internal/store"
	"github.com/espcaa/sablier/internal/version"
)

// a check from an hour ago is reused, so nothing goes to github
func TestUpdateNoticeFromCache(t *testing.T) {
	s, err := store.Open(filepath.Join(t.TempDir(), "sablier.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	now := time.Now()
	ctx := context.Background()
	s.SetMeta(ctx, "update_checked_at", strconv.FormatInt(now.Add(-time.Hour).UnixMilli(), 10))
	s.SetMeta(ctx, "update_latest", "v0.2.0")

	old := version.Version
	t.Cleanup(func() { version.Version = old })
	for v, want := range map[string]string{
		"v0.1.0": "sablier v0.2.0 is out (you have v0.1.0), run sablier update",
		"v0.2.0": "",
		"dev":    "",
	} {
		version.Version = v
		if got := startUpdateCheck(s, now)(); got != want {
			t.Errorf("on %s: got %q, want %q", v, got, want)
		}
	}
}
