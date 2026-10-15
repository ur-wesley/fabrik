package memory

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ur-wesley/fabrik/cli/internal/clierr"
)

func TestSessionStartCreatesAndReuses(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("FABRIK_MEMORY_DIR", dir)
	ctx := context.Background()

	r1, err := SessionStart(ctx, "")
	require.NoError(t, err)
	assert.True(t, r1.Created)
	assert.True(t, len(r1.ID) > 2)
	assert.Equal(t, "s-", r1.ID[:2])

	r2, err := SessionStart(ctx, "")
	require.NoError(t, err)
	assert.False(t, r2.Created)
	assert.Equal(t, r1.ID, r2.ID)
}

func TestSessionSummaryRequiresOpenSession(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("FABRIK_MEMORY_DIR", dir)
	_, err := SessionSummary(context.Background(), SessionSummaryOptions{GoalSet: true, Goal: "x"})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNoOpenSession)
}

func TestSessionSummaryRequiresFlag(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("FABRIK_MEMORY_DIR", dir)
	ctx := context.Background()
	_, err := SessionStart(ctx, "")
	require.NoError(t, err)
	_, err = SessionSummary(ctx, SessionSummaryOptions{})
	require.Error(t, err)
	var u *clierr.UsageError
	assert.ErrorAs(t, err, &u)
}

func TestSessionSummaryAndEnd(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("FABRIK_MEMORY_DIR", dir)
	ctx := context.Background()
	_, err := SessionStart(ctx, "")
	require.NoError(t, err)

	s, err := SessionSummary(ctx, SessionSummaryOptions{GoalSet: true, Goal: "Ship mem", NextStepsSet: true, NextSteps: "MCP"})
	require.NoError(t, err)
	assert.Equal(t, "Ship mem", s.Goal)

	obs, err := Search(ctx, SearchOptions{Query: "Ship"})
	require.NoError(t, err)
	require.Len(t, obs, 1)
	assert.Equal(t, "handoff", obs[0].Kind)

	ended, err := SessionEnd(ctx, "")
	require.NoError(t, err)
	require.NotNil(t, ended.EndedAt)

	_, err = SessionEnd(ctx, "")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNoOpenSession)
}
