package graph

import (
	"context"
	"sort"
	"strings"
	"unicode/utf8"
)

func edgeConfidenceRank(c string) int {
	if c == "extracted" {
		return 0
	}
	return 1
}

func (d *DB) Query(ctx context.Context, text string, budget int) (QueryResult, error) {
	return d.queryWithSeeds(ctx, text, budget, nil)
}

func (d *DB) queryWithSeeds(ctx context.Context, text string, budget int, extraSeeds []string) (QueryResult, error) {
	if err := d.requireIndexed(ctx); err != nil {
		return QueryResult{}, err
	}
	if err := d.RefreshStale(ctx); err != nil {
		return QueryResult{}, err
	}
	words := splitQueryWords(text)
	var seeds []string
	if len(words) > 0 {
		matched, err := d.matchNodes(ctx, words, budget)
		if err != nil {
			return QueryResult{}, err
		}
		seeds = matched
	}
	seeds = mergeSeedIDs(seeds, extraSeeds)
	if len(seeds) == 0 {
		return QueryResult{Nodes: []Node{}, Edges: []Edge{}}, nil
	}
	return d.bfsNeighborhood(ctx, seeds, budget)
}

func splitQueryWords(text string) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	if utf8.RuneCountInString(text) > 200 {
		text = string([]rune(text)[:200])
	}
	var words []string
	for _, w := range strings.Fields(text) {
		words = append(words, strings.ToLower(w))
	}
	return words
}

func (d *DB) matchNodes(ctx context.Context, words []string, budget int) ([]string, error) {
	rows, err := d.sql.QueryContext(ctx, `SELECT id, kind, name, path, line, lang FROM graph_nodes`)
	if err != nil {
		return nil, err
	}
	var matched []Node
	defer rows.Close()
	for rows.Next() {
		var n Node
		var kind string
		if err := rows.Scan(&n.ID, &kind, &n.Name, &n.Path, &n.Line, &n.Lang); err != nil {
			return nil, err
		}
		n.Kind = kind
		if nodeMatches(n, words) {
			matched = append(matched, n)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sort.Slice(matched, func(i, j int) bool {
		si := matched[i].Kind == "symbol"
		sj := matched[j].Kind == "symbol"
		if si != sj {
			return si
		}
		if len(matched[i].Name) != len(matched[j].Name) {
			return len(matched[i].Name) < len(matched[j].Name)
		}
		return matched[i].Path < matched[j].Path
	})
	var ids []string
	for i, n := range matched {
		if i >= budget {
			break
		}
		ids = append(ids, n.ID)
	}
	return ids, nil
}

func nodeMatches(n Node, words []string) bool {
	lowName := strings.ToLower(n.Name)
	lowPath := strings.ToLower(n.Path)
	for _, w := range words {
		if !strings.Contains(lowName, w) && !strings.Contains(lowPath, w) {
			return false
		}
	}
	return true
}

func (d *DB) bfsNeighborhood(ctx context.Context, seeds []string, budget int) (QueryResult, error) {
	inSet := map[string]bool{}
	queue := seeds
	for _, s := range seeds {
		inSet[s] = true
	}
	var edgeList []Edge
	seenEdge := map[string]bool{}

	// Drain the full queue even when seeds already fill the budget: seed
	// visits still collect the edges among the chosen set (membership
	// rule below), while growth (new neighbors) stays budget-gated, so
	// a full budget yields a connected seed subgraph instead of zero
	// edges. Each node is queued at most once (inSet marks on push),
	// so the loop always terminates.
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		edges, err := d.edgesTouching(ctx, cur)
		if err != nil {
			return QueryResult{}, err
		}
		for _, e := range edges {
			key := e.Src + "|" + e.Dst + "|" + e.Kind + "|" + e.Label
			if seenEdge[key] {
				continue
			}
			other := ""
			if e.Src == cur {
				other = e.Dst
			} else if e.Dst == cur {
				other = e.Src
			}
			if e.Dst == "" && e.Src == cur {
				edgeList = append(edgeList, e)
				seenEdge[key] = true
				continue
			}
			if other != "" && !inSet[other] && len(inSet) < budget {
				inSet[other] = true
				queue = append(queue, other)
			}
			if (e.Src != "" && inSet[e.Src]) && (e.Dst == "" || inSet[e.Dst]) {
				edgeList = append(edgeList, e)
				seenEdge[key] = true
			}
		}
	}

	var nodes []Node
	for id := range inSet {
		n, err := d.loadNode(ctx, id)
		if err != nil {
			continue
		}
		nodes = append(nodes, n)
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].ID < nodes[j].ID })

	var outEdges []Edge
	for _, e := range edgeList {
		if inSet[e.Src] && (e.Dst == "" || inSet[e.Dst]) {
			outEdges = append(outEdges, e)
		}
	}
	return QueryResult{Nodes: nodes, Edges: outEdges}, nil
}

func (d *DB) edgesTouching(ctx context.Context, id string) ([]Edge, error) {
	rows, err := d.sql.QueryContext(ctx, `
		SELECT src, dst, kind, label, confidence FROM graph_edges WHERE src = ? OR dst = ?`, id, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Edge
	for rows.Next() {
		var e Edge
		if err := rows.Scan(&e.Src, &e.Dst, &e.Kind, &e.Label, &e.Confidence); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool {
		return edgeConfidenceRank(out[i].Confidence) < edgeConfidenceRank(out[j].Confidence)
	})
	return out, nil
}
