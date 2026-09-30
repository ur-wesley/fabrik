package main

import (
	"github.com/spf13/cobra"
	"github.com/ur-wesley/fabrik/cli/internal/selfupdate"
)

func newSelfUpdateCmd() *cobra.Command {
	var cfg selfupdate.Config
	var nonInteractive bool
	cmd := &cobra.Command{
		Use:   "self-update",
		Short: "Update fabrik to the latest release (checksum-verified)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if nonInteractive {
				cfg.Yes = true
			}
			return selfupdate.Update(cmd.Context(), cfg, selfupdate.Deps{Out: cmd.OutOrStdout()})
		},
	}
	f := cmd.Flags()
	f.BoolVar(&cfg.CheckOnly, "check", false, "report only, do not install")
	f.StringVar(&cfg.WantVersion, "version", "", "install a specific tag (default: latest)")
	f.BoolVar(&cfg.Yes, "yes", false, "answer yes to all prompts (interactive by default)")
	f.BoolVar(&nonInteractive, "non-interactive", false, "alias for --yes (CI)")
	f.BoolVar(&cfg.DryRun, "dry-run", false, "resolve without downloading or swapping")
	return cmd
}
