package server_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Hampton-Black/substrate/internal/core"
	"github.com/Hampton-Black/substrate/internal/db"
	"github.com/Hampton-Black/substrate/internal/server"
	"github.com/stretchr/testify/require"
)

func newTestServer(t *testing.T) (*server.Server, *core.Service, context.Context) {
	t.Helper()
	ctx := context.Background()
	dir := t.TempDir()
	conn, err := db.Open(ctx, filepath.Join(dir, "test.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	svc := core.NewService(conn, nil, slog.New(slog.NewTextHandler(os.Stderr, nil)))
	srv, err := server.New(svc, nil)
	require.NoError(t, err)
	return srv, svc, ctx
}

func TestDashboardAndAPI(t *testing.T) {
	srv, svc, ctx := newTestServer(t)

	_, err := svc.RegisterPod(ctx, core.RegisterPodInput{Name: "solo-pod", Owner: "me@example.com"})
	require.NoError(t, err)

	ws, err := svc.AddWorkstream(ctx, core.AddWorkstreamInput{
		PodID: "solo-pod", Title: "Auth refactor", Intent: "Migrate to OIDC", Branch: "feat/oidc",
	})
	require.NoError(t, err)

	_, err = svc.SetWorkstreamStatus(ctx, ws.ID, core.WorkstreamBlocked)
	require.NoError(t, err)

	gap, err := svc.AddGap(ctx, core.AddGapInput{
		PodID: "solo-pod", Category: core.GapAmbiguousSpec,
		Description: "No spec for sessions", Priority: 2,
	})
	require.NoError(t, err)
	_, err = svc.ResolveGap(ctx, gap.ID, "decided")
	require.NoError(t, err)

	handler := srv.Handler()

	t.Run("index", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)
		body := rec.Body.String()
		require.Contains(t, body, "Substrate")
		require.Contains(t, body, "Auth refactor")
		require.Contains(t, body, "blocked")
		require.Contains(t, body, "No open gaps")
	})

	t.Run("api workstreams", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/workstreams?pod=solo-pod", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)

		var workstreams []core.Workstream
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &workstreams))
		require.Len(t, workstreams, 1)
		require.Equal(t, core.WorkstreamBlocked, workstreams[0].Status)
	})

	t.Run("api gaps default", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/gaps?pod=solo-pod", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)

		var gaps []core.CapabilityGap
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &gaps))
		require.Empty(t, gaps)
	})

	t.Run("api gaps open", func(t *testing.T) {
		_, err := svc.AddGap(ctx, core.AddGapInput{
			PodID: "solo-pod", Category: core.GapOther, Description: "open gap", Priority: 3,
		})
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/api/gaps?pod=solo-pod", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)

		var gaps []core.CapabilityGap
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &gaps))
		require.Len(t, gaps, 1)
		require.True(t, strings.Contains(gaps[0].Description, "open gap"))
	})
}
