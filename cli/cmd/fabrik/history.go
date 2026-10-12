package main

import (
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/spf13/cobra"
	"github.com/ur-wesley/fabrik/cli/internal/clierr"
)

var historyIDRe = regexp.MustCompile(`^(f|n)-[0-9a-f]{8,32}$`)

func newHistoryCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "history <id>",
		Short: "Show Dolt history for an issue or note",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return clierr.Usage("fabrik: usage: fabrik history <id>")
			}
			id := args[0]
			if !historyIDRe.MatchString(id) {
				return fmt.Errorf("fabrik: history expects f- or n- id")
			}
			st, err := openRepoStore(cmd)
			if err != nil {
				return err
			}
			defer st.Close()
			res, err := st.History(cmd.Context(), id)
			if err != nil {
				return err
			}
			if asJSON {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(res)
			}
			for _, ver := range res.Versions {
				hash, _ := ver["commit_hash"].(string)
				date, _ := ver["date"].(string)
				if hash == "" {
					hash, _ = ver["to_commit"].(string)
				}
				if date == "" {
					date, _ = ver["commit_date"].(string)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n", shortHash(hash), date)
				for k, v := range ver {
					if k == "commit_hash" || k == "date" || k == "to_commit" || k == "commit_date" {
						continue
					}
					fmt.Fprintf(cmd.OutOrStdout(), "  %s: %v\n", k, v)
				}
				fmt.Fprintln(cmd.OutOrStdout())
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "machine-readable output")
	return cmd
}
