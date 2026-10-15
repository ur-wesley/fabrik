package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"github.com/ur-wesley/fabrik/cli/internal/clierr"
	"github.com/ur-wesley/fabrik/cli/internal/doltstore"
	"github.com/ur-wesley/fabrik/cli/internal/export"
)

var issueIDRe = regexp.MustCompile(`^f-[0-9a-f]{8,32}$`)

var validIssueStatuses = map[string]bool{
	"open": true, "active": true, "blocked": true, "done": true,
}

func newIssueCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "issue",
		Short: "Repo issue store (Dolt)",
	}
	root.AddCommand(newIssueListCmd())
	root.AddCommand(newIssueReadyCmd())
	root.AddCommand(newIssueShowCmd())
	root.AddCommand(newIssueCreateCmd())
	root.AddCommand(newIssueClaimCmd())
	root.AddCommand(newIssueUnclaimCmd())
	root.AddCommand(newIssueCloseCmd())
	root.AddCommand(newIssueDepCmd())
	root.AddCommand(newIssueNoteCmd())
	return root
}

func issueStoreErr(id string, err error) error {
	if errors.Is(err, doltstore.ErrNotFound) {
		return fmt.Errorf("fabrik: issue %s not found", id)
	}
	return doltstore.StoreErr(err)
}

func publishIssueWrite(ctx context.Context, st *doltstore.Store, issueID string) error {
	return export.PublishAfterSave(ctx, st, issueID, "write")
}

func publishIssueNote(ctx context.Context, st *doltstore.Store, issueID string) error {
	return export.PublishAfterSave(ctx, st, issueID, "note")
}

func openIssueStore(cmd *cobra.Command) (*doltstore.Store, error) {
	return openRepoStore(cmd)
}

func newIssueListCmd() *cobra.Command {
	var asJSON bool
	var status string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List issues",
		RunE: func(cmd *cobra.Command, args []string) error {
			if status != "" && !validIssueStatuses[status] {
				return clierr.Usage("fabrik: status must be open, active, blocked, or done")
			}
			st, err := openIssueStore(cmd)
			if err != nil {
				return err
			}
			defer st.Close()
			issues, err := st.List(cmd.Context(), status)
			if err != nil {
				return doltstore.StoreErr(err)
			}
			if asJSON {
				if issues == nil {
					issues = []doltstore.Issue{}
				}
				enc := json.NewEncoder(cmd.OutOrStdout())
				return enc.Encode(issues)
			}
			for _, i := range issues {
				doltstore.PrintIssueLine(cmd.OutOrStdout(), i)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "machine-readable output")
	cmd.Flags().StringVar(&status, "status", "", "filter by status")
	return cmd
}

func newIssueCreateCmd() *cobra.Command {
	var asJSON bool
	var body, acceptance string
	var priority int
	var kind string
	cmd := &cobra.Command{
		Use:   "create [title]",
		Short: "Create an issue",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return clierr.Usage("fabrik: usage: fabrik issue create <title>")
			}
			title := strings.TrimSpace(strings.Join(args, " "))
			if title == "" {
				return clierr.Usage("fabrik: usage: fabrik issue create <title>")
			}
			st, err := openIssueStore(cmd)
			if err != nil {
				return err
			}
			defer st.Close()
			p := priority
			iss, err := st.Create(cmd.Context(), doltstore.CreateInput{
				Title: title, Body: body, Acceptance: acceptance, Priority: &p, Kind: kind,
			})
			if err != nil {
				return doltstore.StoreErr(err)
			}
			if err := publishIssueWrite(cmd.Context(), st, iss.ID); err != nil {
				return err
			}
			if asJSON {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(iss)
			}
			doltstore.PrintIssueLine(cmd.OutOrStdout(), iss)
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "machine-readable output")
	cmd.Flags().StringVar(&body, "body", "", "issue body")
	cmd.Flags().StringVar(&acceptance, "acceptance", "", "acceptance criteria")
	cmd.Flags().IntVar(&priority, "priority", 2, "priority 0-4")
	cmd.Flags().StringVar(&kind, "kind", "task", "issue kind")
	return cmd
}

func newIssueShowCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "show [id]",
		Short: "Show one issue",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 || !issueIDRe.MatchString(args[0]) {
				return clierr.Usage("fabrik: usage: fabrik issue show <id>")
			}
			id := args[0]
			st, err := openIssueStore(cmd)
			if err != nil {
				return err
			}
			defer st.Close()
			iss, err := st.Show(cmd.Context(), id)
			if errors.Is(err, doltstore.ErrNotFound) {
				return fmt.Errorf("fabrik: issue %s not found", id)
			}
			if err != nil {
				return doltstore.StoreErr(err)
			}
			if asJSON {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(iss)
			}
			doltstore.PrintShowText(cmd.OutOrStdout(), iss)
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "machine-readable output")
	return cmd
}

func newIssueReadyCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "ready",
		Short: "List ready issues",
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := openIssueStore(cmd)
			if err != nil {
				return err
			}
			defer st.Close()
			issues, err := st.Ready(cmd.Context())
			if err != nil {
				return doltstore.StoreErr(err)
			}
			if asJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				return enc.Encode(issues)
			}
			for _, i := range issues {
				doltstore.PrintIssueLine(cmd.OutOrStdout(), i)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "machine-readable output")
	return cmd
}

func newIssueClaimCmd() *cobra.Command {
	var asJSON bool
	var by string
	cmd := &cobra.Command{
		Use:   "claim [id]",
		Short: "Claim an issue",
		RunE: func(cmd *cobra.Command, args []string) error {
			var id string
			if len(args) == 1 {
				if !issueIDRe.MatchString(args[0]) {
					return clierr.Usage("fabrik: usage: fabrik issue claim [<id>]")
				}
				id = args[0]
			} else if len(args) != 0 {
				return clierr.Usage("fabrik: usage: fabrik issue claim [<id>]")
			}
			st, err := openIssueStore(cmd)
			if err != nil {
				return err
			}
			defer st.Close()
			iss, err := st.Claim(cmd.Context(), id, by)
			if err != nil {
				return issueStoreErr(id, err)
			}
			if err := publishIssueWrite(cmd.Context(), st, iss.ID); err != nil {
				return err
			}
			if asJSON {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(iss)
			}
			doltstore.PrintIssueLine(cmd.OutOrStdout(), iss)
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "machine-readable output")
	cmd.Flags().StringVar(&by, "by", "fabrik", "assignee")
	return cmd
}

func newIssueUnclaimCmd() *cobra.Command {
	var asJSON bool
	var by string
	var force bool
	cmd := &cobra.Command{
		Use:   "unclaim [id]",
		Short: "Release an active issue",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 || !issueIDRe.MatchString(args[0]) {
				return clierr.Usage("fabrik: usage: fabrik issue unclaim <id>")
			}
			st, err := openIssueStore(cmd)
			if err != nil {
				return err
			}
			defer st.Close()
			iss, err := st.Unclaim(cmd.Context(), args[0], by, force)
			if err != nil {
				return issueStoreErr(args[0], err)
			}
			if err := publishIssueWrite(cmd.Context(), st, iss.ID); err != nil {
				return err
			}
			if asJSON {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(iss)
			}
			doltstore.PrintIssueLine(cmd.OutOrStdout(), iss)
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "machine-readable output")
	cmd.Flags().StringVar(&by, "by", "fabrik", "assignee releasing the claim")
	cmd.Flags().BoolVar(&force, "force", false, "release even if assignee differs")
	return cmd
}

func newIssueCloseCmd() *cobra.Command {
	var asJSON bool
	var reason string
	var by string
	var force bool
	cmd := &cobra.Command{
		Use:   "close [id]",
		Short: "Close an issue",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 || !issueIDRe.MatchString(args[0]) {
				return clierr.Usage("fabrik: usage: fabrik issue close <id>")
			}
			id := args[0]
			if cmd.Flags().Changed("reason") && strings.TrimSpace(reason) == "" {
				return clierr.Usage("fabrik: reason is empty")
			}
			st, err := openIssueStore(cmd)
			if err != nil {
				return err
			}
			defer st.Close()
			res, err := st.CloseIssueBy(cmd.Context(), id, reason, by, force)
			if err != nil {
				return issueStoreErr(id, err)
			}
			if err := publishIssueWrite(cmd.Context(), st, res.Issue.ID); err != nil {
				return err
			}
			if asJSON {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(res.Issue)
			}
			doltstore.PrintIssueLine(cmd.OutOrStdout(), res.Issue)
			if res.NoteID != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "note %s\n", res.NoteID)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "machine-readable output")
	cmd.Flags().StringVar(&reason, "reason", "", "close reason (stored as note)")
	cmd.Flags().StringVar(&by, "by", "", "assignee closing (required when status is active)")
	cmd.Flags().BoolVar(&force, "force", false, "close even if assignee differs")
	return cmd
}

func newIssueDepCmd() *cobra.Command {
	var asJSON bool
	var kind string
	cmd := &cobra.Command{
		Use:   "dep [blocked] [blocker]",
		Short: "Add issue dependency",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 2 || !issueIDRe.MatchString(args[0]) || !issueIDRe.MatchString(args[1]) {
				return clierr.Usage("fabrik: usage: fabrik issue dep <blocked> <blocker>")
			}
			blocked, blocker := args[0], args[1]
			st, err := openIssueStore(cmd)
			if err != nil {
				return err
			}
			defer st.Close()
			res, err := st.Dep(cmd.Context(), blocked, blocker, kind)
			if err != nil {
				if errors.Is(err, doltstore.ErrNotFound) {
					if _, e := st.Get(cmd.Context(), blocked); errors.Is(e, doltstore.ErrNotFound) {
						return issueStoreErr(blocked, e)
					}
					return issueStoreErr(blocker, err)
				}
				return issueStoreErr(blocked, err)
			}
			if !res.AlreadyExists {
				if err := publishIssueWrite(cmd.Context(), st, res.Issue.ID); err != nil {
					return err
				}
			}
			if asJSON {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(res.Issue)
			}
			if res.AlreadyExists {
				fmt.Fprintf(cmd.OutOrStdout(), "%s already blocks %s\n", blocker, blocked)
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "%s blocks %s\n", blocker, blocked)
			}
			doltstore.PrintIssueLine(cmd.OutOrStdout(), res.Issue)
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "machine-readable output")
	cmd.Flags().StringVar(&kind, "kind", "blocks", "blocks or relates")
	return cmd
}

func newIssueNoteCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "note [id] [body]",
		Short: "Add issue note",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 || !issueIDRe.MatchString(args[0]) {
				return clierr.Usage("fabrik: usage: fabrik issue note <id> <body>")
			}
			id := args[0]
			body := strings.Join(args[1:], " ")
			st, err := openIssueStore(cmd)
			if err != nil {
				return err
			}
			defer st.Close()
			note, err := st.Note(cmd.Context(), id, body)
			if err != nil {
				return issueStoreErr(id, err)
			}
			if err := publishIssueNote(cmd.Context(), st, id); err != nil {
				return err
			}
			if asJSON {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(note)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s  %s  %s\n", note.ID, note.IssueID, note.CreatedAt)
			fmt.Fprintf(cmd.OutOrStdout(), "%s\n", note.Body)
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "machine-readable output")
	return cmd
}
