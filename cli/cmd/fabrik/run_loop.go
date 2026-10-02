package main

import (
	"github.com/spf13/cobra"
	"github.com/ur-wesley/fabrik/cli/internal/exec"
	"github.com/ur-wesley/fabrik/cli/internal/flow"
)

// execRunner is the process runner for run/loop; swapped for a fake in tests.
var execRunner exec.Runner = exec.OSRunner{}

func newRunCmd() *cobra.Command {
	var auto bool
	var prompt string
	var repo string
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run the Fabrik workflow: align, plan, build (port of fabrik.sh)",
		Long: `Setup gate (.beads + .fabrik/config.yaml), then --auto generates the PRD
via 'opencode run' or launches the TUI, then plans and pauses for APPROVE
before the build wave (shared CTA agents/cta.md).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return flow.Run(cmd.Context(), flow.RunConfig{RepoPath: repo, Auto: auto, Prompt: prompt},
				flow.Deps{Exec: execRunner, Out: cmd.OutOrStdout()})
		},
	}
	cmd.Flags().BoolVarP(&auto, "auto", "a", false, "generate PRD non-interactively from -p/--prompt")
	cmd.Flags().StringVarP(&prompt, "prompt", "p", "", "initial requirements for PRD generation (with --auto)")
	cmd.Flags().StringVar(&repo, "repo", "", "repo path (default: cwd)")
	return cmd
}

func newLoopCmd() *cobra.Command {
	var max int
	var push bool
	var repo string
	cmd := &cobra.Command{
		Use:   "loop <plan|build>",
		Short: "Poll bd ready/list and drive opencode runs (port of loop.sh)",
		Long: `Plan runs once then exits; build loops until no open/ready issues
remain or --max iterations hit. Stdout only, no state files. Push is opt-in
via --push (default: no push without explicit approval).`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return flow.Loop(cmd.Context(), flow.LoopConfig{RepoPath: repo, Mode: args[0], Max: max, AutoPush: push},
				flow.Deps{Exec: execRunner, Out: cmd.OutOrStdout()})
		},
	}
	cmd.Flags().IntVar(&max, "max", 0, "max iterations (0 = until no open/ready issues)")
	cmd.Flags().BoolVar(&push, "push", false, "push to origin after each iteration (explicit opt-in)")
	cmd.Flags().StringVar(&repo, "repo", "", "repo path (default: cwd)")
	return cmd
}
