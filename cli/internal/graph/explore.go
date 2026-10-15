package graph

import (
	"context"
	"os"
	"path/filepath"
	"strings"
)

func (d *DB) Explore(ctx context.Context, opts ExploreOpts) (ExploreResult, error) {
	idx, err := d.indexed(ctx)
	if err != nil {
		return ExploreResult{}, err
	}
	if !idx {
		return ExploreResult{Indexed: false, Hint: "run fabrik graph build in repo root"}, nil
	}
	if err := d.RefreshStale(ctx); err != nil {
		return ExploreResult{}, err
	}
	budget := opts.Budget
	if budget <= 0 {
		budget = 40
	}
	issueSeeds, err := d.seedsFromText(ctx, opts.IssueSeedText)
	if err != nil {
		return ExploreResult{}, err
	}
	qr, err := d.queryWithSeeds(ctx, opts.Query, budget, issueSeeds)
	if err != nil {
		return ExploreResult{}, err
	}
	res := ExploreResult{Indexed: true}
	paths := map[string]bool{}
	for _, n := range qr.Nodes {
		if n.Kind == "symbol" && n.Path != "" {
			paths[n.Path] = true
		}
	}
	for p := range paths {
		snippets, err := snippetsForFile(d.Repo, p, qr.Nodes)
		if err != nil {
			continue
		}
		if len(snippets) > 0 {
			res.Files = append(res.Files, ExploreFile{Path: p, Snippets: snippets})
		}
	}
	var callers []string
	for _, e := range qr.Edges {
		if e.Kind == "calls" && e.Confidence == "extracted" && e.Dst != "" {
			callers = append(callers, e.Src)
		}
	}
	res.BlastRadius = &BlastRadius{Callers: callers, DependentsCount: len(callers)}
	res.Paths = buildExplorePaths(mergeSeedIDs(issueSeeds, seedIDsFromNodes(qr.Nodes)), qr.Edges)
	for _, e := range qr.Edges {
		if e.Kind == "calls" && e.Confidence == "inferred" && e.Label != "" {
			if cands := d.candidatesForName(ctx, e.Label); len(cands) > 0 {
				res.Candidates = append(res.Candidates, cands...)
			}
		}
	}
	return res, nil
}

func seedIDsFromNodes(nodes []Node) []string {
	var ids []string
	for _, n := range nodes {
		if n.Kind == "symbol" {
			ids = append(ids, n.ID)
		}
	}
	return ids
}

func (d *DB) candidatesForName(ctx context.Context, name string) []string {
	rows, err := d.sql.QueryContext(ctx, `
		SELECT id FROM graph_nodes WHERE kind = 'symbol' AND name = ?`, name)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil
		}
		out = append(out, id)
	}
	if err := rows.Err(); err != nil {
		return nil
	}
	return out
}

func snippetsForFile(repo, rel string, nodes []Node) ([]Snippet, error) {
	data, err := os.ReadFile(filepath.Join(repo, rel))
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(data), "\n")
	var out []Snippet
	for _, n := range nodes {
		if n.Path != rel || n.Kind != "symbol" {
			continue
		}
		start := n.Line
		if start < 1 || start > len(lines) {
			continue
		}
		end := start
		for end < len(lines) && end-start < 40 {
			if end > start && strings.HasPrefix(strings.TrimSpace(lines[end]), "func ") {
				break
			}
			end++
		}
		if end == start {
			end = start + 1
		}
		if end > len(lines) {
			end = len(lines)
		}
		text := strings.Join(lines[start-1:end], "\n")
		out = append(out, Snippet{
			LineStart: start,
			LineEnd:   end,
			Symbol:    n.Name,
			Text:      text,
		})
	}
	return out, nil
}
