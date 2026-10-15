package graph

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ur-wesley/fabrik/cli/internal/sqlmigrate"
	"github.com/ur-wesley/fabrik/cli/internal/store"
)

func setupBareGraphRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".fabrik", "dolt"), 0o755))
	_, err := sqlmigrate.ApplySQLite(context.Background(), "graph", store.GraphDBPath(dir))
	require.NoError(t, err)
	return dir
}

func seedOverviewRepo(t *testing.T) *DB {
	t.Helper()
	// Bare repo (no source files) so RefreshStale inside Overview is a
	// no-op and degrees stay exactly as seeded below.
	dir := setupBareGraphRepo(t)
	db, err := Open(context.Background(), dir)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	ctx := context.Background()
	_, err = db.sql.ExecContext(ctx, `INSERT INTO graph_files (path, mtime, lang, ok) VALUES ('a.go', '2026-10-14T00:00:00Z', 'go', 1)`)
	require.NoError(t, err)
	nodes := [][6]string{
		{"file:a.go", "file", "a.go", "a.go", "0", "go"},
		{"sym:a.go:Alpha", "symbol", "Alpha", "a.go", "1", "go"},
		{"sym:a.go:Beta", "symbol", "Beta", "a.go", "2", "go"},
		{"sym:a.go:Gamma", "symbol", "Gamma", "a.go", "3", "go"},
		{"sym:a.go:Delta", "symbol", "Delta", "a.go", "4", "go"},
	}
	for _, n := range nodes {
		_, err = db.sql.ExecContext(ctx, `INSERT INTO graph_nodes (id, kind, name, path, line, lang) VALUES (?, ?, ?, ?, ?, ?)`,
			n[0], n[1], n[2], n[3], n[4], n[5])
		require.NoError(t, err)
	}
	// Degrees: Alpha 4 (3 calls + 1 contains), Beta/Gamma/Delta 2 each
	// (1 calls + 1 contains), file:a.go 4 (contains x4, must be filtered).
	edges := [][5]string{
		{"sym:a.go:Alpha", "sym:a.go:Beta", "calls", "", "extracted"},
		{"sym:a.go:Alpha", "sym:a.go:Gamma", "calls", "", "extracted"},
		{"sym:a.go:Alpha", "sym:a.go:Delta", "calls", "", "extracted"},
		{"file:a.go", "sym:a.go:Alpha", "contains", "", "extracted"},
		{"file:a.go", "sym:a.go:Beta", "contains", "", "extracted"},
		{"file:a.go", "sym:a.go:Gamma", "contains", "", "extracted"},
		{"file:a.go", "sym:a.go:Delta", "contains", "", "extracted"},
	}
	for _, e := range edges {
		_, err = db.sql.ExecContext(ctx, `INSERT INTO graph_edges (src, dst, kind, label, confidence) VALUES (?, ?, ?, ?, ?)`,
			e[0], e[1], e[2], e[3], e[4])
		require.NoError(t, err)
	}
	return db
}

func TestOverviewDegreeOrder(t *testing.T) {
	db := seedOverviewRepo(t)
	r, err := db.Overview(context.Background(), 10)
	require.NoError(t, err)
	// Seed+expand: the 4 ranked seeds lead in degree order, then the
	// CONTAINS backbone (file:a.go joins via contains edges).
	require.Len(t, r.Nodes, 5)
	assert.Equal(t, "sym:a.go:Alpha", r.Nodes[0].ID)
	// Ties at degree 2 break by id ASC: Beta, Delta, Gamma.
	assert.Equal(t, "sym:a.go:Beta", r.Nodes[1].ID)
	assert.Equal(t, "sym:a.go:Delta", r.Nodes[2].ID)
	assert.Equal(t, "sym:a.go:Gamma", r.Nodes[3].ID)
	assert.Equal(t, "file:a.go", r.Nodes[4].ID)
	// 3 calls edges plus the 4 contains backbone edges.
	require.Len(t, r.Edges, 7)
	inSet := map[string]bool{}
	for _, n := range r.Nodes {
		inSet[n.ID] = true
	}
	for _, e := range r.Edges {
		assert.True(t, inSet[e.Src], "edge src %s not in set", e.Src)
		assert.True(t, e.Dst == "" || inSet[e.Dst], "edge dst %s not in set", e.Dst)
	}
	assertIncident(t, r)
}

func TestOverviewBudgetCap(t *testing.T) {
	db := seedOverviewRepo(t)
	r, err := db.Overview(context.Background(), 2)
	require.NoError(t, err)
	require.Len(t, r.Nodes, 2)
	assert.Equal(t, "sym:a.go:Alpha", r.Nodes[0].ID)
	assert.Equal(t, "sym:a.go:Beta", r.Nodes[1].ID)
	// Only Alpha->Beta survives: both endpoints must be chosen.
	require.Len(t, r.Edges, 1)
	assert.Equal(t, "sym:a.go:Alpha", r.Edges[0].Src)
	assert.Equal(t, "sym:a.go:Beta", r.Edges[0].Dst)
}

func TestOverviewDefaultBudget(t *testing.T) {
	db := seedOverviewRepo(t)
	r, err := db.Overview(context.Background(), 0)
	require.NoError(t, err)
	assert.Len(t, r.Nodes, 5)
	require.NotEmpty(t, r.Edges)
}

func TestOverviewEmptyGraph(t *testing.T) {
	dir := setupBareGraphRepo(t)
	db, err := Open(context.Background(), dir)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Overview(context.Background(), 10)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrEmpty))
}

// seedContainsOnlyRepo builds n symbols joined only by CONTAINS backbone
// edges (no calls): the old top-N induced subgraph returned 0 edges here
// because the file node was never chosen.
func seedContainsOnlyRepo(t *testing.T, n int) *DB {
	t.Helper()
	dir := setupBareGraphRepo(t)
	db, err := Open(context.Background(), dir)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	ctx := context.Background()
	_, err = db.sql.ExecContext(ctx, `INSERT INTO graph_files (path, mtime, lang, ok) VALUES ('b.go', '2026-10-14T00:00:00Z', 'go', 1)`)
	require.NoError(t, err)
	_, err = db.sql.ExecContext(ctx, `INSERT INTO graph_nodes (id, kind, name, path, line, lang) VALUES ('file:b.go', 'file', 'b.go', 'b.go', 0, 'go')`)
	require.NoError(t, err)
	for i := 0; i < n; i++ {
		id := "sym:b.go:Sym" + itoa(i)
		_, err = db.sql.ExecContext(ctx, `INSERT INTO graph_nodes (id, kind, name, path, line, lang) VALUES (?, 'symbol', ?, 'b.go', ?, 'go')`,
			id, "Sym"+itoa(i), i+1)
		require.NoError(t, err)
		_, err = db.sql.ExecContext(ctx, `INSERT INTO graph_edges (src, dst, kind, label, confidence) VALUES ('file:b.go', ?, 'contains', '', 'extracted')`, id)
		require.NoError(t, err)
	}
	return db
}

func itoa(i int) string {
	return fmt.Sprintf("%02d", i)
}

func TestOverviewDisconnectedHubsHaveEdges(t *testing.T) {
	db := seedContainsOnlyRepo(t, 20)
	r, err := db.Overview(context.Background(), 60)
	require.NoError(t, err)
	// 15 ranked seeds expand through the file backbone to every
	// reachable node: 15 seeds + file + 5 remaining symbols.
	require.Len(t, r.Nodes, 21)
	// Every symbol reaches the canvas through its contains edge.
	require.Len(t, r.Edges, 20)
	for _, e := range r.Edges {
		assert.Equal(t, "contains", e.Kind)
	}
	assertIncident(t, r)
}

// seedChainRepo builds n padded symbols joined by a calls chain plus
// CONTAINS backbone edges, so degree ranking and id order agree.
func seedChainRepo(t *testing.T, n int) *DB {
	t.Helper()
	dir := setupBareGraphRepo(t)
	db, err := Open(context.Background(), dir)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	ctx := context.Background()
	_, err = db.sql.ExecContext(ctx, `INSERT INTO graph_files (path, mtime, lang, ok) VALUES ('c.go', '2026-10-14T00:00:00Z', 'go', 1)`)
	require.NoError(t, err)
	_, err = db.sql.ExecContext(ctx, `INSERT INTO graph_nodes (id, kind, name, path, line, lang) VALUES ('file:c.go', 'file', 'c.go', 'c.go', 0, 'go')`)
	require.NoError(t, err)
	for i := 0; i < n; i++ {
		id := "sym:c.go:Sym" + itoa(i)
		_, err = db.sql.ExecContext(ctx, `INSERT INTO graph_nodes (id, kind, name, path, line, lang) VALUES (?, 'symbol', ?, 'c.go', ?, 'go')`,
			id, "Sym"+itoa(i), i+1)
		require.NoError(t, err)
		_, err = db.sql.ExecContext(ctx, `INSERT INTO graph_edges (src, dst, kind, label, confidence) VALUES ('file:c.go', ?, 'contains', '', 'extracted')`, id)
		require.NoError(t, err)
		if i+1 < n {
			next := "sym:c.go:Sym" + itoa(i+1)
			_, err = db.sql.ExecContext(ctx, `INSERT INTO graph_edges (src, dst, kind, label, confidence) VALUES (?, ?, 'calls', '', 'extracted')`, id, next)
			require.NoError(t, err)
		}
	}
	return db
}

func TestOverviewRespectsBudget(t *testing.T) {
	db := seedChainRepo(t, 40)
	r, err := db.Overview(context.Background(), 10)
	require.NoError(t, err)
	// Seeds already fill the budget, so no expansion: the 10 interior
	// chain symbols (degree 3) plus the 9 chain edges among them.
	require.Len(t, r.Nodes, 10)
	require.Len(t, r.Edges, 9)
	assertIncident(t, r)
}

// assertIncident fails when any returned node is orphaned (no edge
// touches it): the overview must be connected, never nodes without
// edges on a non-trivial graph.
func assertIncident(t *testing.T, r QueryResult) {
	t.Helper()
	incident := map[string]bool{}
	for _, e := range r.Edges {
		if e.Src != "" {
			incident[e.Src] = true
		}
		if e.Dst != "" {
			incident[e.Dst] = true
		}
	}
	for _, n := range r.Nodes {
		assert.True(t, incident[n.ID], "node %s has no incident edge", n.ID)
	}
}
