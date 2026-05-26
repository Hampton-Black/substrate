package server

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRelativeTime(t *testing.T) {
	now := time.Now().UTC()

	require.Equal(t, "just now", relativeTime(now.Add(-30*time.Second)))
	require.Equal(t, "5m ago", relativeTime(now.Add(-5*time.Minute)))
	require.Equal(t, "2h ago", relativeTime(now.Add(-2*time.Hour)))
	require.Equal(t, "3 days ago", relativeTime(now.Add(-72*time.Hour)))
}
