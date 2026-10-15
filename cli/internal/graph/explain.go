package graph

import (
	"context"
	"sort"
)

func (d *DB) Explain(ctx context.Context, id string) (ExplainResult, error) {
	if err := d.requireIndexed(ctx); err != nil {
		return ExplainResult{}, err
	}
	node, err := d.loadNode(ctx, id)
	if err != nil {
		return ExplainResult{}, err
	}
	rows, err := d.sql.QueryContext(ctx, `
		SELECT src, dst, kind, label, confidence FROM graph_edges WHERE src = ? OR dst = ?`, id, id)
	if err != nil {
		return ExplainResult{}, err
	}
	defer rows.Close()
	var edges []Edge
	for rows.Next() {
		var e Edge
		if err := rows.Scan(&e.Src, &e.Dst, &e.Kind, &e.Label, &e.Confidence); err != nil {
			return ExplainResult{}, err
		}
		edges = append(edges, e)
	}
	if err := rows.Err(); err != nil {
		return ExplainResult{}, err
	}
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].Kind != edges[j].Kind {
			return edgeKindOrder(edges[i].Kind) < edgeKindOrder(edges[j].Kind)
		}
		otherI := otherEnd(edges[i], id)
		otherJ := otherEnd(edges[j], id)
		return otherI < otherJ
	})
	truncated := len(edges) > 100
	if truncated {
		edges = edges[:100]
	}
	return ExplainResult{Node: node, Edges: edges, Truncated: truncated}, nil
}

func edgeKindOrder(k string) int {
	switch k {
	case "contains":
		return 0
	case "imports":
		return 1
	case "calls":
		return 2
	default:
		return 3
	}
}

func otherEnd(e Edge, id string) string {
	if e.Src == id {
		if e.Dst != "" {
			return e.Dst
		}
		return e.Label
	}
	return e.Src
}
