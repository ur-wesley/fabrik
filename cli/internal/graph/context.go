package graph

import (
	"context"
	"fmt"
	"strings"

	"github.com/ur-wesley/fabrik/cli/internal/memory"
)

func (d *DB) Context(ctx context.Context, symbol string) (ContextResult, error) {
	if err := d.requireIndexed(ctx); err != nil {
		return ContextResult{}, err
	}
	sid, err := d.resolveSymbol(ctx, symbol)
	if err != nil {
		return ContextResult{}, err
	}
	node, err := d.loadNode(ctx, sid)
	if err != nil {
		return ContextResult{}, err
	}
	if err := d.RefreshStale(ctx); err != nil {
		return ContextResult{}, err
	}
	ex, err := d.Explain(ctx, sid)
	if err != nil {
		return ContextResult{}, err
	}
	res := ContextResult{Symbol: node}
	for _, e := range ex.Edges {
		if e.Kind == "calls" && e.Src == sid {
			res.Callees = append(res.Callees, e)
		}
		if e.Kind == "calls" && e.Dst == sid {
			if n, err := d.loadNode(ctx, e.Src); err == nil {
				res.Callers = append(res.Callers, n)
			}
		}
	}
	res.Observations = loadObservations(ctx, d.Repo, sid)
	res.Tests = d.findRelatedTests(ctx, node.Name)
	return res, nil
}

func (d *DB) resolveSymbol(ctx context.Context, symbol string) (string, error) {
	if strings.HasPrefix(symbol, "sym:") {
		if _, err := d.loadNode(ctx, symbol); err != nil {
			return "", err
		}
		return symbol, nil
	}
	rows, err := d.sql.QueryContext(ctx, `
		SELECT id FROM graph_nodes WHERE kind = 'symbol' AND name = ?`, symbol)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return "", err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	if len(ids) == 1 {
		return ids[0], nil
	}
	if len(ids) == 0 {
		return "", fmt.Errorf("fabrik: node sym:%s not found", symbol)
	}
	return "", fmt.Errorf("fabrik: ambiguous symbol %s (%d matches)", symbol, len(ids))
}

func loadObservations(ctx context.Context, project, symbolID string) []ContextObservation {
	refs, err := memory.ObservationsForSymbol(ctx, project, symbolID)
	if err != nil || len(refs) == 0 {
		return nil
	}
	out := make([]ContextObservation, len(refs))
	for i, r := range refs {
		out[i] = ContextObservation{ID: r.ID, Title: r.Title, Kind: r.Kind}
	}
	return out
}

func (d *DB) findRelatedTests(ctx context.Context, name string) []ContextTest {
	rows, err := d.sql.QueryContext(ctx, `
		SELECT path, name FROM graph_nodes
		WHERE kind = 'symbol'
		AND (path LIKE '%_test.go' OR path LIKE '%.test.ts' OR path LIKE '%.test.tsx')
		AND (instr(name, ?) > 0 OR instr(?, name) > 0)`, name, name)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []ContextTest
	for rows.Next() {
		var path, sym string
		if err := rows.Scan(&path, &sym); err != nil {
			return nil
		}
		out = append(out, ContextTest{Path: path, Symbol: sym})
	}
	if err := rows.Err(); err != nil {
		return nil
	}
	return out
}
