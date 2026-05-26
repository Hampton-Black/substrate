package mcp

import (
	"context"
	"log/slog"

	"github.com/Hampton-Black/substrate/internal/core"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func registerReadTools(server *sdkmcp.Server, svc *core.Service, log *slog.Logger) {
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "substrate.query_active_work",
		Description: "List active workstreams with optional scope, pod_id, and status filters.",
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, in queryActiveWorkIn) (*sdkmcp.CallToolResult, queryActiveWorkOut, error) {
		f := core.WorkFilters{}
		if in.Scope != nil {
			s := core.Scope(*in.Scope)
			f.Scope = &s
		}
		f.PodID = in.PodID
		if in.Component != nil {
			f.Component = in.Component
		}
		if in.Status != nil {
			st := core.WorkstreamStatus(*in.Status)
			f.Status = &st
		}
		ws, err := svc.QueryActiveWork(ctx, f)
		if err != nil {
			log.Error("query_active_work failed", "error", err)
			return nil, queryActiveWorkOut{}, err
		}
		return nil, queryActiveWorkOut{Workstreams: ws}, nil
	})

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "substrate.get_workstream",
		Description: "Get a single workstream by ID.",
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, in getWorkstreamIn) (*sdkmcp.CallToolResult, core.Workstream, error) {
		ws, err := svc.GetWorkstream(ctx, in.ID)
		if err != nil {
			log.Error("get_workstream failed", "workstream_id", in.ID, "error", err)
			return nil, core.Workstream{}, err
		}
		return nil, ws, nil
	})

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "substrate.list_capability_gaps",
		Description: "List capability gaps with optional status, category, and pod_id filters.",
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, in listGapsIn) (*sdkmcp.CallToolResult, listGapsOut, error) {
		f := core.GapFilters{PodID: in.PodID}
		if in.Status != nil {
			st := core.GapStatus(*in.Status)
			f.Status = &st
		}
		if in.Category != nil {
			cat := core.GapCategory(*in.Category)
			f.Category = &cat
		}
		f.PriorityMax = in.PriorityMax
		gaps, err := svc.ListCapabilityGaps(ctx, f)
		if err != nil {
			log.Error("list_capability_gaps failed", "error", err)
			return nil, listGapsOut{}, err
		}
		return nil, listGapsOut{Gaps: gaps}, nil
	})

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "substrate.who_owns",
		Description: "Resolve a component to its owner pod.",
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, in whoOwnsIn) (*sdkmcp.CallToolResult, whoOwnsOut, error) {
		owner, err := svc.WhoOwns(ctx, in.Component)
		if err != nil {
			log.Error("who_owns failed", "component", in.Component, "error", err)
			return nil, whoOwnsOut{}, err
		}
		if owner == nil {
			return nil, whoOwnsOut{}, nil
		}
		return nil, whoOwnsOut{
			PodID:       owner.PodID,
			DisplayName: owner.DisplayName,
		}, nil
	})

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "substrate.get_pod_state",
		Description: "Snapshot of a pod: active workstreams, open gaps, last activity.",
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, in getPodStateIn) (*sdkmcp.CallToolResult, core.PodState, error) {
		state, err := svc.GetPodState(ctx, in.PodID)
		if err != nil {
			log.Error("get_pod_state failed", "pod_id", in.PodID, "error", err)
			return nil, core.PodState{}, err
		}
		return nil, state, nil
	})
}

type queryActiveWorkIn struct {
	Scope     *string `json:"scope,omitempty"`
	PodID     *string `json:"pod_id,omitempty"`
	Component *string `json:"component,omitempty"`
	Status    *string `json:"status,omitempty"`
}

type queryActiveWorkOut struct {
	Workstreams []core.Workstream `json:"workstreams"`
}

type getWorkstreamIn struct {
	ID string `json:"id"`
}

type listGapsIn struct {
	Status      *string `json:"status,omitempty"`
	Category    *string `json:"category,omitempty"`
	PodID       *string `json:"pod_id,omitempty"`
	PriorityMax *int    `json:"priority_max,omitempty"`
}

type listGapsOut struct {
	Gaps []core.CapabilityGap `json:"gaps"`
}

type whoOwnsIn struct {
	Component string `json:"component"`
}

type whoOwnsOut struct {
	PodID       string `json:"pod_id,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
}

type getPodStateIn struct {
	PodID string `json:"pod_id"`
}
