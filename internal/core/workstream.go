package core

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Hampton-Black/substrate/internal/db"
	sqlcdb "github.com/Hampton-Black/substrate/internal/db/sqlc"
	"github.com/google/uuid"
)

// AddWorkstream creates a new workstream for a pod.
func (s *Service) AddWorkstream(ctx context.Context, in AddWorkstreamInput) (Workstream, error) {
	if _, err := s.GetPod(ctx, in.PodID); err != nil {
		return Workstream{}, err
	}

	status := in.Status
	if status == "" {
		status = WorkstreamActive
	}
	if !status.Valid() {
		return Workstream{}, fmt.Errorf("invalid workstream status: %q", status)
	}

	scope := in.Scope
	if scope == "" {
		scope = ScopeTeam
	}
	if !scope.Valid() {
		return Workstream{}, fmt.Errorf("invalid scope: %q", scope)
	}

	if err := s.checkScope("", in.PodID, scope); err != nil {
		return Workstream{}, err
	}

	id := uuid.NewString()
	now := time.Now().UTC()
	components, _ := json.Marshal([]string{})
	meta, _ := json.Marshal(map[string]any{})

	err := s.q.CreateWorkstream(ctx, sqlcdb.CreateWorkstreamParams{
		ID:           id,
		PodID:        in.PodID,
		Title:        in.Title,
		Intent:       nullString(in.Intent),
		Status:       string(status),
		SpecRef:      sql.NullString{},
		Branch:       nullString(in.Branch),
		Components:   components,
		Scope:        string(scope),
		LastActivity: now,
		CreatedAt:    now,
		Metadata:     meta,
	})
	if err != nil {
		return Workstream{}, fmt.Errorf("create workstream: %w", err)
	}

	if err := s.emitEvent(ctx, in.PodID, id, "workstream.created", map[string]any{
		"workstream_id": id,
		"title":         in.Title,
		"status":        status,
	}); err != nil {
		return Workstream{}, err
	}

	s.log.Info("workstream created", "pod_id", in.PodID, "workstream_id", id)
	return s.GetWorkstream(ctx, id)
}

// GetWorkstream returns a workstream by id.
func (s *Service) GetWorkstream(ctx context.Context, id string) (Workstream, error) {
	row, err := s.q.GetWorkstream(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Workstream{}, fmt.Errorf("workstream not found: %s", id)
		}
		return Workstream{}, fmt.Errorf("get workstream: %w", err)
	}
	return workstreamFromSQLC(row), nil
}

// SetWorkstreamStatus updates a workstream's status.
func (s *Service) SetWorkstreamStatus(ctx context.Context, id string, status WorkstreamStatus) (Workstream, error) {
	if !status.Valid() {
		return Workstream{}, fmt.Errorf("invalid workstream status: %q", status)
	}

	ws, err := s.GetWorkstream(ctx, id)
	if err != nil {
		return Workstream{}, err
	}

	if err := s.checkScope("", ws.PodID, ws.Scope); err != nil {
		return Workstream{}, err
	}

	now := time.Now().UTC()
	if err := s.q.UpdateWorkstreamStatus(ctx, sqlcdb.UpdateWorkstreamStatusParams{
		Status:       string(status),
		LastActivity: now,
		ID:           id,
	}); err != nil {
		return Workstream{}, fmt.Errorf("update workstream status: %w", err)
	}

	if err := s.emitEvent(ctx, ws.PodID, id, "workstream.status_changed", map[string]any{
		"workstream_id": id,
		"status":        status,
	}); err != nil {
		return Workstream{}, err
	}

	s.log.Info("workstream status changed", "pod_id", ws.PodID, "workstream_id", id, "status", status)
	return s.GetWorkstream(ctx, id)
}

// QueryActiveWork returns workstreams matching optional filters.
func (s *Service) QueryActiveWork(ctx context.Context, f WorkFilters) ([]Workstream, error) {
	store := db.NewStore(s.conn)
	filter := db.WorkstreamFilter{}
	if f.Scope != nil {
		sc := string(*f.Scope)
		filter.Scope = &sc
	}
	filter.PodID = f.PodID
	if f.Component != nil {
		filter.Component = f.Component
	}
	if f.Status != nil {
		st := string(*f.Status)
		filter.Status = &st
	}

	rows, err := store.QueryActiveWork(ctx, filter)
	if err != nil {
		return nil, err
	}

	out := make([]Workstream, 0, len(rows))
	for _, r := range rows {
		out = append(out, workstreamFromDynamic(r))
	}
	if out == nil {
		out = []Workstream{}
	}
	return out, nil
}

func nullString(v string) sql.NullString {
	if v == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: v, Valid: true}
}
