package memory

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetObservation(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("FABRIK_MEMORY_DIR", dir)
	ctx := context.Background()
	saved, err := Save(ctx, SaveOptions{Title: "t", Body: "full body", Kind: "pattern"})
	require.NoError(t, err)

	got, err := Get(ctx, saved.ID)
	require.NoError(t, err)
	assert.Equal(t, saved.ID, got.ID)
	assert.Equal(t, "full body", got.Body)
}

func TestGetNotFound(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("FABRIK_MEMORY_DIR", dir)
	_, err := Get(context.Background(), "o-deadbeef")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}
