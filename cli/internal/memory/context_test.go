package memory

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContextEmpty(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("FABRIK_MEMORY_DIR", dir)
	r, err := Context(context.Background(), ContextOptions{})
	require.NoError(t, err)
	assert.Nil(t, r.Session)
	assert.Empty(t, r.Observations)
}

func TestContextAfterSave(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("FABRIK_MEMORY_DIR", dir)
	ctx := context.Background()
	o, err := Save(ctx, SaveOptions{Title: "t", Body: "b", Kind: "decision"})
	require.NoError(t, err)
	r, err := Context(ctx, ContextOptions{})
	require.NoError(t, err)
	require.Len(t, r.Observations, 1)
	assert.Equal(t, o.ID, r.Observations[0].ID)
	assert.Equal(t, 1, r.Stats.ObservationCount)
}

func TestContextUnknownProject(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("FABRIK_MEMORY_DIR", dir)
	ctx := context.Background()
	_, err := Save(ctx, SaveOptions{Title: "t", Body: "b"})
	require.NoError(t, err)
	_, err = Context(ctx, ContextOptions{Project: filepath.Join(dir, "nope")})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrProjectNotFound)
}
