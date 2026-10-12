package memory

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ur-wesley/fabrik/cli/internal/clierr"
)

func TestSearchFindsObservation(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("FABRIK_MEMORY_DIR", dir)
	ctx := context.Background()
	_, err := Save(ctx, SaveOptions{Title: "Issue ids", Body: "Hash prefix o-", Kind: "decision"})
	require.NoError(t, err)

	obs, err := Search(ctx, SearchOptions{Query: "issue ids"})
	require.NoError(t, err)
	require.Len(t, obs, 1)
	assert.Equal(t, "Issue ids", obs[0].Title)
}

func TestSearchRejectsQuotes(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("FABRIK_MEMORY_DIR", dir)
	_, err := Search(context.Background(), SearchOptions{Query: `"foo"`})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `"`)
}

func TestSearchEmptyQuery(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("FABRIK_MEMORY_DIR", dir)
	_, err := Search(context.Background(), SearchOptions{Query: "  "})
	require.Error(t, err)
	var u *clierr.UsageError
	assert.ErrorAs(t, err, &u)
}

func TestSearchUnknownProject(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("FABRIK_MEMORY_DIR", dir)
	ctx := context.Background()
	_, err := Save(ctx, SaveOptions{Title: "t", Body: "b"})
	require.NoError(t, err)
	_, err = Search(ctx, SearchOptions{Query: "t", Project: filepath.Join(dir, "missing")})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrProjectNotFound)
}
