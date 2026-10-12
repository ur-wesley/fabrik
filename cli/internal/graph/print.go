package graph

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

func PrintBuild(out io.Writer, r BuildResult) {
	fmt.Fprintf(out, "parsed %d of %d files, skipped %d\n", r.Parsed, r.Files, r.Skipped)
	fmt.Fprintf(out, "%d nodes, %d edges\n", r.Nodes, r.Edges)
}

func PrintBuildJSON(out io.Writer, r BuildResult) error {
	return json.NewEncoder(out).Encode(r)
}

func PrintQuery(out io.Writer, r QueryResult) {
	if len(r.Nodes) == 0 {
		fmt.Fprintln(out, "no matches")
		return
	}
	for _, n := range r.Nodes {
		fmt.Fprintf(out, "%s  %s  %s:%d\n", n.ID, n.Lang, n.Path, n.Line)
		for _, e := range r.Edges {
			if e.Src != n.ID {
				continue
			}
			printEdgeLine(out, "  ", e, n.ID)
		}
	}
}

func PrintQueryJSON(out io.Writer, r QueryResult) error {
	return json.NewEncoder(out).Encode(r)
}

func printEdgeLine(out io.Writer, prefix string, e Edge, from string) {
	if e.Dst == "" {
		fmt.Fprintf(out, "%s%s -> (unresolved) %s\n", prefix, e.Kind, e.Label)
		return
	}
	arrow := "->"
	if e.Dst == from {
		arrow = "<-"
	}
	fmt.Fprintf(out, "%s%s %s %s\n", prefix, e.Kind, arrow, e.Dst)
}

func PrintPath(out io.Writer, r QueryResult) {
	if len(r.Nodes) == 0 {
		fmt.Fprintln(out, "no path")
		return
	}
	for i, n := range r.Nodes {
		fmt.Fprintln(out, n.ID)
		if i < len(r.Edges) {
			e := r.Edges[i]
			printPathEdge(out, n.ID, e)
		}
	}
}

func printPathEdge(out io.Writer, upper string, e Edge) {
	if e.Src == upper {
		if e.Dst == "" {
			fmt.Fprintf(out, "  %s -> (unresolved) %s\n", e.Kind, e.Label)
		} else {
			fmt.Fprintf(out, "  %s -> %s\n", e.Kind, e.Dst)
		}
	} else {
		fmt.Fprintf(out, "  %s <- %s\n", e.Kind, e.Src)
	}
}

func PrintExplain(out io.Writer, r ExplainResult) {
	n := r.Node
	fmt.Fprintf(out, "%s  %s  %s  %s:%d\n", n.ID, n.Kind, n.Lang, n.Path, n.Line)
	for _, e := range r.Edges {
		printEdgeLine(out, "  ", e, n.ID)
	}
	if r.Truncated {
		fmt.Fprintln(out, "truncated")
	}
}

func PrintExplainJSON(out io.Writer, r ExplainResult) error {
	return json.NewEncoder(out).Encode(r)
}

func PrintStatus(out io.Writer, r StatusResult) {
	if !r.Indexed {
		fmt.Fprintln(out, "not indexed")
		return
	}
	fmt.Fprintf(out, "files %d  symbols %d  stale %d\n", r.Files, r.Symbols, r.StaleFiles)
	if r.LastBuildAt != "" {
		fmt.Fprintf(out, "last %s\n", r.LastBuildAt)
	}
}

func PrintStatusJSON(out io.Writer, r StatusResult) error {
	return json.NewEncoder(out).Encode(r)
}

func PrintExplore(out io.Writer, r ExploreResult) {
	if !r.Indexed {
		fmt.Fprintln(out, r.Hint)
		return
	}
	for _, f := range r.Files {
		fmt.Fprintf(out, "## %s\n", f.Path)
		for _, s := range f.Snippets {
			fmt.Fprintf(out, "### %s (%d-%d)\n%s\n", s.Symbol, s.LineStart, s.LineEnd, s.Text)
		}
	}
	if r.BlastRadius != nil && len(r.BlastRadius.Callers) > 0 {
		fmt.Fprintf(out, "## blast radius\n%s\n", strings.Join(r.BlastRadius.Callers, ", "))
	}
}

func PrintExploreJSON(out io.Writer, r ExploreResult) error {
	return json.NewEncoder(out).Encode(r)
}

func PrintContextJSON(out io.Writer, r ContextResult) error {
	return json.NewEncoder(out).Encode(r)
}

func PrintImpactJSON(out io.Writer, r ImpactResult) error {
	return json.NewEncoder(out).Encode(r)
}

func PrintDetectJSON(out io.Writer, r DetectChangesResult) error {
	return json.NewEncoder(out).Encode(r)
}
