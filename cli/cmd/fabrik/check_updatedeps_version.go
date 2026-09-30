package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/ur-wesley/fabrik/cli/internal/check"
	"github.com/ur-wesley/fabrik/cli/internal/deps"
	"github.com/ur-wesley/fabrik/cli/internal/exec"
	"github.com/ur-wesley/fabrik/cli/internal/updatedeps"
	"github.com/ur-wesley/fabrik/cli/internal/version"
)

func runCheckQuiet(ctx context.Context) error {
	r := check.Run(ctx, exec.OSRunner{})
	if !r.OK {
		return fmt.Errorf("missing tools")
	}
	return nil
}

func newCheckCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Probe PATH for bd, engram, graphify, bun, pi, uv",
		RunE: func(cmd *cobra.Command, args []string) error {
			r := check.Run(cmd.Context(), exec.OSRunner{})
			if asJSON {
				return check.PrintJSONTo(cmd.OutOrStdout(), r)
			}
			check.Print(cmd.OutOrStdout(), r)
			if !r.OK {
				os.Exit(1)
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
		Short: "Print CLI version + pinned tool versions",
		RunE: func(cmd *cobra.Command, args []string) error {
			pins, err := deps.Load()
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "fabrik %s\n", version.Get())
			fmt.Fprintf(out, "beads %s, engram %s, graphify %s\n", pins.Beads.Tag, pins.Engram.Tag, pins.Graphify.Version)
			return nil
		},
	}
}
