package main

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/ur-wesley/fabrik/cli/internal/doltstore"
)

// newSyncCmd is the `fabrik sync` trigger: pull (git + snapshot merge)
// then push (export + commit + push) through the same seams as
// `fabrik dolt pull` / `fabrik dolt push`.
func newSyncCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Pull then push the issue snapshot via git",
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := openRepoStore(cmd)
			if err != nil {
				return err
			}
			defer st.Close()
			res, err := st.Sync(cmd.Context())
			if errors.Is(err, doltstore.ErrNoDoltRemote) {
				return err
			}
			if err != nil {
				return err
			}
			if asJSON {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(res)
			}
			if res.Pulled {
				fmt.Fprintf(cmd.OutOrStdout(), "pulled %d commits from %s\n", res.Commits, res.Remote)
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "already up to date")
			}
			if res.Pushed {
				fmt.Fprintf(cmd.OutOrStdout(), "pushed to %s\n", res.Remote)
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "nothing to push")
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "machine-readable output")
	return cmd
}
