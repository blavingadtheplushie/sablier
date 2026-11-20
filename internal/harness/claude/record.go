package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/espcaa/sablier/internal/event"
	"github.com/espcaa/sablier/internal/harness"
	"github.com/espcaa/sablier/internal/project"
	"github.com/espcaa/sablier/internal/store"
)

func Record(ctx context.Context, s *store.Store, raw []byte, now time.Time, session string) error {
	var h Hook
	if err := json.Unmarshal(raw, &h); err != nil {
		return fmt.Errorf("decode hook: %w", err)
	}
	if h.SessionID == "" {
		return fmt.Errorf("hook %q has no session_id", h.Event)
	}
	salt, err := s.Salt(ctx)
	if err != nil {
		return err
	}
	pid := project.Detect(h.Cwd)
	return harness.Record(ctx, s, h.SessionID, now, func(st State) (State, []event.Event) {
		return Map(st, Input{Hook: h, Now: now, Salt: salt, ProjectID: pid, Session: session})
	})
}
