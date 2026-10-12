package main

import (
	"github.com/spf13/cobra"
	"github.com/ur-wesley/fabrik/cli/internal/flow"
	"github.com/ur-wesley/fabrik/cli/internal/ui"
	"github.com/ur-wesley/fabrik/cli/internal/version"
)

func init() {
	flow.OpenLoopStore = openDoltLoopStore
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "fabrik",
		Short: "Fabrik setup: machine tools + per-repo init for Cursor, OpenCode, Pi",
		Long: `Fabrik sets up native stores (machine memory, repo Dolt issues, repo code graph)
and registers the fabrik MCP server for Cursor, OpenCode, Pi, and Antigravity.`,
		Version:      version.Get(),
		SilenceUsage: true,
	}
	root.SetVersionTemplate("fabrik {{.Version}}\n")
	root.Flags().BoolP("version", "v", false, "print CLI version")
	root.PersistentFlags().StringVarP(&repoDirFlag, "C", "C", "", "repo directory (default: walk up for .fabrik/dolt)")
	root.AddCommand(newSetupCmd(), newInitCmd(), newResetCmd(), newCheckCmd(), newUpdateDepsCmd(), newVersionCmd(), newSelfUpdateCmd(), newShowCmd(), newListCmd(), newRunCmd(), newLoopCmd(), newMigrateCmd(), newListenCmd(), newMemCmd(), newIssueCmd(), newGraphCmd(), newMCPCmd(), newVcCmd(), newDoltCmd(), newHistoryCmd(), newExportCmd())
	_ = ui.OK
	return root
}
