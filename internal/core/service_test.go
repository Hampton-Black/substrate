package core_test

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/Hampton-Black/substrate/internal/core"
	"github.com/Hampton-Black/substrate/internal/db"
	"github.com/stretchr/testify/require"
)

func newTestService(t *testing.T) (*core.Service, context.Context) {
	t.Helper()
	ctx := context.Background()
	dir := t.TempDir()
	conn, err := db.Open(ctx, filepath.Join(dir, "test.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	return core.NewService(conn, nil, slog.New(slog.NewTextHandler(os.Stderr, nil))), ctx
}

func TestRegisterPodAndList(t *testing.T) {
	svc, ctx := newTestService(t)

	pod, err := svc.RegisterPod(ctx, core.RegisterPodInput{
		Name:  "Solo Pod",
		Owner: "me@example.com",
	})
	require.NoError(t, err)
	require.Equal(t, "solo-pod", pod.ID)

	pods, err := svc.ListPods(ctx)
	require.NoError(t, err)
	require.Len(t, pods, 1)
}

func TestSlugify(t *testing.T) {
	require.Equal(t, "solo-pod", core.Slugify("Solo Pod"))
	require.Equal(t, "hampton-pod", core.Slugify("hampton-pod"))
}

func TestWorkstreamLifecycle(t *testing.T) {
	svc, ctx := newTestService(t)

	_, err := svc.RegisterPod(ctx, core.RegisterPodInput{
		Name:  "solo-pod",
		Owner: "me@example.com",
	})
	require.NoError(t, err)

	ws, err := svc.AddWorkstream(ctx, core.AddWorkstreamInput{
		PodID:  "solo-pod",
		Title:  "Auth refactor",
		Intent: "Migrate to OIDC",
		Branch: "feat/oidc",
	})
	require.NoError(t, err)
	require.Equal(t, core.WorkstreamActive, ws.Status)
	require.Equal(t, "Auth refactor", ws.Title)

	ws, err = svc.SetWorkstreamStatus(ctx, ws.ID, core.WorkstreamReview)
	require.NoError(t, err)
	require.Equal(t, core.WorkstreamReview, ws.Status)
}

func TestGapDedup(t *testing.T) {
	svc, ctx := newTestService(t)

	_, err := svc.RegisterPod(ctx, core.RegisterPodInput{
		Name:  "solo-pod",
		Owner: "me@example.com",
	})
	require.NoError(t, err)

	desc := "Unclear whether refresh tokens should be stored in SQLite or memory"
	g1, err := svc.AddGap(ctx, core.AddGapInput{
		PodID:       "solo-pod",
		Category:    core.GapAmbiguousSpec,
		Description: desc,
		Priority:    2,
	})
	require.NoError(t, err)
	require.Equal(t, 1, g1.Frequency)

	g2, err := svc.AddGap(ctx, core.AddGapInput{
		PodID:       "solo-pod",
		Category:    core.GapAmbiguousSpec,
		Description: desc,
		Priority:    2,
	})
	require.NoError(t, err)
	require.Equal(t, g1.ID, g2.ID)
	require.Equal(t, 2, g2.Frequency)
}

func TestGetPodState(t *testing.T) {
	svc, ctx := newTestService(t)

	_, err := svc.RegisterPod(ctx, core.RegisterPodInput{
		Name:  "solo-pod",
		Owner: "me@example.com",
	})
	require.NoError(t, err)

	_, err = svc.AddWorkstream(ctx, core.AddWorkstreamInput{
		PodID:  "solo-pod",
		Title:  "Auth refactor",
		Intent: "Migrate to OIDC",
		Branch: "feat/oidc",
	})
	require.NoError(t, err)

	_, err = svc.AddGap(ctx, core.AddGapInput{
		PodID:       "solo-pod",
		Category:    core.GapAmbiguousSpec,
		Description: "blocked on spec",
		Priority:    2,
	})
	require.NoError(t, err)

	state, err := svc.GetPodState(ctx, "solo-pod")
	require.NoError(t, err)
	require.Equal(t, "solo-pod", state.Pod.ID)
	require.Len(t, state.ActiveWorkstreams, 1)
	require.Len(t, state.OpenGaps, 1)
	require.NotNil(t, state.LastActivity)
}

func TestQueryActiveWorkAndListGaps(t *testing.T) {
	svc, ctx := newTestService(t)

	_, err := svc.RegisterPod(ctx, core.RegisterPodInput{Name: "solo-pod", Owner: "me@example.com"})
	require.NoError(t, err)

	_, err = svc.AddWorkstream(ctx, core.AddWorkstreamInput{
		PodID: "solo-pod", Title: "T1", Branch: "b1",
	})
	require.NoError(t, err)

	pod := "solo-pod"
	ws, err := svc.QueryActiveWork(ctx, core.WorkFilters{PodID: &pod})
	require.NoError(t, err)
	require.Len(t, ws, 1)

	_, err = svc.AddGap(ctx, core.AddGapInput{
		PodID: "solo-pod", Category: core.GapOther, Description: "x", Priority: 3,
	})
	require.NoError(t, err)

	gaps, err := svc.ListCapabilityGaps(ctx, core.GapFilters{PodID: &pod})
	require.NoError(t, err)
	require.Len(t, gaps, 1)
}
