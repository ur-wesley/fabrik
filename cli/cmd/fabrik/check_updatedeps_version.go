package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/ur-wesley/fabrik/cli/internal/check"
	"github.com/ur-wesley/fabrik/cli/internal/store"
	"github.com/ur-wesley/fabrik/cli/internal/updatedeps"
	"github.com/ur-wesley/fabrik/cli/internal/version"
)

func runCheckQuiet(ctx context.Context) error {
	_, code := check.Run(ctx, "")
	if code != 0 {
		return fmt.Errorf("repo store not ready")
	}
	return nil
}

func newCheckCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Report repo issues/graph store health",
		RunE: func(cmd *cobra.Command, args []string) error {
			repoRoot := repoDirFlag
			if repoRoot == "" {
				if root, ok := store.RepoRoot(repoStartDir()); ok {
					repoRoot = root
				}
			}
			r, code := check.Run(cmd.Context(), repoRoot)
			if asJSON {
				if err := check.PrintJSONTo(cmd.OutOrStdout(), r); err != nil {
					return err
				}
				if code != 0 {
					return &check.ExitError{Code: code}
				}
				return nil
			}
			check.Print(cmd.OutOrStdout(), r)
			check.PrintPluginAdvisory(cmd.OutOrStdout())
			if code != 0 {
				return &check.ExitError{Code: code}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "machine-readable output")
	return cmd
}

func newUpdateDepsCmd() *cobra.Command {
	var depsPath string
	var yes, nonInteractive, dryRun bool
	cmd := &cobra.Command{
		Use:   "update-deps",
		Short: "Refresh pinned tool versions in deps.json (GitHub + PyPI)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if nonInteractive {
				yes = true
			}
			return updatedeps.Refresh(cmd.Context(), cmd.OutOrStdout(), &updatedeps.Fetcher{}, depsPath, yes, dryRun)
		},
	}
	cmd.Flags().StringVar(&depsPath, "deps", "install/deps.json", "path to deps.json to rewrite")
	cmd.Flags().BoolVar(&yes, "yes", false, "answer yes to all prompts (interactive by default)")
	cmd.Flags().BoolVar(&nonInteractive, "non-interactive", false, "alias for --yes (CI)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "resolve pins without writing")
	return cmd
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print CLI version",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintf(cmd.OutOrStdout(), "fabrik %s\n", version.Get())
			return nil
		},
	}
}
