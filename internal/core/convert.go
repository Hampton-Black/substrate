package core

import (
	"encoding/json"

	"github.com/Hampton-Black/substrate/internal/db"
	sqlcdb "github.com/Hampton-Black/substrate/internal/db/sqlc"
)

func workstreamFromSQLC(r sqlcdb.Workstream) Workstream {
	w := Workstream{
		ID:           r.ID,
		PodID:        r.PodID,
		Title:        r.Title,
		Status:       WorkstreamStatus(r.Status),
		Scope:        Scope(r.Scope),
		LastActivity: r.LastActivity,
		CreatedAt:    r.CreatedAt,
		Components:   []string{},
	}
	if r.Intent.Valid {
		w.Intent = r.Intent.String
	}
	if r.SpecRef.Valid {
		w.SpecRef = r.SpecRef.String
	}
	if r.Branch.Valid {
		w.Branch = r.Branch.String
	}
	if len(r.Components) > 0 {
		_ = json.Unmarshal(r.Components, &w.Components)
	}
	if w.Components == nil {
		w.Components = []string{}
	}
	if len(r.Metadata) > 0 {
		_ = json.Unmarshal(r.Metadata, &w.Metadata)
	}
	return w
}

func workstreamFromDynamic(r db.WorkstreamRow) Workstream {
	w := Workstream{
		ID:           r.ID,
		PodID:        r.PodID,
		Title:        r.Title,
		Status:       WorkstreamStatus(r.Status),
		Scope:        Scope(r.Scope),
		LastActivity: r.LastActivity,
		CreatedAt:    r.CreatedAt,
		Components:   []string{},
	}
	if r.Intent.Valid {
		w.Intent = r.Intent.String
	}
	if r.SpecRef.Valid {
		w.SpecRef = r.SpecRef.String
	}
	if r.Branch.Valid {
		w.Branch = r.Branch.String
	}
	if len(r.Components) > 0 {
		_ = json.Unmarshal(r.Components, &w.Components)
	}
	if w.Components == nil {
		w.Components = []string{}
	}
	if len(r.Metadata) > 0 {
		_ = json.Unmarshal(r.Metadata, &w.Metadata)
	}
	return w
}

func gapFromSQLC(r sqlcdb.CapabilityGap) CapabilityGap {
	g := CapabilityGap{
		ID:          r.ID,
		PodID:       r.PodID,
		Category:    GapCategory(r.Category),
		Description: r.Description,
		Priority:    int(r.Priority),
		Status:      GapStatus(r.Status),
		Frequency:   int(r.Frequency),
		Scope:       Scope(r.Scope),
		OccurredAt:  r.OccurredAt,
	}
	if r.WorkstreamID.Valid {
		g.WorkstreamID = r.WorkstreamID.String
	}
	if r.ResolutionRef.Valid {
		g.ResolutionRef = r.ResolutionRef.String
	}
	if r.ResolvedAt.Valid {
		t := r.ResolvedAt.Time
		g.ResolvedAt = &t
	}
	if len(r.Metadata) > 0 {
		_ = json.Unmarshal(r.Metadata, &g.Metadata)
	}
	return g
}

func gapFromDynamic(r db.GapRow) CapabilityGap {
	g := CapabilityGap{
		ID:          r.ID,
		PodID:       r.PodID,
		Category:    GapCategory(r.Category),
		Description: r.Description,
		Priority:    int(r.Priority),
		Status:      GapStatus(r.Status),
		Frequency:   int(r.Frequency),
		Scope:       Scope(r.Scope),
		OccurredAt:  r.OccurredAt,
	}
	if r.WorkstreamID.Valid {
		g.WorkstreamID = r.WorkstreamID.String
	}
	if r.ResolutionRef.Valid {
		g.ResolutionRef = r.ResolutionRef.String
	}
	if r.ResolvedAt.Valid {
		t := r.ResolvedAt.Time
		g.ResolvedAt = &t
	}
	if len(r.Metadata) > 0 {
		_ = json.Unmarshal(r.Metadata, &g.Metadata)
	}
	return g
}
