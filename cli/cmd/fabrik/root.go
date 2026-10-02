package main

import (
	"github.com/spf13/cobra"
	"github.com/ur-wesley/fabrik/cli/internal/ui"
)

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "fabrik",
		Short: "Fabrik setup: machine tools + per-repo init for Cursor, OpenCode, Pi",
		Long: `Fabrik sets up the AI workflow (Beads, Engram, Graphify) on your machine
and initializes repos with the .fabrik overview hub.

Apps: Cursor, OpenCode, Pi only.`,
		SilenceUsage: true,
	}
	root.AddCommand(newSetupCmd(), newInitCmd(), newResetCmd(), newCheckCmd(), newUpdateDepsCmd(), newVersionCmd(), newSelfUpdateCmd(), newShowCmd(), newListCmd(), newRunCmd(), newLoopCmd(), newMigrateCmd())
	_ = ui.OK
	return root
}
