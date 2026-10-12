package graph

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ur-wesley/fabrik/cli/internal/repo"
)

func (d *DB) DetectChanges(ctx context.Context, ref string) (DetectChangesResult, error) {
	if err := d.requireIndexed(ctx); err != nil {
		return DetectChangesResult{}, err
	}
	if !repo.IsGitRepo(d.Repo) {
		return DetectChangesResult{}, fmt.Errorf("fabrik: graph detect-changes: not a git repository")
	}
	if err := d.RefreshStale(ctx); err != nil {
		return DetectChangesResult{}, err
	}
	args := []string{"diff", "-U0"}
	if ref != "" && ref != "HEAD" {
		args = append(args, ref)
	}
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = d.Repo
	out, err := cmd.Output()
	if err != nil {
		return DetectChangesResult{}, err
	}
	ranges := parseGitDiffHunks(string(out))
	res := DetectChangesResult{}
	seenSym := map[string]bool{}
	for file, hunks := range ranges {
		res.ChangedFiles = append(res.ChangedFiles, file)
		for _, line := range hunks {
			sid := d.symbolAtLine(ctx, file, line)
			if sid != "" && !seenSym[sid] {
				seenSym[sid] = true
				res.TouchedSymbols = append(res.TouchedSymbols, sid)
			}
		}
	}
	impSeen := map[string]bool{}
	for _, sid := range res.TouchedSymbols {
		imp, err := d.Impact(ctx, sid, 2)
		if err != nil {
			continue
		}
		for _, nodes := range imp.ByDepth {
			for _, n := range nodes {
				if !impSeen[n.ID] {
					impSeen[n.ID] = true
					res.ImpactedSymbols = append(res.ImpactedSymbols, n.ID)
				}
			}
		}
	}
	for _, f := range res.ChangedFiles {
		if chk := suggestCheck(f); chk != "" {
			res.SuggestedChecks = append(res.SuggestedChecks, chk)
		}
	}
	return res, nil
}

func parseGitDiffHunks(diff string) map[string][]int {
	out := map[string][]int{}
	var file string
	for _, line := range strings.Split(diff, "\n") {
		if strings.HasPrefix(line, "+++ b/") {
			file = strings.TrimPrefix(line, "+++ b/")
			continue
		}
		if !strings.HasPrefix(line, "@@") || file == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		seg := strings.TrimPrefix(parts[1], "+")
		if i := strings.Index(seg, ","); i >= 0 {
			seg = seg[:i]
		}
		start, err := strconv.Atoi(seg)
		if err != nil {
			continue
		}
		out[file] = append(out[file], start)
	}
	return out
}

func (d *DB) symbolAtLine(ctx context.Context, file string, line int) string {
	rows, err := d.sql.QueryContext(ctx, `
		SELECT id, line FROM graph_nodes WHERE kind = 'symbol' AND path = ? ORDER BY line DESC`, file)
	if err != nil {
		return ""
	}
	var best string
	defer rows.Close()
	for rows.Next() {
		var id string
		var ln int
		if err := rows.Scan(&id, &ln); err != nil {
			return ""
		}
		if ln <= line {
			best = id
			break
		}
	}
	_ = rows.Err()
	return best
}

func suggestCheck(rel string) string {
	ext := filepath.Ext(rel)
	switch ext {
	case ".go":
		dir := filepath.ToSlash(filepath.Dir(rel))
		if dir == "." {
			return "go test ."
		}
		return "go test ./" + dir + "/..."
	default:
		return ""
	}
}
