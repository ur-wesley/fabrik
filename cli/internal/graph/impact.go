package graph

import (
	"context"
	"fmt"
)

func (d *DB) Impact(ctx context.Context, symbol string, depth int) (ImpactResult, error) {
	if err := d.requireIndexed(ctx); err != nil {
		return ImpactResult{}, err
	}
	if depth <= 0 {
		depth = 3
	}
	sid, err := d.resolveSymbol(ctx, symbol)
	if err != nil {
		return ImpactResult{}, err
	}
	res := ImpactResult{Root: sid, ByDepth: map[string][]Node{}, Total: 0}
	frontier := []string{sid}
	seen := map[string]bool{sid: true}
	for dpt := 1; dpt <= depth; dpt++ {
		var next []string
		key := fmt.Sprintf("%d", dpt)
		res.ByDepth[key] = []Node{}
		for _, id := range frontier {
			rows, err := d.sql.QueryContext(ctx, `
				SELECT src, confidence FROM graph_edges
				WHERE dst = ? AND kind IN ('calls', 'imports')`, id)
			if err != nil {
				return ImpactResult{}, err
			}
			for rows.Next() {
				var src, conf string
				if err := rows.Scan(&src, &conf); err != nil {
					rows.Close()
					return ImpactResult{}, err
				}
				if seen[src] {
					continue
				}
				seen[src] = true
				n, err := d.loadNode(ctx, src)
				if err != nil {
					continue
				}
				res.ByDepth[key] = append(res.ByDepth[key], n)
				res.Total++
				next = append(next, src)
			}
			if err := rows.Err(); err != nil {
				rows.Close()
				return ImpactResult{}, err
			}
			rows.Close()
		}
		frontier = next
		if len(frontier) == 0 {
			break
		}
	}
	return res, nil
}
