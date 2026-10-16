package main

import (
	"github.com/spf13/cobra"
	"github.com/ur-wesley/fabrik/cli/internal/migrater"
)

func newMigrateCmd() *cobra.Command {
	var cfg migrater.Config
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Import Beads/Graphify/Engram into the Fabrik-native stack; optionally prune legacy dumps",
		Long: `Import legacy state without deleting it: .beads/issues.jsonl (else bd list --all)
-> .fabrik/issues.db, graphify-out/graph.json -> .fabrik/graph.db, Engram
observations (~/.engram/engram.db, FABRIK_ENGRAM_DB override) ->
~/.fabrik/memory.db (FABRIK_MEMORY_DIR override). Sources are never deleted,
even with --prune. With --prune also delete legacy generated dump files
(.fabrik PROMPT_*/loop/fabrik/setup dumps, README/CONTEXT/skills/AGENTS
bodies, STYLEGUIDE, agents/*.md bodies, skill/agent full copies). Never
deletes .fabrik/config.yaml, .fabrik/docs/, .fabrik/specs/, user PRD, root
AGENTS.md, opencode.json. Shim files containing
"fabrik show skill|agent" are kept. Use --dry-run to preview. A run with
nothing to do prints "migrate: nothing to migrate".`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return migrater.Migrate(cfg, migrater.Deps{Out: cmd.OutOrStdout()})
		},
	}
	f := cmd.Flags()
	f.BoolVar(&cfg.Prune, "prune", false, "also delete legacy generated dump files (import runs first)")
	f.BoolVar(&cfg.DryRun, "dry-run", false, "print would-import (and, with --prune, would-delete) list without changing anything")
	f.StringVar(&cfg.RepoPath, "repo", "", "repo path (default: cwd)")
	return cmd
}
