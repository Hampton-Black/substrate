package core_test

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Hampton-Black/substrate/internal/core"
	"github.com/Hampton-Black/substrate/internal/db"
	"github.com/stretchr/testify/require"
)

func newTestService(t *testing.T) (*core.Service, *sql.DB, context.Context) {
	t.Helper()
	ctx := context.Background()
	dir := t.TempDir()
	conn, err := db.Open(ctx, filepath.Join(dir, "test.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	return core.NewService(conn, nil, slog.New(slog.NewTextHandler(os.Stderr, nil))), conn, ctx
}

func TestRegisterPodAndList(t *testing.T) {
	svc, _, ctx := newTestService(t)

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
	svc, _, ctx := newTestService(t)

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
	svc, _, ctx := newTestService(t)

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
	svc, _, ctx := newTestService(t)

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
	svc, _, ctx := newTestService(t)

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

func TestGapEventWorkstreamID(t *testing.T) {
	svc, conn, ctx := newTestService(t)

	_, err := svc.RegisterPod(ctx, core.RegisterPodInput{Name: "solo-pod", Owner: "me@example.com"})
	require.NoError(t, err)

	_, err = svc.AddGap(ctx, core.AddGapInput{
		PodID: "solo-pod", Category: core.GapOther, Description: "no workstream", Priority: 3,
	})
	require.NoError(t, err)

	var wsID sql.NullString
	err = conn.QueryRowContext(ctx,
		`SELECT workstream_id FROM events WHERE event_type = 'gap.registered' LIMIT 1`,
	).Scan(&wsID)
	require.NoError(t, err)
	require.False(t, wsID.Valid)
}

func TestWhoOwns(t *testing.T) {
	svc, conn, ctx := newTestService(t)

	_, err := svc.RegisterPod(ctx, core.RegisterPodInput{Name: "solo-pod", Owner: "me@example.com"})
	require.NoError(t, err)

	owner, err := svc.WhoOwns(ctx, "auth-module")
	require.NoError(t, err)
	require.Nil(t, owner)

	now := time.Now().UTC()
	_, err = conn.ExecContext(ctx,
		`INSERT INTO components (id, display_name, owner_pod, last_updated, metadata)
		 VALUES (?, ?, ?, ?, ?)`,
		"auth-module", "Auth Module", "solo-pod", now, `{}`,
	)
	require.NoError(t, err)

	owner, err = svc.WhoOwns(ctx, "auth-module")
	require.NoError(t, err)
	require.NotNil(t, owner)
	require.Equal(t, "solo-pod", owner.PodID)
	require.Equal(t, "solo-pod", owner.DisplayName)
}

func TestQueryActiveWorkComponentFilter(t *testing.T) {
	svc, conn, ctx := newTestService(t)

	_, err := svc.RegisterPod(ctx, core.RegisterPodInput{Name: "solo-pod", Owner: "me@example.com"})
	require.NoError(t, err)

	ws1, err := svc.AddWorkstream(ctx, core.AddWorkstreamInput{
		PodID: "solo-pod", Title: "Auth", Branch: "feat/auth",
	})
	require.NoError(t, err)
	_, err = svc.AddWorkstream(ctx, core.AddWorkstreamInput{
		PodID: "solo-pod", Title: "Billing", Branch: "feat/billing",
	})
	require.NoError(t, err)

	_, err = conn.ExecContext(ctx,
		`UPDATE workstreams SET components = ? WHERE id = ?`,
		[]byte(`["auth-module"]`), ws1.ID,
	)
	require.NoError(t, err)

	component := "auth-module"
	results, err := svc.QueryActiveWork(ctx, core.WorkFilters{Component: &component})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, "Auth", results[0].Title)
}

func TestListGapsPriorityMaxFilter(t *testing.T) {
	svc, _, ctx := newTestService(t)

	_, err := svc.RegisterPod(ctx, core.RegisterPodInput{Name: "solo-pod", Owner: "me@example.com"})
	require.NoError(t, err)

	_, err = svc.AddGap(ctx, core.AddGapInput{
		PodID: "solo-pod", Category: core.GapOther, Description: "high priority", Priority: 2,
	})
	require.NoError(t, err)
	_, err = svc.AddGap(ctx, core.AddGapInput{
		PodID: "solo-pod", Category: core.GapOther, Description: "low priority", Priority: 4,
	})
	require.NoError(t, err)

	max := 2
	gaps, err := svc.ListCapabilityGaps(ctx, core.GapFilters{PriorityMax: &max})
	require.NoError(t, err)
	require.Len(t, gaps, 1)
	require.Equal(t, 2, gaps[0].Priority)
}

func TestGapDedupOpenOnly(t *testing.T) {
	svc, _, ctx := newTestService(t)

	_, err := svc.RegisterPod(ctx, core.RegisterPodInput{Name: "solo-pod", Owner: "me@example.com"})
	require.NoError(t, err)

	desc := "Same description after resolve"
	g1, err := svc.RegisterCapabilityGap(ctx, core.RegisterCapabilityGapInput{
		PodID: "solo-pod", Category: core.GapAmbiguousSpec, Description: desc, Priority: 2,
	})
	require.NoError(t, err)

	_, err = svc.ResolveGap(ctx, g1.ID, "decided")
	require.NoError(t, err)

	g2, err := svc.RegisterCapabilityGap(ctx, core.RegisterCapabilityGapInput{
		PodID: "solo-pod", Category: core.GapAmbiguousSpec, Description: desc, Priority: 2,
	})
	require.NoError(t, err)
	require.NotEqual(t, g1.ID, g2.ID)
	require.Equal(t, 1, g2.Frequency)
}

func TestGapFrequencyIncrementedEvent(t *testing.T) {
	svc, conn, ctx := newTestService(t)

	_, err := svc.RegisterPod(ctx, core.RegisterPodInput{Name: "solo-pod", Owner: "me@example.com"})
	require.NoError(t, err)

	desc := "Repeated gap"
	_, err = svc.RegisterCapabilityGap(ctx, core.RegisterCapabilityGapInput{
		PodID: "solo-pod", Category: core.GapOther, Description: desc, Priority: 3,
	})
	require.NoError(t, err)
	_, err = svc.RegisterCapabilityGap(ctx, core.RegisterCapabilityGapInput{
		PodID: "solo-pod", Category: core.GapOther, Description: desc, Priority: 3,
	})
	require.NoError(t, err)

	var eventType string
	err = conn.QueryRowContext(ctx,
		`SELECT event_type FROM events WHERE event_type = 'gap.frequency_incremented' LIMIT 1`,
	).Scan(&eventType)
	require.NoError(t, err)
	require.Equal(t, "gap.frequency_incremented", eventType)
}

func TestAcknowledgeGap(t *testing.T) {
	svc, _, ctx := newTestService(t)

	_, err := svc.RegisterPod(ctx, core.RegisterPodInput{Name: "solo-pod", Owner: "me@example.com"})
	require.NoError(t, err)

	gap, err := svc.AddGap(ctx, core.AddGapInput{
		PodID: "solo-pod", Category: core.GapOther, Description: "needs ack", Priority: 3,
	})
	require.NoError(t, err)

	gap, err = svc.AcknowledgeGap(ctx, gap.ID)
	require.NoError(t, err)
	require.Equal(t, core.GapAcknowledged, gap.Status)

	_, err = svc.AcknowledgeGap(ctx, gap.ID)
	require.Error(t, err)
}

func TestResolveGap(t *testing.T) {
	svc, _, ctx := newTestService(t)

	_, err := svc.RegisterPod(ctx, core.RegisterPodInput{Name: "solo-pod", Owner: "me@example.com"})
	require.NoError(t, err)

	gap, err := svc.AddGap(ctx, core.AddGapInput{
		PodID: "solo-pod", Category: core.GapOther, Description: "needs resolve", Priority: 3,
	})
	require.NoError(t, err)

	gap, err = svc.ResolveGap(ctx, gap.ID, "decided: JWT")
	require.NoError(t, err)
	require.Equal(t, core.GapResolved, gap.Status)
	require.Equal(t, "decided: JWT", gap.ResolutionRef)
	require.NotNil(t, gap.ResolvedAt)

	_, err = svc.ResolveGap(ctx, gap.ID, "")
	require.Error(t, err)
}

func TestPublishWorkstreamStateCreateAndUpdate(t *testing.T) {
	svc, _, ctx := newTestService(t)

	_, err := svc.RegisterPod(ctx, core.RegisterPodInput{Name: "solo-pod", Owner: "me@example.com"})
	require.NoError(t, err)

	ws, err := svc.PublishWorkstreamState(ctx, core.PublishWorkstreamStateInput{
		PodID:  "solo-pod",
		Title:  "Auth refactor",
		Intent: "Migrate to OIDC",
		Branch: "feat/oidc",
	})
	require.NoError(t, err)
	require.Equal(t, core.WorkstreamActive, ws.Status)
	require.Equal(t, "Auth refactor", ws.Title)

	ws, err = svc.PublishWorkstreamState(ctx, core.PublishWorkstreamStateInput{
		PodID:  "solo-pod",
		ID:     ws.ID,
		Status: core.WorkstreamBlocked,
		Intent: "Waiting on session token decision",
	})
	require.NoError(t, err)
	require.Equal(t, core.WorkstreamBlocked, ws.Status)
	require.Equal(t, "Auth refactor", ws.Title)
	require.Equal(t, "Waiting on session token decision", ws.Intent)
}
