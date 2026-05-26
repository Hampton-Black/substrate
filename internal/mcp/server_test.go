package mcp_test

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/Hampton-Black/substrate/internal/config"
	"github.com/Hampton-Black/substrate/internal/core"
	"github.com/Hampton-Black/substrate/internal/db"
	"github.com/stretchr/testify/require"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestMCPReadTools(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	dbPath := filepath.Join(dir, "substrate.db")
	repoPath := filepath.Join(dir, "repo")

	cfg := config.Default()
	cfg.Storage.DBPath = dbPath
	cfg.Storage.GitRepoDir = repoPath
	require.NoError(t, config.Save(cfgPath, cfg))

	conn, err := db.Open(ctx, dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	svc := core.NewService(conn, nil, nil)
	_, err = svc.RegisterPod(ctx, core.RegisterPodInput{Name: "solo-pod", Owner: "me@example.com"})
	require.NoError(t, err)
	ws, err := svc.AddWorkstream(ctx, core.AddWorkstreamInput{
		PodID: "solo-pod", Title: "Auth refactor", Intent: "Migrate to OIDC", Branch: "feat/oidc",
	})
	require.NoError(t, err)
	_, err = svc.AddGap(ctx, core.AddGapInput{
		PodID: "solo-pod", Category: core.GapAmbiguousSpec,
		Description: "Unclear whether refresh tokens should be stored in SQLite or memory",
		Priority: 2,
	})
	require.NoError(t, err)
	_, err = svc.AddGap(ctx, core.AddGapInput{
		PodID: "solo-pod", Category: core.GapOther,
		Description: "Low priority gap",
		Priority: 4,
	})
	require.NoError(t, err)
	_, err = conn.ExecContext(ctx,
		`UPDATE workstreams SET components = ? WHERE id = ?`,
		[]byte(`["auth-module"]`), ws.ID,
	)
	require.NoError(t, err)
	_, err = conn.ExecContext(ctx,
		`INSERT INTO components (id, display_name, owner_pod, last_updated, metadata)
		 VALUES (?, ?, ?, ?, ?)`,
		"auth-module", "Auth Module", "solo-pod", time.Now().UTC(), `{}`,
	)
	require.NoError(t, err)
	_ = conn.Close()

	root, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err)

	bin := filepath.Join(dir, "substrate")
	build := exec.Command("go", "build", "-o", bin, "./cmd/substrate")
	build.Dir = root
	build.Env = append(os.Environ(), "SUBSTRATE_CONFIG="+cfgPath)
	out, err := build.CombinedOutput()
	require.NoError(t, err, string(out))

	cmd := exec.Command(bin, "mcp", "serve")
	cmd.Env = append(os.Environ(), "SUBSTRATE_CONFIG="+cfgPath)

	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "test-client", Version: "0.1.0"}, nil)
	session, err := client.Connect(ctx, &sdkmcp.CommandTransport{Command: cmd}, nil)
	require.NoError(t, err)
	defer session.Close()

	t.Run("query_active_work", func(t *testing.T) {
		res, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
			Name:      "substrate.query_active_work",
			Arguments: map[string]any{"pod_id": "solo-pod"},
		})
		require.NoError(t, err)
		require.False(t, res.IsError)

		var body struct {
			Workstreams []core.Workstream `json:"workstreams"`
		}
		require.NoError(t, decodeStructured(res.StructuredContent, &body))
		require.Len(t, body.Workstreams, 1)
		require.Equal(t, "Auth refactor", body.Workstreams[0].Title)
	})

	t.Run("list_capability_gaps", func(t *testing.T) {
		res, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
			Name:      "substrate.list_capability_gaps",
			Arguments: map[string]any{"pod_id": "solo-pod"},
		})
		require.NoError(t, err)
		require.False(t, res.IsError)

		var body struct {
			Gaps []core.CapabilityGap `json:"gaps"`
		}
		require.NoError(t, decodeStructured(res.StructuredContent, &body))
		require.Len(t, body.Gaps, 2)
	})

	t.Run("get_pod_state", func(t *testing.T) {
		res, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
			Name:      "substrate.get_pod_state",
			Arguments: map[string]any{"pod_id": "solo-pod"},
		})
		require.NoError(t, err)
		require.False(t, res.IsError)

		var body core.PodState
		require.NoError(t, decodeStructured(res.StructuredContent, &body))
		require.Equal(t, "solo-pod", body.Pod.ID)
		require.Len(t, body.ActiveWorkstreams, 1)
		require.Len(t, body.OpenGaps, 2)
	})

	t.Run("get_workstream", func(t *testing.T) {
		res, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
			Name:      "substrate.get_workstream",
			Arguments: map[string]any{"id": ws.ID},
		})
		require.NoError(t, err)
		require.False(t, res.IsError)

		var body core.Workstream
		require.NoError(t, decodeStructured(res.StructuredContent, &body))
		require.Equal(t, ws.ID, body.ID)
		require.Equal(t, "Auth refactor", body.Title)
	})

	t.Run("who_owns", func(t *testing.T) {
		res, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
			Name:      "substrate.who_owns",
			Arguments: map[string]any{"component": "auth-module"},
		})
		require.NoError(t, err)
		require.False(t, res.IsError)

		var body struct {
			PodID       string `json:"pod_id"`
			DisplayName string `json:"display_name"`
		}
		require.NoError(t, decodeStructured(res.StructuredContent, &body))
		require.Equal(t, "solo-pod", body.PodID)
		require.Equal(t, "solo-pod", body.DisplayName)
	})

	t.Run("query_active_work_component_filter", func(t *testing.T) {
		res, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
			Name:      "substrate.query_active_work",
			Arguments: map[string]any{"component": "auth-module"},
		})
		require.NoError(t, err)
		require.False(t, res.IsError)

		var body struct {
			Workstreams []core.Workstream `json:"workstreams"`
		}
		require.NoError(t, decodeStructured(res.StructuredContent, &body))
		require.Len(t, body.Workstreams, 1)
		require.Equal(t, ws.ID, body.Workstreams[0].ID)
	})

	t.Run("list_capability_gaps_priority_max", func(t *testing.T) {
		res, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
			Name:      "substrate.list_capability_gaps",
			Arguments: map[string]any{"pod_id": "solo-pod", "priority_max": 2},
		})
		require.NoError(t, err)
		require.False(t, res.IsError)

		var body struct {
			Gaps []core.CapabilityGap `json:"gaps"`
		}
		require.NoError(t, decodeStructured(res.StructuredContent, &body))
		require.Len(t, body.Gaps, 1)
		require.Equal(t, 2, body.Gaps[0].Priority)
	})
}

func decodeStructured(v any, dest any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, dest)
}
