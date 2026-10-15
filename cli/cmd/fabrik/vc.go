package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/spf13/cobra"
	"github.com/ur-wesley/fabrik/cli/internal/clierr"
	"github.com/ur-wesley/fabrik/cli/internal/doltstore"
)

func newVcCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "vc",
		Short: "Dolt version control for the issue store",
	}
	root.AddCommand(newVcStatusCmd(), newVcLogCmd(), newVcDiffCmd(), newVcCommitCmd(), newVcBranchCmd(), newVcCheckoutCmd(), newVcMergeCmd())
	return root
}

func newVcStatusCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show branch, HEAD, and working changes",
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := openRepoStore(cmd)
			if err != nil {
				return err
			}
			defer st.Close()
			res, err := st.VCStatus(cmd.Context())
			if err != nil {
				return doltstore.StoreErr(err)
			}
			if asJSON {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(res)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "branch %s\nhead %s\n", res.Branch, res.Head)
			if res.Dirty {
				fmt.Fprintln(cmd.OutOrStdout(), "dirty")
				for _, c := range res.Changes {
					fmt.Fprintf(cmd.OutOrStdout(), "%s\n", c.Table)
				}
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "clean")
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "machine-readable output")
	return cmd
}

func newVcLogCmd() *cobra.Command {
	var asJSON bool
	var limit int
	cmd := &cobra.Command{
		Use:   "log",
		Short: "List recent Dolt commits",
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := openRepoStore(cmd)
			if err != nil {
				return err
			}
			defer st.Close()
			res, err := st.VCLog(cmd.Context(), limit)
			if err != nil {
				return doltstore.StoreErr(err)
			}
			if asJSON {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(res)
			}
			for _, c := range res.Commits {
				fmt.Fprintf(cmd.OutOrStdout(), "%s %s %s\n", shortHash(c.Hash), c.Date, c.Message)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "machine-readable output")
	cmd.Flags().IntVar(&limit, "limit", 20, "max commits (max 200)")
	return cmd
}

func newVcDiffCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "diff [commit-a] [commit-b]",
		Short: "Show table diff stats",
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := openRepoStore(cmd)
			if err != nil {
				return err
			}
			defer st.Close()
			var res doltstore.VCDiff
			switch len(args) {
			case 0:
				res, err = st.VCDiffWorking(cmd.Context())
			case 2:
				res, err = st.VCDiffRange(cmd.Context(), args[0], args[1])
			default:
				return clierr.Usage("fabrik: usage: fabrik vc diff [commit-a commit-b]")
			}
			if err != nil {
				var unknown *doltstore.ErrUnknownCommit
				if errors.As(err, &unknown) {
					return err
				}
				return doltstore.StoreErr(err)
			}
			if asJSON {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(res)
			}
			for _, t := range res.Tables {
				fmt.Fprintf(cmd.OutOrStdout(), "%s  +%d ~%d -%d\n", t.Name, t.RowsAdded, t.RowsModified, t.RowsRemoved)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "machine-readable output")
	return cmd
}

func newVcCommitCmd() *cobra.Command {
	var asJSON bool
	var message string
	cmd := &cobra.Command{
		Use:   "commit",
		Short: "Commit working set changes",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !cmd.Flags().Changed("message") {
				return clierr.Usage("fabrik: usage: fabrik vc commit -m <message>")
			}
			message = strings.TrimSpace(message)
			if message == "" {
				return clierr.Usage("fabrik: usage: fabrik vc commit -m <message>")
			}
			if utf8.RuneCountInString(message) > 500 {
				return fmt.Errorf("fabrik: commit message is too long")
			}
			st, err := openRepoStore(cmd)
			if err != nil {
				return err
			}
			defer st.Close()
			res, err := st.VCCommitMessage(cmd.Context(), message)
			if doltstore.IsNothingToCommit(err) {
				return fmt.Errorf("fabrik: nothing to commit")
			}
			if err != nil {
				return doltstore.StoreErr(err)
			}
			if asJSON {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(res)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "committed %s\n", shortHash(res.Hash))
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "machine-readable output")
	cmd.Flags().StringVarP(&message, "message", "m", "", "commit message")
	return cmd
}

func newVcBranchCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "branch [create <name>]",
		Short: "List or create Dolt branches",
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := openRepoStore(cmd)
			if err != nil {
				return err
			}
			defer st.Close()
			if len(args) >= 1 && args[0] == "create" {
				if len(args) != 2 {
					return clierr.Usage("fabrik: usage: fabrik vc branch create <name>")
				}
				if err := st.VCBranchCreate(cmd.Context(), args[1]); err != nil {
					return doltstore.StoreErr(err)
				}
				if asJSON {
					return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]string{"branch": args[1]})
				}
				fmt.Fprintf(cmd.OutOrStdout(), "created %s\n", args[1])
				return nil
			}
			if len(args) != 0 {
				return clierr.Usage("fabrik: usage: fabrik vc branch [create <name>]")
			}
			res, err := st.VCBranchList(cmd.Context())
			if err != nil {
				return doltstore.StoreErr(err)
			}
			if asJSON {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(res)
			}
			for _, b := range res.Branches {
				prefix := "  "
				if b == res.Current {
					prefix = "* "
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%s%s\n", prefix, b)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "machine-readable output")
	return cmd
}

func newVcCheckoutCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "checkout <name>",
		Short: "Switch Dolt branch",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return clierr.Usage("fabrik: usage: fabrik vc checkout <name>")
			}
			st, err := openRepoStore(cmd)
			if err != nil {
				return err
			}
			defer st.Close()
			if err := st.VCCheckout(cmd.Context(), args[0]); err != nil {
				var nf *doltstore.ErrBranchNotFound
				if errors.As(err, &nf) {
					return err
				}
				return doltstore.StoreErr(err)
			}
			if asJSON {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]string{"branch": args[0]})
			}
			fmt.Fprintf(cmd.OutOrStdout(), "checkout %s\n", args[0])
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "machine-readable output")
	return cmd
}

func newVcMergeCmd() *cobra.Command {
	var asJSON bool
	var strategy string
	cmd := &cobra.Command{
		Use:   "merge <branch>",
		Short: "Merge another Dolt branch",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return clierr.Usage("fabrik: usage: fabrik vc merge <branch>")
			}
			st, err := openRepoStore(cmd)
			if err != nil {
				return err
			}
			defer st.Close()
			res, err := st.VCMerge(cmd.Context(), args[0], strategy)
			if err != nil {
				return doltstore.StoreErr(err)
			}
			if asJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				if !res.Merged {
					_ = enc.Encode(res)
					return fmt.Errorf("fabrik: merge has conflicts")
				}
				return enc.Encode(res)
			}
			if !res.Merged {
				for _, c := range res.Conflicts {
					fmt.Fprintln(cmd.OutOrStdout(), c)
				}
				return fmt.Errorf("fabrik: merge has conflicts")
			}
			fmt.Fprintf(cmd.OutOrStdout(), "merged %s\n", shortHash(res.Hash))
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "machine-readable output")
	cmd.Flags().StringVar(&strategy, "strategy", "ours", "ours or theirs on conflict")
	return cmd
}

func shortHash(h string) string {
	if len(h) > 12 {
		return h[:12]
	}
	return h
}
