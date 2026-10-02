package main

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/ur-wesley/fabrik/cli/internal/exec"
	"github.com/ur-wesley/fabrik/cli/internal/initer"
	"github.com/ur-wesley/fabrik/cli/internal/installer"
)

func newSetupCmd() *cobra.Command {
	var cfg installer.Config
	var nonInteractive bool
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Install machine tools and initialize the current repo",
		RunE: func(cmd *cobra.Command, args []string) error {
			if nonInteractive {
				cfg.Yes = true
			}
			return installer.Setup(cmd.Context(), cfg, installer.Deps{
				Exec: exec.OSRunner{},
				Out:  cmd.OutOrStdout(),
				InitRepo: func(repo string) error {
					return initer.Init(context.Background(), initer.Config{RepoPath: repo, Yes: cfg.Yes, DryRun: cfg.DryRun, Apps: cfg.Apps}, initer.Deps{
						Exec:  exec.OSRunner{},
						Out:   cmd.OutOrStdout(),
						Check: func(ctx context.Context) error { return nil }, // already set up
					})
				},
			})
		},
	}
	f := cmd.Flags()
	f.BoolVar(&cfg.SkipToolInstall, "skip-tool-install", false, "fail if a tool is missing instead of installing it")
	f.BoolVar(&cfg.SkipEngramSetup, "skip-engram-setup", false, "skip engram setup for Cursor/OpenCode/Pi")
	f.BoolVar(&cfg.SkipPiPackages, "skip-pi-packages", false, "skip Pi package install")
	f.StringVar(&cfg.RepoPath, "repo", "", "repo to initialize (default: current directory)")
	f.StringSliceVar(&cfg.Apps, "apps", nil, "apps to configure (cursor, pi, antigravity, opencode)")
	f.BoolVar(&cfg.Yes, "yes", false, "answer yes to all prompts (interactive by default)")
	f.BoolVar(&nonInteractive, "non-interactive", false, "alias for --yes (CI)")
	f.BoolVar(&cfg.DryRun, "dry-run", false, "print actions without changing anything")
	return cmd
}

func newInitCmd() *cobra.Command {
	var cfg initer.Config
	var nonInteractive bool
	cmd := &cobra.Command{
		Use:   "init [path]",
		Short: "Initialize a repo (.fabrik hub + app wiring)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				cfg.RepoPath = args[0]
			}
			if nonInteractive {
				cfg.Yes = true
			}
			return initer.Init(cmd.Context(), cfg, initer.Deps{
				Exec: exec.OSRunner{},
				Out:  cmd.OutOrStdout(),
				Check: func(ctx context.Context) error {
					return runCheckQuiet(ctx)
				},
			})
		},
	}
	f := cmd.Flags()
	f.BoolVar(&cfg.SkipChecks, "skip-checks", false, "skip advisory tool check")
	f.StringSliceVar(&cfg.Apps, "apps", nil, "apps to configure (cursor, pi, antigravity, opencode)")
	f.BoolVar(&cfg.Yes, "yes", false, "answer yes to all prompts (interactive by default)")
	f.BoolVar(&nonInteractive, "non-interactive", false, "alias for --yes (CI)")
	f.BoolVar(&cfg.DryRun, "dry-run", false, "print actions without changing anything")
	f.BoolVar(&cfg.Full, "full", false, "restore legacy full-hub dump (one-release rollback; default is thin)")
	return cmd
}
