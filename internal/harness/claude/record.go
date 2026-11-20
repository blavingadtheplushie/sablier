package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/espcaa/sablier/internal/project"
	"github.com/espcaa/sablier/internal/store"
)

// one transaction so concurrent hooks for a session can't interleave.
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

	return s.Update(ctx, func(tx *store.Tx) error {
		var st State
		if data, err := tx.SessionState(ctx, h.SessionID); err != nil {
			return err
		} else if data != nil {
			if err := json.Unmarshal(data, &st); err != nil {
				return fmt.Errorf("decode session state: %w", err)
			}
		}

		pid := project.Detect(h.Cwd)

		st, events := Map(st, Input{Hook: h, Now: now, Salt: salt, ProjectID: pid, Session: session})

		for _, e := range events {
			if _, err := tx.Append(ctx, e); err != nil {
				return err
			}
		}
		data, err := json.Marshal(st)
		if err != nil {
			return err
		}
		return tx.PutSessionState(ctx, h.SessionID, data, now)
	})
}
