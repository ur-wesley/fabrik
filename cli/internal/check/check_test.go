package check

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ur-wesley/fabrik/cli/internal/sqlmigrate"
	"github.com/ur-wesley/fabrik/cli/internal/store"
)

func TestRunDoltMissing(t *testing.T) {
	repo := t.TempDir()
	r, code := Run(context.Background(), repo)
	assert.Equal(t, 1, code)
	assert.Equal(t, "missing", r.Dolt)
	assert.Equal(t, "missing", r.Graph)
	assert.Equal(t, 0, r.Issues)
}

func TestRunGraphMissingDoltPresent(t *testing.T) {
	if os.Getenv("CGO_ENABLED") == "0" {
		t.Skip("requires CGO_ENABLED=1")
	}
	repo := t.TempDir()
	require.NoError(t, os.MkdirAll(store.DoltDir(repo), 0o755))
	r, code := Run(context.Background(), repo)
	if r.Dolt == "ok" {
		assert.Equal(t, 0, code)
		assert.Equal(t, "missing", r.Graph)
	}
}

func TestPrintJSONShape(t *testing.T) {
	repo := t.TempDir()
	r, _ := Run(context.Background(), repo)
	var buf bytes.Buffer
	require.NoError(t, PrintJSONTo(&buf, r))
	var m map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &m))
	for _, k := range []string{"dolt", "dolt_path", "graph", "graph_path", "schema_dolt", "schema_graph", "issues", "graph_files"} {
		assert.Contains(t, m, k)
	}
}

func TestRunGraphSQLiteOnly(t *testing.T) {
	repo := t.TempDir()
	graphPath := store.GraphDBPath(repo)
	require.NoError(t, os.MkdirAll(filepath.Dir(graphPath), 0o755))
	_, err := sqlmigrate.ApplySQLite(context.Background(), "graph", graphPath)
	require.NoError(t, err)
	r, code := Run(context.Background(), repo)
	assert.Equal(t, 1, code)
	assert.Equal(t, "missing", r.Dolt)
}
