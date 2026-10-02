package main

import (
	"github.com/spf13/cobra"
	"github.com/ur-wesley/fabrik/cli/internal/migrater"
)

func newMigrateCmd() *cobra.Command {
	var cfg migrater.Config
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Prune legacy generated dumps (keeps config.yaml, docs/, specs/, shims)",
		Long: `Delete legacy generated files only (.fabrik PROMPT_*/loop/fabrik/setup dumps,
README/CONTEXT/skills/AGENTS bodies, STYLEGUIDE, agents/*.md bodies, skill/agent
full copies). Never deletes .fabrik/config.yaml, .fabrik/docs/, .fabrik/specs/,
user PRD, root AGENTS.md, opencode.json. Shim files containing
"fabrik show skill|agent" are kept. Requires --prune; use --dry-run to preview.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return migrater.Migrate(cfg, migrater.Deps{Out: cmd.OutOrStdout()})
		},
	}
	f := cmd.Flags()
	f.BoolVar(&cfg.Prune, "prune", false, "delete generated files (required)")
	f.BoolVar(&cfg.DryRun, "dry-run", false, "print would-delete list without changing anything")
	f.StringVar(&cfg.RepoPath, "repo", "", "repo path (default: cwd)")
	return cmd
}
