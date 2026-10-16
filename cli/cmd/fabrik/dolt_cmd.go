package main

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/ur-wesley/fabrik/cli/internal/doltstore"
	"github.com/ur-wesley/fabrik/cli/internal/export"
)

func newDoltCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "dolt",
		Short: "Sync the issue store via git (dolt-compatible surface)",
	}
	root.AddCommand(newDoltPushCmd(), newDoltPullCmd(), newDoltRemoteCmd())
	return root
}

func newDoltPushCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "push",
		Short: "Push issue snapshot to remote",
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := openRepoStore(cmd)
			if err != nil {
				return err
			}
			defer st.Close()
			res, err := st.DoltPush(cmd.Context())
			if errors.Is(err, doltstore.ErrNoDoltRemote) {
				return err
			}
			if err != nil {
				return err
			}
			if asJSON {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(res)
			}
			if res.Pushed && res.Commits > 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "pushed %d commits to %s\n", res.Commits, res.Remote)
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "nothing to push")
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "machine-readable output")
	return cmd
}

func newDoltPullCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "pull",
		Short: "Pull issue snapshot from remote",
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := openRepoStore(cmd)
			if err != nil {
				return err
			}
			defer st.Close()
			res, err := st.DoltPull(cmd.Context())
			if errors.Is(err, doltstore.ErrNoDoltRemote) {
				return err
			}
			if err != nil {
				return err
			}
			if _, err := export.Issues(cmd.Context(), st); err != nil {
				return doltstore.StoreErr(err)
			}
			if asJSON {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(res)
			}
			if res.Pulled && res.Commits > 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "pulled %d commits from %s\n", res.Commits, res.Remote)
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "already up to date")
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "machine-readable output")
	return cmd
}

func newDoltRemoteCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "remote",
		Short: "Manage git remotes for issue sync",
	}
	root.AddCommand(newDoltRemoteAddCmd(), newDoltRemoteListCmd())
	return root
}

func newDoltRemoteAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <name> <url>",
		Short: "Add a sync remote",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := openRepoStore(cmd)
			if err != nil {
				return err
			}
			defer st.Close()
			if err := st.RemoteAdd(cmd.Context(), args[0], args[1]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "remote %s %s\n", args[0], args[1])
			return nil
		},
	}
	return cmd
}

func newDoltRemoteListCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List sync remotes",
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := openRepoStore(cmd)
			if err != nil {
				return err
			}
			defer st.Close()
			remotes, err := st.ListRemotes(cmd.Context())
			if err != nil {
				return doltstore.StoreErr(err)
			}
			if asJSON {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(remotes)
			}
			for _, r := range remotes {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\n", r.Name, r.URL)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "machine-readable output")
	return cmd
}
