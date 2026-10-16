package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/ur-wesley/fabrik/cli/internal/clierr"
	"github.com/ur-wesley/fabrik/cli/internal/doltstore"
	"github.com/ur-wesley/fabrik/cli/internal/graph"
)

func newGraphCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "graph",
		Short: "Repo code graph (SQLite)",
	}
	root.AddCommand(
		newGraphBuildCmd(),
		newGraphStatusCmd(),
		newGraphQueryCmd(),
		newGraphPathCmd(),
		newGraphExplainCmd(),
		newGraphExploreCmd(),
		newGraphOverviewCmd(),		newGraphContextCmd(),
		newGraphImpactCmd(),
		newGraphDetectChangesCmd(),
	)
	return root
}

func openGraph(cmd *cobra.Command) (*graph.DB, error) {
	return graph.Open(cmd.Context(), repoStartDir())
}

func newGraphBuildCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "build",
		Short: "Parse repo into graph.db",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := openGraph(cmd)
			if err != nil {
				return err
			}
			defer db.Close()
			r, err := db.Build(cmd.Context())
			if err != nil {
				return err
			}
			if asJSON {
				return graph.PrintBuildJSON(cmd.OutOrStdout(), r)
			}
			graph.PrintBuild(cmd.OutOrStdout(), r)
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "machine-readable output")
	return cmd
}

func newGraphStatusCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Graph index health",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := openGraph(cmd)
			if err != nil {
				return err
			}
			defer db.Close()
			r, err := db.Status(cmd.Context())
			if err != nil {
				return err
			}
			if asJSON {
				return graph.PrintStatusJSON(cmd.OutOrStdout(), r)
			}
			graph.PrintStatus(cmd.OutOrStdout(), r)
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "machine-readable output")
	return cmd
}

func newGraphQueryCmd() *cobra.Command {
	var asJSON bool
	var budget int
	cmd := &cobra.Command{
		Use:   "query <text>",
		Short: "Neighborhood search",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return clierr.Usage("fabrik: usage: fabrik graph query <text>")
			}
			if budget < 1 || budget > 200 {
				return clierr.Usage("fabrik: budget must be 1..200")
			}
			db, err := openGraph(cmd)
			if err != nil {
				return err
			}
			defer db.Close()
			text := strings.TrimSpace(strings.Join(args, " "))
			r, err := db.Query(cmd.Context(), text, budget)
			if err != nil {
				return err
			}
			if asJSON {
				return graph.PrintQueryJSON(cmd.OutOrStdout(), r)
			}
			graph.PrintQuery(cmd.OutOrStdout(), r)
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "machine-readable output")
	cmd.Flags().IntVar(&budget, "budget", 40, "max nodes (1-200)")
	return cmd
}

func newGraphPathCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "path <from> <to>",
		Short: "Shortest path between nodes",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 2 {
				return clierr.Usage("fabrik: usage: fabrik graph path <from> <to>")
			}
			db, err := openGraph(cmd)
			if err != nil {
				return err
			}
			defer db.Close()
			r, err := db.Path(cmd.Context(), args[0], args[1])
			if err != nil {
				return err
			}
			if asJSON {
				return graph.PrintQueryJSON(cmd.OutOrStdout(), r)
			}
			graph.PrintPath(cmd.OutOrStdout(), r)
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "machine-readable output")
	return cmd
}

func newGraphExplainCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "explain <id>",
		Short: "One node and its edges",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return clierr.Usage("fabrik: usage: fabrik graph explain <id>")
			}
			db, err := openGraph(cmd)
			if err != nil {
				return err
			}
			defer db.Close()
			r, err := db.Explain(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if asJSON {
				return graph.PrintExplainJSON(cmd.OutOrStdout(), r)
			}
			graph.PrintExplain(cmd.OutOrStdout(), r)
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "machine-readable output")
	return cmd
}

func newGraphExploreCmd() *cobra.Command {
	var asJSON bool
	var budget int
	var issueID string
	cmd := &cobra.Command{
		Use:   "explore <query>",
		Short: "Explore symbols with snippets",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return clierr.Usage("fabrik: usage: fabrik graph explore <query>")
			}
			db, err := openGraph(cmd)
			if err != nil {
				return err
			}
			defer db.Close()
			opts := graph.ExploreOpts{Query: strings.Join(args, " "), Budget: budget}
			if issueID != "" {
				st, err := openIssueStore(cmd)
				if err != nil {
					return err
				}
				defer st.Close()
				iss, err := st.Show(cmd.Context(), issueID)
				if err != nil {
					return err
				}
				opts.IssueSeedText = doltstore.IssueExploreSeedText(iss)
			}
			r, err := db.Explore(cmd.Context(), opts)
			if err != nil {
				return err
			}
			if asJSON {
				return graph.PrintExploreJSON(cmd.OutOrStdout(), r)
			}
			graph.PrintExplore(cmd.OutOrStdout(), r)
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "machine-readable output")
	cmd.Flags().IntVar(&budget, "budget", 40, "max symbols")
	cmd.Flags().StringVar(&issueID, "issue", "", "seed from issue id (f-…)")
	return cmd
}

func newGraphOverviewCmd() *cobra.Command {
	var asJSON bool
	var budget int
	cmd := &cobra.Command{
		Use:   "overview",
		Short: "Top symbols by edge degree",
		RunE: func(cmd *cobra.Command, args []string) error {
			if budget < 1 || budget > 200 {
				return clierr.Usage("fabrik: budget must be 1..200")
			}
			db, err := openGraph(cmd)
			if err != nil {
				return err
			}
			defer db.Close()
			r, err := db.Overview(cmd.Context(), budget)
			if err != nil {
				return err
			}
			if asJSON {
				return graph.PrintQueryJSON(cmd.OutOrStdout(), r)
			}
			graph.PrintQuery(cmd.OutOrStdout(), r)
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "machine-readable output")
	cmd.Flags().IntVar(&budget, "budget", 60, "max nodes (1-200)")
	return cmd
}

func newGraphContextCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "context <symbol>",
		Short: "Symbol context with memory",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return clierr.Usage("fabrik: usage: fabrik graph context <symbol>")
			}
			db, err := openGraph(cmd)
			if err != nil {
				return err
			}
			defer db.Close()
			r, err := db.Context(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if asJSON {
				return graph.PrintContextJSON(cmd.OutOrStdout(), r)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s  %s:%d\n", r.Symbol.ID, r.Symbol.Path, r.Symbol.Line)
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "machine-readable output")
	return cmd
}

func newGraphImpactCmd() *cobra.Command {
	var asJSON bool
	var depth int
	cmd := &cobra.Command{
		Use:   "impact <symbol>",
		Short: "Blast radius of a symbol",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return clierr.Usage("fabrik: usage: fabrik graph impact <symbol>")
			}
			db, err := openGraph(cmd)
			if err != nil {
				return err
			}
			defer db.Close()
			r, err := db.Impact(cmd.Context(), args[0], depth)
			if err != nil {
				return err
			}
			if asJSON {
				return graph.PrintImpactJSON(cmd.OutOrStdout(), r)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "total %d\n", r.Total)
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "machine-readable output")
	cmd.Flags().IntVar(&depth, "depth", 3, "walk depth")
	return cmd
}

func newGraphDetectChangesCmd() *cobra.Command {
	var asJSON bool
	var ref string
	cmd := &cobra.Command{
		Use:   "detect-changes",
		Short: "Map git diff to symbols",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := openGraph(cmd)
			if err != nil {
				return err
			}
			defer db.Close()
			r, err := db.DetectChanges(cmd.Context(), ref)
			if err != nil {
				return err
			}
			if asJSON {
				return graph.PrintDetectJSON(cmd.OutOrStdout(), r)
			}
			for _, s := range r.TouchedSymbols {
				fmt.Fprintln(cmd.OutOrStdout(), s)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "machine-readable output")
	cmd.Flags().StringVar(&ref, "ref", "HEAD", "git diff base ref")
	return cmd
}
