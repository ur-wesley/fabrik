package main

import (
	"github.com/spf13/cobra"
	"github.com/ur-wesley/fabrik/cli/internal/doltstore"
	"github.com/ur-wesley/fabrik/cli/internal/store"
)

func openRepoStore(cmd *cobra.Command) (*doltstore.Store, error) {
	root, err := store.RequireRepo(repoStartDir())
	if err != nil {
		return nil, err
	}
	return doltstore.Open(cmd.Context(), root)
}
