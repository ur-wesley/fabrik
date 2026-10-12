package graph

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ur-wesley/fabrik/cli/internal/sqlmigrate"
	"github.com/ur-wesley/fabrik/cli/internal/store"
)

func setupGraphRepo(t *testing.T) string {
	dir := t.TempDir()
	src := filepath.Join("testdata", "callgraph")
	require.NoError(t, copyDir(src, dir))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".fabrik", "dolt"), 0o755))
	_, err := sqlmigrate.ApplySQLite(context.Background(), "graph", store.GraphDBPath(dir))
	require.NoError(t, err)
	return dir
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}

func TestBuildCallgraph(t *testing.T) {
	dir := setupGraphRepo(t)
	db, err := Open(context.Background(), dir)
	require.NoError(t, err)
	defer db.Close()

	r, err := db.Build(context.Background())
	require.NoError(t, err)
	assert.GreaterOrEqual(t, r.Parsed, 2)

	var helperID string
	err = db.sql.QueryRow(`SELECT id FROM graph_nodes WHERE name = 'Helper'`).Scan(&helperID)
	require.NoError(t, err)
	assert.Equal(t, "sym:pkg/lib/lib.go:Helper", helperID)

	var extracted int
	err = db.sql.QueryRow(`
		SELECT COUNT(*) FROM graph_edges
		WHERE kind = 'calls' AND confidence = 'extracted' AND dst = ?`, helperID).Scan(&extracted)
	require.NoError(t, err)
	assert.Equal(t, 1, extracted)

	var inferred int
	err = db.sql.QueryRow(`
		SELECT COUNT(*) FROM graph_edges
		WHERE kind = 'calls' AND confidence = 'inferred' AND label = 'Other'`).Scan(&inferred)
	require.NoError(t, err)
	assert.Equal(t, 1, inferred)
}

func TestBuildJavaCallgraph(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join("testdata", "jvm")
	require.NoError(t, copyDir(src, dir))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".fabrik", "dolt"), 0o755))
	_, err := sqlmigrate.ApplySQLite(context.Background(), "graph", store.GraphDBPath(dir))
	require.NoError(t, err)
	db, err := Open(context.Background(), dir)
	require.NoError(t, err)
	defer db.Close()
	_, err = db.Build(context.Background())
	require.NoError(t, err)
	var helperID string
	err = db.sql.QueryRow(`SELECT id FROM graph_nodes WHERE name = 'helper'`).Scan(&helperID)
	require.NoError(t, err)
	var extracted int
	err = db.sql.QueryRow(`
		SELECT COUNT(*) FROM graph_edges
		WHERE kind = 'calls' AND confidence = 'extracted' AND dst = ?`, helperID).Scan(&extracted)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, extracted, 1)
}

func TestBuildCSharpCallgraph(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join("testdata", "csharp")
	require.NoError(t, copyDir(src, dir))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".fabrik", "dolt"), 0o755))
	_, err := sqlmigrate.ApplySQLite(context.Background(), "graph", store.GraphDBPath(dir))
	require.NoError(t, err)
	db, err := Open(context.Background(), dir)
	require.NoError(t, err)
	defer db.Close()
	_, err = db.Build(context.Background())
	require.NoError(t, err)
	var helperID string
	err = db.sql.QueryRow(`SELECT id FROM graph_nodes WHERE name = 'Helper'`).Scan(&helperID)
	require.NoError(t, err)
	var extracted int
	err = db.sql.QueryRow(`
		SELECT COUNT(*) FROM graph_edges
		WHERE kind = 'calls' AND confidence = 'extracted' AND dst = ?`, helperID).Scan(&extracted)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, extracted, 1)
}

func TestExploreNotIndexedJSON(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".fabrik", "dolt"), 0o755))
	_, err := sqlmigrate.ApplySQLite(context.Background(), "graph", store.GraphDBPath(dir))
	require.NoError(t, err)
	db, err := Open(context.Background(), dir)
	require.NoError(t, err)
	defer db.Close()
	r, err := db.Explore(context.Background(), ExploreOpts{Query: "Run", Budget: 40})
	require.NoError(t, err)
	assert.False(t, r.Indexed)
	assert.Contains(t, r.Hint, "fabrik graph build")
}
