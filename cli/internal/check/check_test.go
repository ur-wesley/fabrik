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
	"github.com/ur-wesley/fabrik/cli/internal/doltstore"
	"github.com/ur-wesley/fabrik/cli/internal/plugin"
	"github.com/ur-wesley/fabrik/cli/internal/sqlmigrate"
	"github.com/ur-wesley/fabrik/cli/internal/store"
)

func TestRunDoltMissing(t *testing.T) {
	repo := t.TempDir()
	r, code := Run(context.Background(), repo)
	assert.Equal(t, 1, code)
	assert.Equal(t, "missing", r.IssuesStore)
	assert.Equal(t, "missing", r.Dolt)
	assert.Equal(t, "missing", r.Graph)
	assert.Equal(t, 0, r.Issues)
}

func TestRunGraphMissingDoltPresent(t *testing.T) {
	repo := t.TempDir()
	st, err := doltstore.Open(context.Background(), repo)
	require.NoError(t, err)
	require.NoError(t, st.Close())
	r, code := Run(context.Background(), repo)
	assert.Equal(t, "ok", r.IssuesStore)
	assert.Equal(t, "ok", r.Dolt)
	assert.Equal(t, 0, code)
	assert.Equal(t, "missing", r.Graph)
}

func TestPrintJSONShape(t *testing.T) {
	repo := t.TempDir()
	r, _ := Run(context.Background(), repo)
	var buf bytes.Buffer
	require.NoError(t, PrintJSONTo(&buf, r))
	var m map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &m))
	for _, k := range []string{"dolt", "dolt_path", "graph", "graph_path", "schema_dolt", "schema_graph", "issues", "graph_files", "issues_store", "issues_path", "schema_issues"} {
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
	assert.Equal(t, "missing", r.IssuesStore)
	assert.Equal(t, "missing", r.Dolt)
}

func TestPrintPluginAdvisory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	var missing bytes.Buffer
	PrintPluginAdvisory(&missing)
	assert.Contains(t, missing.String(), "warning: antigravity plugin fabrik is not installed")

	require.NoError(t, plugin.Sync(home, false, &bytes.Buffer{}))
	var current bytes.Buffer
	PrintPluginAdvisory(&current)
	assert.Empty(t, current.String(), "advisory must stay silent when the plugin is current")
}
