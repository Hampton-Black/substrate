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

// AddGap registers a capability gap with exact-match dedup on pod_id + description.
func (s *Service) AddGap(ctx context.Context, in AddGapInput) (CapabilityGap, error) {
	if _, err := s.GetPod(ctx, in.PodID); err != nil {
		return CapabilityGap{}, err
	}

	if !in.Category.Valid() {
		return CapabilityGap{}, fmt.Errorf("invalid gap category: %q", in.Category)
	}

	priority := in.Priority
	if priority < 1 || priority > 5 {
		return CapabilityGap{}, fmt.Errorf("priority must be 1-5, got %d", priority)
	}

	if err := s.checkScope("", in.PodID, ScopePod); err != nil {
		return CapabilityGap{}, err
	}

	existing, err := s.q.GetGapByPodAndDescription(ctx, sqlcdb.GetGapByPodAndDescriptionParams{
		PodID:       in.PodID,
		Description: in.Description,
	})
	if err == nil {
		now := time.Now().UTC()
		if err := s.q.IncrementGapFrequency(ctx, sqlcdb.IncrementGapFrequencyParams{
			OccurredAt: now,
			ID:         existing.ID,
		}); err != nil {
			return CapabilityGap{}, fmt.Errorf("increment gap frequency: %w", err)
		}
		wsEventID := ""
		if existing.WorkstreamID.Valid {
			wsEventID = existing.WorkstreamID.String
		}
		if err := s.emitEvent(ctx, in.PodID, wsEventID, "gap.deduped", map[string]any{
			"gap_id":      existing.ID,
			"description": in.Description,
		}); err != nil {
			return CapabilityGap{}, err
		}
		s.log.Info("gap deduped", "pod_id", in.PodID, "gap_id", existing.ID)
		return s.getGap(ctx, existing.ID)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return CapabilityGap{}, fmt.Errorf("lookup gap: %w", err)
	}

	id := uuid.NewString()
	now := time.Now().UTC()
	meta, _ := json.Marshal(map[string]any{})

	wsID := sql.NullString{}
	if in.WorkstreamID != "" {
		wsID = sql.NullString{String: in.WorkstreamID, Valid: true}
	}

	err = s.q.CreateGap(ctx, sqlcdb.CreateGapParams{
		ID:            id,
		PodID:         in.PodID,
		WorkstreamID:  wsID,
		Category:      string(in.Category),
		Description:   in.Description,
		Priority:      int64(priority),
		Status:        string(GapOpen),
		ResolutionRef: sql.NullString{},
		Frequency:     1,
		Scope:         string(ScopePod),
		OccurredAt:    now,
		ResolvedAt:    sql.NullTime{},
		Metadata:      meta,
	})
	if err != nil {
		return CapabilityGap{}, fmt.Errorf("create gap: %w", err)
	}

	if err := s.emitEvent(ctx, in.PodID, in.WorkstreamID, "gap.registered", map[string]any{
		"gap_id":      id,
		"category":    in.Category,
		"description": in.Description,
	}); err != nil {
		return CapabilityGap{}, err
	}

	s.log.Info("gap registered", "pod_id", in.PodID, "gap_id", id)
	return s.getGap(ctx, id)
}

func (s *Service) getGap(ctx context.Context, id string) (CapabilityGap, error) {
	row, err := s.q.GetGap(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return CapabilityGap{}, fmt.Errorf("gap not found: %s", id)
		}
		return CapabilityGap{}, fmt.Errorf("get gap: %w", err)
	}
	return gapFromSQLC(row), nil
}

// ListCapabilityGaps returns gaps matching optional filters.
func (s *Service) ListCapabilityGaps(ctx context.Context, f GapFilters) ([]CapabilityGap, error) {
	store := db.NewStore(s.conn)
	filter := db.GapFilter{PodID: f.PodID}
	if f.Status != nil {
		st := string(*f.Status)
		filter.Status = &st
	}
	if f.Category != nil {
		cat := string(*f.Category)
		filter.Category = &cat
	}
	filter.PriorityMax = f.PriorityMax

	rows, err := store.ListCapabilityGaps(ctx, filter)
	if err != nil {
		return nil, err
	}

	out := make([]CapabilityGap, 0, len(rows))
	for _, r := range rows {
		out = append(out, gapFromDynamic(r))
	}
	if out == nil {
		out = []CapabilityGap{}
	}
	return out, nil
}

// WhoOwns resolves a component to its owner pod.
// TODO(spec): components table is populated in M6; returns empty owner when not found.
func (s *Service) WhoOwns(ctx context.Context, componentID string) (*ComponentOwner, error) {
	store := db.NewStore(s.conn)
	owner, err := store.WhoOwnsComponent(ctx, componentID)
	if err != nil {
		return nil, err
	}
	if owner == nil {
		return nil, nil
	}
	return &ComponentOwner{
		PodID:       owner.PodID,
		DisplayName: owner.DisplayName,
	}, nil
}

// GetPodState returns a snapshot of pod activity.
func (s *Service) GetPodState(ctx context.Context, podID string) (PodState, error) {
	pod, err := s.GetPod(ctx, podID)
	if err != nil {
		return PodState{}, err
	}

	wsRows, err := s.q.ListActiveWorkstreamsByPod(ctx, podID)
	if err != nil {
		return PodState{}, fmt.Errorf("list active workstreams: %w", err)
	}
	workstreams := make([]Workstream, 0, len(wsRows))
	var lastActivity *time.Time
	for _, r := range wsRows {
		w := workstreamFromSQLC(r)
		workstreams = append(workstreams, w)
		if lastActivity == nil || w.LastActivity.After(*lastActivity) {
			t := w.LastActivity
			lastActivity = &t
		}
	}

	gapRows, err := s.q.ListOpenGapsByPod(ctx, podID)
	if err != nil {
		return PodState{}, fmt.Errorf("list open gaps: %w", err)
	}
	gaps := make([]CapabilityGap, 0, len(gapRows))
	for _, r := range gapRows {
		gaps = append(gaps, gapFromSQLC(r))
	}

	return PodState{
		Pod:               pod,
		ActiveWorkstreams: workstreams,
		OpenGaps:          gaps,
		LastActivity:      lastActivity,
	}, nil
}
