package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"

	"github.com/espcaa/sablier/internal/api"
	"github.com/espcaa/sablier/internal/event"
	"github.com/espcaa/sablier/internal/project"
	"github.com/espcaa/sablier/internal/server/db/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func (s *Server) ingest(w http.ResponseWriter, r *http.Request) {
	var b api.Batch
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<20)).Decode(&b); err != nil {
		writeError(w, http.StatusBadRequest, "bad json: "+err.Error())
		return
	}
	var install pgtype.UUID
	if err := install.Scan(b.InstallID); err != nil {
		writeError(w, http.StatusBadRequest, "bad install_id")
		return
	}
	if len(b.Events) > api.MaxBatch {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("at most %d events per batch", api.MaxBatch))
		return
	}
	for _, e := range b.Events {
		if err := validate(e); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("event %d: %v", e.Seq, err))
			return
		}
	}
	for _, p := range b.Projects {
		if !validProjectID.MatchString(p.ID) || !project.ValidName(p.Name) {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("project %q: name must be a folder name, not a path", p.ID))
			return
		}
	}

	u := apiUser(r)
	tx, err := s.db.Begin(r.Context())
	if err != nil {
		log.Print(err)
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	defer tx.Rollback(r.Context())
	q := s.q.WithTx(tx)

	_, err = q.ClaimInstall(r.Context(), sqlc.ClaimInstallParams{ID: install, UserID: u.ID})
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusForbidden, "this install belongs to another account")
		return
	}
	if err != nil {
		log.Print(err)
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	var res api.Result
	for _, e := range b.Events {
		data := []byte(e.Data)
		if len(data) == 0 {
			data = []byte("{}")
		}
		n, err := q.InsertEvent(r.Context(), sqlc.InsertEventParams{
			InstallID: install,
			Seq:       e.Seq,
			Ts:        pgtype.Timestamptz{Time: e.TS, Valid: true},
			Kind:      string(e.Kind),
			SessionID: e.SessionID,
			ProjectID: e.ProjectID,
			Harness:   e.Harness,
			Source:    string(e.Source),
			Data:      data,
		})
		if err != nil {
			log.Print(err)
			writeError(w, http.StatusInternalServerError, "db error")
			return
		}
		if n == 1 {
			res.Accepted++
		} else {
			res.Duplicates++
		}
	}

	for _, p := range b.Projects {
		err := q.UpsertProject(r.Context(), sqlc.UpsertProjectParams{InstallID: install, ProjectID: p.ID, Name: p.Name})
		if err != nil {
			log.Print(err)
			writeError(w, http.StatusInternalServerError, "db error")
			return
		}
	}

	if err := tx.Commit(r.Context()); err != nil {
		log.Print(err)
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func validate(e event.Event) error {
	switch {
	case e.Seq <= 0:
		return errors.New("seq must be positive")
	case e.TS.IsZero():
		return errors.New("missing ts")
	case !e.Kind.Valid():
		return fmt.Errorf("unknown kind %q", e.Kind)
	case len(e.Data) > 0 && !json.Valid(e.Data):
		return errors.New("data is not json")
	}
	return nil
}

var validProjectID = regexp.MustCompile(`^[^\x00-\x1f\x7f]{1,300}$`)
