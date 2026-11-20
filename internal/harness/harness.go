// shared by harnesses whose hooks run `sablier hook`
package harness

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/espcaa/sablier/internal/event"
	"github.com/espcaa/sablier/internal/store"
)

// loads a session's state, maps one hook and saves both in one transaction,
func Record[S any](ctx context.Context, s *store.Store, session string, now time.Time, mapFn func(S) (S, []event.Event)) error {
	return s.Update(ctx, func(tx *store.Tx) error {
		var st S
		if data, err := tx.SessionState(ctx, session); err != nil {
			return err
		} else if data != nil {
			if err := json.Unmarshal(data, &st); err != nil {
				return fmt.Errorf("decode session state: %w", err)
			}
		}

		st, events := mapFn(st)
		for _, e := range events {
			if _, err := tx.Append(ctx, e); err != nil {
				return err
			}
		}
		data, err := json.Marshal(st)
		if err != nil {
			return err
		}
		return tx.PutSessionState(ctx, session, data, now)
	})
}

// a path for a hook command line
func Quote(path string) string {
	if runtime.GOOS == "windows" {
		return `"` + filepath.ToSlash(path) + `"`
	}
	return "'" + strings.ReplaceAll(path, "'", `'\''`) + "'"
}
