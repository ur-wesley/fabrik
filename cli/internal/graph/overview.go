package graph

import (
	"context"
	"sort"
	"strings"
)

// OverviewDefaultBudget is the node cap used when the caller passes no
// positive budget. It matches the GUI auto-load so Lookup opens with a
// useful instant graph without extra tuning.
const OverviewDefaultBudget = 60

// OverviewSeedLimit caps the degree-ranked seeds; the remaining budget is
// filled by BFS expansion so the returned nodes arrive with connecting
// edges instead of a disconnected induced subgraph.
const OverviewSeedLimit = 15

// Overview ranks symbol seeds by edge degree (COUNT over graph_edges
// grouped by endpoint, ORDER BY degree DESC, top OverviewSeedLimit) then
// expands via bfsNeighborhood within budget so nodes arrive with the
// traversed edges plus the CONTAINS backbone. Same QueryResult shape as
// Query so callers reuse PrintQueryJSON and the GUI query-loaded
// rendering. Seeds lead in degree order; expansion remainder follows by
// ID. Callers must not re-sort.
func (d *DB) Overview(ctx context.Context, budget int) (QueryResult, error) {
	if budget <= 0 {
		budget = OverviewDefaultBudget
	}
	if err := d.requireIndexed(ctx); err != nil {
		return QueryResult{}, err
	}
	if err := d.RefreshStale(ctx); err != nil {
		return QueryResult{}, err
	}
	seedLimit := budget
	if seedLimit > OverviewSeedLimit {
		seedLimit = OverviewSeedLimit
	}
	rows, err := d.sql.QueryContext(ctx, `
		SELECT n.id, COUNT(*) AS degree
		FROM (
			SELECT src AS endpoint FROM graph_edges WHERE src != ''
			UNION ALL
			SELECT dst AS endpoint FROM graph_edges WHERE dst != ''
		) AS deg
		JOIN graph_nodes n ON n.id = deg.endpoint AND n.kind = 'symbol'
		GROUP BY n.id
		ORDER BY degree DESC, n.id ASC
		LIMIT ?`, seedLimit)
	if err != nil {
		return QueryResult{}, err
	}
	var ids []string
	defer rows.Close()
	for rows.Next() {
		var id string
		var degree int
		if err := rows.Scan(&id, &degree); err != nil {
			return QueryResult{}, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return QueryResult{}, err
	}
	if len(ids) == 0 {
		return QueryResult{Nodes: []Node{}, Edges: []Edge{}}, nil
	}
	// Seed+expand: BFS from the ranked seeds fills the budget with
	// neighbors, keeping the traversed edges plus the CONTAINS backbone
	// (file nodes join via contains edges) instead of filtering to a
	// both-ends-in-set induced subgraph, which stranded high-degree hubs
	// with zero edges.
	res, err := d.bfsNeighborhood(ctx, ids, budget)
	if err != nil {
		return QueryResult{}, err
	}
	// Seeds lead in degree order; expansion remainder follows by ID so
	// the top symbols stay first without re-sorting the whole set.
	byID := make(map[string]Node, len(res.Nodes))
	for _, n := range res.Nodes {
		byID[n.ID] = n
	}
	ordered := make([]Node, 0, len(res.Nodes))
	for _, id := range ids {
		if n, ok := byID[id]; ok {
			ordered = append(ordered, n)
			delete(byID, id)
		}
	}
	rest := make([]Node, 0, len(byID))
	for _, n := range byID {
		rest = append(rest, n)
	}
	sort.Slice(rest, func(i, j int) bool { return rest[i].ID < rest[j].ID })
	res.Nodes = append(ordered, rest...)
	return res, nil
}

// edgesAmong returns deduplicated edges whose endpoints both sit in ids
// (unresolved dst '' edges count when their src is chosen), mirroring the
// bfsNeighborhood membership rule. Sorted by confidence then endpoint so
// output is deterministic across runs.
func (d *DB) edgesAmong(ctx context.Context, ids []string) ([]Edge, error) {
	inSet := make(map[string]bool, len(ids))
	for _, id := range ids {
		inSet[id] = true
	}
	placeholders := strings.Repeat("?,", len(ids))
	placeholders = strings.TrimSuffix(placeholders, ",")
	args := make([]any, 0, 2*len(ids))
	for _, id := range ids {
		args = append(args, id)
	}
	for _, id := range ids {
		args = append(args, id)
	}
	rows, err := d.sql.QueryContext(ctx, `
		SELECT src, dst, kind, label, confidence FROM graph_edges
		WHERE src IN (`+placeholders+`) AND (dst = '' OR dst IN (`+placeholders+`))`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Edge
	seen := map[string]bool{}
	for rows.Next() {
		var e Edge
		if err := rows.Scan(&e.Src, &e.Dst, &e.Kind, &e.Label, &e.Confidence); err != nil {
			return nil, err
		}
		if !inSet[e.Src] || (e.Dst != "" && !inSet[e.Dst]) {
			continue
		}
		key := e.Src + "|" + e.Dst + "|" + e.Kind + "|" + e.Label
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool {
		ri := edgeConfidenceRank(out[i].Confidence)
		rj := edgeConfidenceRank(out[j].Confidence)
		if ri != rj {
			return ri < rj
		}
		if out[i].Src != out[j].Src {
			return out[i].Src < out[j].Src
		}
		if out[i].Dst != out[j].Dst {
			return out[i].Dst < out[j].Dst
		}
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		return out[i].Label < out[j].Label
	})
	return out, nil
}
