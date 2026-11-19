// store is the local event db. several processes (hooks, launcher) write to it at once.
package store

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/espcaa/sablier/internal/event"
	"github.com/espcaa/sablier/internal/store/sqlc"
	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schema string

type Store struct {
	db *sql.DB
	q  *sqlc.Queries
}

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	// wal + busy_timeout so concurrent hook processes wait instead of failing
	dsn := "file:" + path + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("apply schema: %w", err)
	}
	return &Store{db: db, q: sqlc.New(db)}, nil
}

func (s *Store) Close() error { return s.db.Close() }

// Append saves e and returns it with Seq set.
func (s *Store) Append(ctx context.Context, e event.Event) (event.Event, error) {
	data := string(e.Data)
	if data == "" {
		data = "{}"
	}
	seq, err := s.q.InsertEvent(ctx, sqlc.InsertEventParams{
		Ts:        e.TS.UnixMilli(),
		Kind:      string(e.Kind),
		SessionID: e.SessionID,
		ProjectID: e.ProjectID,
		Harness:   e.Harness,
		Source:    string(e.Source),
		Data:      data,
	})
	if err != nil {
		return e, err
	}
	e.Seq = seq
	return e, nil
}

// Between returns events with from <= ts < to, oldest first.
func (s *Store) Between(ctx context.Context, from, to time.Time) ([]event.Event, error) {
	rows, err := s.q.EventsBetween(ctx, sqlc.EventsBetweenParams{
		FromTs: from.UnixMilli(),
		ToTs:   to.UnixMilli(),
	})
	if err != nil {
		return nil, err
	}
	return fromRows(rows), nil
}

// Pending returns up to limit events not uploaded yet, by seq.
func (s *Store) Pending(ctx context.Context, limit int64) ([]event.Event, error) {
	rows, err := s.q.PendingEvents(ctx, limit)
	if err != nil {
		return nil, err
	}
	return fromRows(rows), nil
}

// MarkUploaded marks every event up to and including seq as sent.
func (s *Store) MarkUploaded(ctx context.Context, seq int64, at time.Time) error {
	return s.q.MarkUploaded(ctx, sqlc.MarkUploadedParams{
		UploadedAt: sql.NullInt64{Int64: at.UnixMilli(), Valid: true},
		UpToSeq:    seq,
	})
}

func fromRows(rows []sqlc.Event) []event.Event {
	out := make([]event.Event, len(rows))
	for i, r := range rows {
		out[i] = event.Event{
			Seq:       r.Seq,
			TS:        time.UnixMilli(r.Ts).UTC(),
			Kind:      event.Kind(r.Kind),
			SessionID: r.SessionID,
			ProjectID: r.ProjectID,
			Harness:   r.Harness,
			Source:    event.Source(r.Source),
			Data:      []byte(r.Data),
		}
	}
	return out
}
