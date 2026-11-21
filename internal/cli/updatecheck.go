package cli

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/espcaa/sablier/internal/store"
	"github.com/espcaa/sablier/internal/update"
	"github.com/espcaa/sablier/internal/version"
)

const checkEvery = 24 * time.Hour

// asks github for the latest release at most once a day, in the background, returns the line to print
func startUpdateCheck(s *store.Store, now time.Time) func() string {
	current := version.Version
	if s == nil || current == "dev" {
		return func() string { return "" }
	}
	notice := func(latest string) string {
		if !update.Newer(latest, current) {
			return ""
		}
		return fmt.Sprintf("sablier %s is out (you have %s), run sablier update", latest, current)
	}

	ctx := context.Background()
	checked, _ := s.Meta(ctx, "update_checked_at")
	if ms, err := strconv.ParseInt(checked, 10, 64); err == nil && now.Sub(time.UnixMilli(ms)) < checkEvery {
		latest, _ := s.Meta(ctx, "update_latest")
		return func() string { return notice(latest) }
	}

	found := make(chan string, 1)
	go func() {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		latest, err := update.Latest(ctx, http.DefaultClient)
		if err != nil {
			found <- ""
			return // offline or no release, try again next time
		}
		s.SetMeta(ctx, "update_latest", latest)
		s.SetMeta(ctx, "update_checked_at", strconv.FormatInt(now.UnixMilli(), 10))
		found <- latest
	}()
	return func() string {
		select {
		case latest := <-found:
			return notice(latest)
		default:
			return "" // still checking, it'll show next time
		}
	}
}
