package memory

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ur-wesley/fabrik/cli/internal/sqlmigrate"
)

func TestCheckMissing(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("FABRIK_MEMORY_DIR", dir)
	r, err := Check(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "missing", r.Memory)
}

func TestCheckOK(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("FABRIK_MEMORY_DIR", dir)
	dbPath := filepath.Join(dir, "memory.db")
	_, err := sqlmigrate.ApplySQLite(context.Background(), "memory", dbPath)
	require.NoError(t, err)

	r, err := Check(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "ok", r.Memory)
	assert.Equal(t, dbPath, r.MemoryPath)
	assert.Equal(t, 2, r.SchemaMemory)
}
