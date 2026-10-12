package graph

import (
	"context"
	"sort"
)

func (d *DB) Path(ctx context.Context, from, to string) (QueryResult, error) {
	if err := d.requireIndexed(ctx); err != nil {
		return QueryResult{}, err
	}
	if from == to {
		n, err := d.loadNode(ctx, from)
		if err != nil {
			return QueryResult{}, err
		}
		return QueryResult{Nodes: []Node{n}, Edges: []Edge{}}, nil
	}
	if _, err := d.loadNode(ctx, from); err != nil {
		return QueryResult{}, err
	}
	if _, err := d.loadNode(ctx, to); err != nil {
		return QueryResult{}, err
	}

	type state struct {
		id    string
		path  []string
		edges []Edge
	}
	queue := []state{{id: from, path: []string{from}, edges: nil}}
	visited := map[string]bool{from: true}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if len(cur.edges) >= 20 {
			continue
		}
		if cur.id == to {
			return buildPathResult(ctx, d, cur.path, cur.edges)
		}
		edges, err := d.edgesTouching(ctx, cur.id)
		if err != nil {
			return QueryResult{}, err
		}
		sort.Slice(edges, func(i, j int) bool {
			di, dj := edges[i].Dst, edges[j].Dst
			if di == "" {
				di = edges[i].Label
			}
			if dj == "" {
				dj = edges[j].Label
			}
			return di < dj
		})
		for _, e := range edges {
			if e.Dst == "" {
				continue
			}
			next := ""
			if e.Src == cur.id {
				next = e.Dst
			} else if e.Dst == cur.id {
				next = e.Src
			}
			if next == "" || visited[next] {
				continue
			}
			visited[next] = true
			p := append(append([]string{}, cur.path...), next)
			es := append(append([]Edge{}, cur.edges...), e)
			queue = append(queue, state{id: next, path: p, edges: es})
			if next == to {
				return buildPathResult(ctx, d, p, es)
			}
		}
	}
	return QueryResult{Nodes: []Node{}, Edges: []Edge{}}, nil
}

func buildPathResult(ctx context.Context, d *DB, ids []string, edges []Edge) (QueryResult, error) {
	var nodes []Node
	for _, id := range ids {
		n, err := d.loadNode(ctx, id)
		if err != nil {
			return QueryResult{}, err
		}
		nodes = append(nodes, n)
	}
	return QueryResult{Nodes: nodes, Edges: edges}, nil
}
