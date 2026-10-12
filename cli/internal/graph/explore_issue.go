package graph

import (
	"context"
	"strings"
	"unicode"
)

type ExploreOpts struct {
	Query         string
	Budget        int
	IssueSeedText string
}

func (d *DB) seedsFromText(ctx context.Context, text string) ([]string, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, nil
	}
	tokens := identifierTokens(text)
	if len(tokens) == 0 {
		return nil, nil
	}
	rows, err := d.sql.QueryContext(ctx, `SELECT id, name FROM graph_nodes WHERE kind = 'symbol'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tokenSet := map[string]bool{}
	for _, t := range tokens {
		tokenSet[strings.ToLower(t)] = true
	}
	var ids []string
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		if tokenSet[strings.ToLower(name)] {
			ids = append(ids, id)
		}
	}
	return ids, rows.Err()
}

func identifierTokens(s string) []string {
	var out []string
	var cur strings.Builder
	flush := func() {
		if cur.Len() >= 2 {
			out = append(out, cur.String())
		}
		cur.Reset()
	}
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			cur.WriteRune(r)
		} else {
			flush()
		}
	}
	flush()
	return out
}

func mergeSeedIDs(a, b []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, id := range a {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	for _, id := range b {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}

func buildExplorePaths(seeds []string, edges []Edge) []ExplorePath {
	var paths []ExplorePath
	for _, s := range seeds {
		for _, e := range edges {
			if e.Src != s || e.Dst == "" {
				continue
			}
			if e.Kind != "calls" && e.Kind != "imports" {
				continue
			}
			hop := e.Confidence
			if hop == "inferred" && e.Label != "" {
				hop = "dynamic"
			}
			paths = append(paths, ExplorePath{From: s, To: e.Dst, Hops: []string{hop}})
			if len(paths) >= 20 {
				return paths
			}
		}
	}
	return paths
}
