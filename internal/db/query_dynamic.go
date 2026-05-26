package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// Store provides dynamic SQL queries that are too conditional for sqlc.
type Store struct {
	conn *sql.DB
}

// NewStore wraps a database connection for dynamic queries.
func NewStore(conn *sql.DB) *Store {
	return &Store{conn: conn}
}

// flexJSON scans JSON columns from SQLite whether returned as []byte or string.
type flexJSON []byte

func (f *flexJSON) Scan(value any) error {
	if value == nil {
		*f = nil
		return nil
	}
	switch v := value.(type) {
	case []byte:
		*f = append((*f)[0:0], v...)
	case string:
		*f = []byte(v)
	default:
		return fmt.Errorf("flexJSON: unsupported type %T", value)
	}
	return nil
}

// WorkstreamRow is a workstream row for dynamic queries.
type WorkstreamRow struct {
	ID           string
	PodID        string
	Title        string
	Intent       sql.NullString
	Status       string
	SpecRef      sql.NullString
	Branch       sql.NullString
	Components   flexJSON
	Scope        string
	LastActivity time.Time
	CreatedAt    time.Time
	Metadata     flexJSON
}

// GapRow is a capability gap row for dynamic queries.
type GapRow struct {
	ID            string
	PodID         string
	WorkstreamID  sql.NullString
	Category      string
	Description   string
	Priority      int64
	Status        string
	ResolutionRef sql.NullString
	Frequency     int64
	Scope         string
	OccurredAt    time.Time
	ResolvedAt    sql.NullTime
	Metadata      flexJSON
}

// WorkstreamFilter optional filters for active work queries.
type WorkstreamFilter struct {
	Scope     *string
	PodID     *string
	Component *string
	Status    *string
}

// GapFilter optional filters for capability gap queries.
type GapFilter struct {
	Status      *string
	Category    *string
	PodID       *string
	PriorityMax *int
}

// QueryActiveWork lists workstreams with optional filters.
// sqlc: too dynamic — optional WHERE clauses built at runtime.
func (s *Store) QueryActiveWork(ctx context.Context, f WorkstreamFilter) ([]WorkstreamRow, error) {
	var b strings.Builder
	b.WriteString(`SELECT id, pod_id, title, intent, status, spec_ref, branch, components,
		scope, last_activity, created_at, metadata FROM workstreams WHERE 1=1`)

	args := make([]any, 0, 4)
	if f.Scope != nil && *f.Scope != "" {
		b.WriteString(" AND scope = ?")
		args = append(args, *f.Scope)
	}
	if f.PodID != nil && *f.PodID != "" {
		b.WriteString(" AND pod_id = ?")
		args = append(args, *f.PodID)
	}
	if f.Component != nil && *f.Component != "" {
		b.WriteString(" AND EXISTS (SELECT 1 FROM json_each(components) WHERE value = ?)")
		args = append(args, *f.Component)
	}
	if f.Status != nil && *f.Status != "" {
		b.WriteString(" AND status = ?")
		args = append(args, *f.Status)
	} else {
		b.WriteString(" AND status IN ('planning', 'active', 'blocked', 'review')")
	}
	b.WriteString(" ORDER BY last_activity DESC")

	rows, err := s.conn.QueryContext(ctx, b.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("query active work: %w", err)
	}
	defer rows.Close()

	var out []WorkstreamRow
	for rows.Next() {
		var r WorkstreamRow
		if err := rows.Scan(
			&r.ID, &r.PodID, &r.Title, &r.Intent, &r.Status, &r.SpecRef, &r.Branch,
			&r.Components, &r.Scope, &r.LastActivity, &r.CreatedAt, &r.Metadata,
		); err != nil {
			return nil, fmt.Errorf("scan workstream: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ListCapabilityGaps lists gaps with optional filters.
// sqlc: too dynamic — optional WHERE clauses built at runtime.
func (s *Store) ListCapabilityGaps(ctx context.Context, f GapFilter) ([]GapRow, error) {
	var b strings.Builder
	b.WriteString(`SELECT id, pod_id, workstream_id, category, description, priority, status,
		resolution_ref, frequency, scope, occurred_at, resolved_at, metadata
		FROM capability_gaps WHERE 1=1`)

	args := make([]any, 0, 4)
	if f.PodID != nil && *f.PodID != "" {
		b.WriteString(" AND pod_id = ?")
		args = append(args, *f.PodID)
	}
	if f.Status != nil && *f.Status != "" {
		b.WriteString(" AND status = ?")
		args = append(args, *f.Status)
	}
	if f.Category != nil && *f.Category != "" {
		b.WriteString(" AND category = ?")
		args = append(args, *f.Category)
	}
	if f.PriorityMax != nil {
		b.WriteString(" AND priority <= ?")
		args = append(args, *f.PriorityMax)
	}
	b.WriteString(" ORDER BY priority ASC, occurred_at DESC")

	rows, err := s.conn.QueryContext(ctx, b.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("list capability gaps: %w", err)
	}
	defer rows.Close()

	var out []GapRow
	for rows.Next() {
		var r GapRow
		if err := rows.Scan(
			&r.ID, &r.PodID, &r.WorkstreamID, &r.Category, &r.Description, &r.Priority,
			&r.Status, &r.ResolutionRef, &r.Frequency, &r.Scope, &r.OccurredAt,
			&r.ResolvedAt, &r.Metadata,
		); err != nil {
			return nil, fmt.Errorf("scan gap: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ComponentOwner is the owner of a component.
type ComponentOwner struct {
	PodID       string
	DisplayName string
}

// WhoOwnsComponent resolves component ownership via components + pods tables.
// sqlc: too dynamic — join lookup; kept here for consistency with filtered queries.
func (s *Store) WhoOwnsComponent(ctx context.Context, componentID string) (*ComponentOwner, error) {
	const q = `
		SELECT p.id, p.display_name
		FROM components c
		JOIN pods p ON p.id = c.owner_pod
		WHERE c.id = ?
		LIMIT 1
	`
	var owner ComponentOwner
	err := s.conn.QueryRowContext(ctx, q, componentID).Scan(&owner.PodID, &owner.DisplayName)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("who owns component: %w", err)
	}
	return &owner, nil
}
