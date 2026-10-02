package main

import (
	"github.com/spf13/cobra"
	"github.com/ur-wesley/fabrik/cli/internal/exec"
	"github.com/ur-wesley/fabrik/cli/internal/reseter"
)

func newResetCmd() *cobra.Command {
	var cfg reseter.Config
	var nonInteractive bool
	cmd := &cobra.Command{
		Use:     "reset [path]",
		Aliases: []string{"deinit", "uninit"},
		Short:   "Reset and remove Fabrik configuration from a repo",
		Long: `Reset and remove Fabrik configuration from the local project so default
behavior is restored across Cursor, OpenCode, and Pi.

Removes opencode.json (Fabrik agents), .cursor rules & agents, .opencode skills,
.pi skills & agents, strips Fabrik workflow instructions from AGENTS.md, and removes
the entire .fabrik/ directory. Empty directories are pruned. Prompts to remove
Beads integration and .beads as well (or use --beads / --no-beads).`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				cfg.RepoPath = args[0]
			}
			if nonInteractive {
				cfg.Yes = true
			}
			return reseter.Reset(cmd.Context(), cfg, reseter.Deps{
				Exec: exec.OSRunner{},
				Out:  cmd.OutOrStdout(),
			})
		},
	}
	f := cmd.Flags()
	f.BoolVarP(&cfg.Yes, "yes", "y", false, "answer yes to all prompts (interactive by default)")
	f.BoolVar(&nonInteractive, "non-interactive", false, "alias for --yes (CI)")
	f.BoolVar(&cfg.DryRun, "dry-run", false, "print actions without changing anything")
	f.BoolVar(&cfg.Beads, "beads", false, "remove beads integration and .beads as well (skips prompt)")
	f.BoolVar(&cfg.NoBeads, "no-beads", false, "keep beads integration and .beads (skips prompt)")
	f.StringVar(&cfg.RepoPath, "repo", "", "repo path (default: cwd)")
	return cmd
}
