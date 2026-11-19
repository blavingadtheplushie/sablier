// local event db, written by several processes at once
package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	_ "embed"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/espcaa/sablier/internal/event"
	"github.com/espcaa/sablier/internal/store/sqlc"
	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
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

	dsn := "file:" + path + "?_pragma=busy_timeout(5000)&_txlock=immediate"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	if err := useWAL(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable wal: %w", err)
	}
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("apply schema: %w", err)
	}
	return &Store{db: db, q: sqlc.New(db)}, nil
}

// busy_timeout doesn't cover switching to wal, so racing first opens retry
func useWAL(db *sql.DB) error {
	deadline := time.Now().Add(5 * time.Second)
	for {
		var mode string
		err := db.QueryRow("PRAGMA journal_mode=WAL").Scan(&mode)
		var sqliteErr *sqlite.Error
		busy := errors.As(err, &sqliteErr) && sqliteErr.Code()&0xff == sqlite3.SQLITE_BUSY
		if !busy || time.Now().After(deadline) {
			return err
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) Append(ctx context.Context, e event.Event) (event.Event, error) {
	err := s.Update(ctx, func(tx *Tx) error {
		var err error
		e, err = tx.Append(ctx, e)
		return err
	})
	return e, err
}

type Tx struct{ q *sqlc.Queries }

func (s *Store) Update(ctx context.Context, fn func(*Tx) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(&Tx{q: s.q.WithTx(tx)}); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (t *Tx) Append(ctx context.Context, e event.Event) (event.Event, error) {
	data := string(e.Data)
	if data == "" {
		data = "{}"
	}
	seq, err := t.q.InsertEvent(ctx, sqlc.InsertEventParams{
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

func (t *Tx) SessionState(ctx context.Context, sessionID string) ([]byte, error) {
	data, err := t.q.GetSessionState(ctx, sessionID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return []byte(data), err
}

func (t *Tx) PutSessionState(ctx context.Context, sessionID string, data []byte, now time.Time) error {
	return t.q.PutSessionState(ctx, sqlc.PutSessionStateParams{
		SessionID: sessionID,
		Data:      string(data),
		UpdatedAt: now.UnixMilli(),
	})
}

func (s *Store) Salt(ctx context.Context) ([]byte, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	if err := s.q.SetMetaIfMissing(ctx, sqlc.SetMetaIfMissingParams{Key: "salt", Value: hex.EncodeToString(b)}); err != nil {
		return nil, err
	}
	v, err := s.q.GetMeta(ctx, "salt")
	if err != nil {
		return nil, err
	}
	return hex.DecodeString(v)
}

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

func (t *Tx) PutProject(ctx context.Context, id, root string) error {
	return t.q.PutProject(ctx, sqlc.PutProjectParams{ID: id, Root: root})
}

// project id to root dir
func (s *Store) Projects(ctx context.Context) (map[string]string, error) {
	rows, err := s.q.Projects(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(rows))
	for _, r := range rows {
		out[r.ID] = r.Root
	}
	return out, nil
}

// events not uploaded yet, by seq
func (s *Store) Pending(ctx context.Context, limit int64) ([]event.Event, error) {
	rows, err := s.q.PendingEvents(ctx, limit)
	if err != nil {
		return nil, err
	}
	return fromRows(rows), nil
}

// marks every event up to seq as sent
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
