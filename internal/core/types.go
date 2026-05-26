package core

import (
	"context"
	"time"
)

// WorkstreamStatus is the lifecycle state of a workstream.
type WorkstreamStatus string

const (
	WorkstreamPlanning  WorkstreamStatus = "planning"
	WorkstreamActive    WorkstreamStatus = "active"
	WorkstreamBlocked   WorkstreamStatus = "blocked"
	WorkstreamReview    WorkstreamStatus = "review"
	WorkstreamMerged    WorkstreamStatus = "merged"
	WorkstreamAbandoned WorkstreamStatus = "abandoned"
)

func (s WorkstreamStatus) Valid() bool {
	switch s {
	case WorkstreamPlanning, WorkstreamActive, WorkstreamBlocked,
		WorkstreamReview, WorkstreamMerged, WorkstreamAbandoned:
		return true
	default:
		return false
	}
}

// Scope controls visibility of entities.
type Scope string

const (
	ScopePod     Scope = "pod"
	ScopeTeam    Scope = "team"
	ScopeOrg     Scope = "org"
	ScopePublic  Scope = "public"
)

func (s Scope) Valid() bool {
	switch s {
	case ScopePod, ScopeTeam, ScopeOrg, ScopePublic:
		return true
	default:
		return false
	}
}

// GapCategory classifies why an agent was blocked.
type GapCategory string

const (
	GapMissingTool      GapCategory = "missing_tool"
	GapMissingSkill     GapCategory = "missing_skill"
	GapAmbiguousSpec    GapCategory = "ambiguous_spec"
	GapMissingContext   GapCategory = "missing_context"
	GapEnvConstraint    GapCategory = "env_constraint"
	GapOther            GapCategory = "other"
)

func (c GapCategory) Valid() bool {
	switch c {
	case GapMissingTool, GapMissingSkill, GapAmbiguousSpec,
		GapMissingContext, GapEnvConstraint, GapOther:
		return true
	default:
		return false
	}
}

// GapStatus is the lifecycle state of a capability gap.
type GapStatus string

const (
	GapOpen          GapStatus = "open"
	GapAcknowledged  GapStatus = "acknowledged"
	GapInProgress    GapStatus = "in_progress"
	GapResolved      GapStatus = "resolved"
	GapWontFix       GapStatus = "wontfix"
)

func (s GapStatus) Valid() bool {
	switch s {
	case GapOpen, GapAcknowledged, GapInProgress, GapResolved, GapWontFix:
		return true
	default:
		return false
	}
}

// Pod is a developer+agent fleet.
type Pod struct {
	ID          string         `json:"id"`
	DisplayName string         `json:"display_name"`
	Owner       string         `json:"owner"`
	PublicKey   string         `json:"public_key,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	Active      bool           `json:"active"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// Workstream is an active thread of work within a pod.
type Workstream struct {
	ID           string            `json:"id"`
	PodID        string            `json:"pod_id"`
	Title        string            `json:"title"`
	Intent       string            `json:"intent,omitempty"`
	Status       WorkstreamStatus  `json:"status"`
	SpecRef      string            `json:"spec_ref,omitempty"`
	Branch       string            `json:"branch,omitempty"`
	Components   []string          `json:"components"`
	Scope        Scope             `json:"scope"`
	LastActivity time.Time         `json:"last_activity"`
	CreatedAt    time.Time         `json:"created_at"`
	Metadata     map[string]any    `json:"metadata,omitempty"`
}

// CapabilityGap records something that blocked an agent.
type CapabilityGap struct {
	ID            string         `json:"id"`
	PodID         string         `json:"pod_id"`
	WorkstreamID  string         `json:"workstream_id,omitempty"`
	Category      GapCategory    `json:"category"`
	Description   string         `json:"description"`
	Priority      int            `json:"priority"`
	Status        GapStatus      `json:"status"`
	ResolutionRef string         `json:"resolution_ref,omitempty"`
	Frequency     int            `json:"frequency"`
	Scope         Scope          `json:"scope"`
	OccurredAt    time.Time      `json:"occurred_at"`
	ResolvedAt    *time.Time     `json:"resolved_at,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
}

// PodState is a snapshot of a pod's current activity.
type PodState struct {
	Pod               Pod              `json:"pod"`
	ActiveWorkstreams []Workstream     `json:"active_workstreams"`
	OpenGaps          []CapabilityGap  `json:"open_gaps"`
	LastActivity      *time.Time       `json:"last_activity,omitempty"`
}

// ComponentOwner is the result of who_owns lookup.
type ComponentOwner struct {
	PodID       string `json:"pod_id"`
	DisplayName string `json:"display_name"`
}

// WorkFilters optional filters for query_active_work.
type WorkFilters struct {
	Scope     *Scope
	PodID     *string
	Component *string
	Status    *WorkstreamStatus
}

// GapFilters optional filters for list_capability_gaps.
type GapFilters struct {
	Status      *GapStatus
	Statuses    []GapStatus
	Category    *GapCategory
	PodID       *string
	PriorityMax *int
}

// RegisterPodInput is input for pod registration.
type RegisterPodInput struct {
	Name  string
	Owner string
}

// AddWorkstreamInput is input for creating a workstream via CLI.
type AddWorkstreamInput struct {
	PodID  string
	Title  string
	Intent string
	Branch string
	Status WorkstreamStatus
	Scope  Scope
}

// AddGapInput is input for registering a capability gap via CLI.
type AddGapInput struct {
	PodID        string
	WorkstreamID string
	Category     GapCategory
	Description  string
	Priority     int
}

// RegisterCapabilityGapInput is input for MCP gap registration.
type RegisterCapabilityGapInput struct {
	PodID        string
	WorkstreamID string
	Category     GapCategory
	Description  string
	Priority     int
	Scope        Scope
}

// PublishWorkstreamStateInput is input for MCP workstream publish/upsert.
type PublishWorkstreamStateInput struct {
	PodID      string
	ID         string
	Title      string
	Intent     string
	Status     WorkstreamStatus
	Branch     string
	Components []string
	SpecRef    string
	Scope      Scope
}

// CompletionRequest is a minimal LLM completion request (stub for later milestones).
type CompletionRequest struct {
	Prompt string
}

// CompletionResponse is a minimal LLM completion response.
type CompletionResponse struct {
	Text string
}

// LLMBackend is the pluggable synthesis LLM interface (not wired in M0-M1).
type LLMBackend interface {
	Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error)
}
