package mcp

import (
	"context"
	"log/slog"

	"github.com/Hampton-Black/substrate/internal/core"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func registerWriteTools(server *sdkmcp.Server, svc *core.Service, log *slog.Logger) {
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "substrate.register_capability_gap",
		Description: "Register a capability gap. Dedupes open gaps by pod_id + description.",
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, in registerGapIn) (*sdkmcp.CallToolResult, core.CapabilityGap, error) {
		priority := 3
		if in.Priority != nil {
			priority = *in.Priority
		}
		scope := core.ScopePod
		if in.Scope != nil {
			scope = core.Scope(*in.Scope)
		}
		gap, err := svc.RegisterCapabilityGap(ctx, core.RegisterCapabilityGapInput{
			PodID:        in.PodID,
			WorkstreamID: derefString(in.WorkstreamID),
			Category:     core.GapCategory(in.Category),
			Description:  in.Description,
			Priority:     priority,
			Scope:        scope,
		})
		if err != nil {
			log.Error("register_capability_gap failed", "pod_id", in.PodID, "error", err)
			return nil, core.CapabilityGap{}, err
		}
		return nil, gap, nil
	})

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "substrate.publish_workstream_state",
		Description: "Create or update a workstream. Sets last_activity to now.",
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, in publishWorkstreamIn) (*sdkmcp.CallToolResult, core.Workstream, error) {
		status := core.WorkstreamActive
		if in.Status != nil {
			status = core.WorkstreamStatus(*in.Status)
		}
		scope := core.ScopeTeam
		if in.Scope != nil {
			scope = core.Scope(*in.Scope)
		}
		ws, err := svc.PublishWorkstreamState(ctx, core.PublishWorkstreamStateInput{
			PodID:      in.PodID,
			ID:         derefString(in.ID),
			Title:      derefString(in.Title),
			Intent:     derefString(in.Intent),
			Status:     status,
			Branch:     derefString(in.Branch),
			Components: in.Components,
			SpecRef:    derefString(in.SpecRef),
			Scope:      scope,
		})
		if err != nil {
			log.Error("publish_workstream_state failed", "pod_id", in.PodID, "error", err)
			return nil, core.Workstream{}, err
		}
		return nil, ws, nil
	})
}

type registerGapIn struct {
	PodID        string  `json:"pod_id"`
	WorkstreamID *string `json:"workstream_id,omitempty"`
	Category     string  `json:"category"`
	Description  string  `json:"description"`
	Priority     *int    `json:"priority,omitempty"`
	Scope        *string `json:"scope,omitempty"`
}

type publishWorkstreamIn struct {
	PodID      string   `json:"pod_id"`
	ID         *string  `json:"id,omitempty"`
	Title      *string  `json:"title,omitempty"`
	Intent     *string  `json:"intent,omitempty"`
	Status     *string  `json:"status,omitempty"`
	Branch     *string  `json:"branch,omitempty"`
	Components []string `json:"components,omitempty"`
	SpecRef    *string  `json:"spec_ref,omitempty"`
	Scope      *string  `json:"scope,omitempty"`
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
