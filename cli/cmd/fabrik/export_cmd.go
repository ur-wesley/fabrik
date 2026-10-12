package main

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/ur-wesley/fabrik/cli/internal/doltstore"
	"github.com/ur-wesley/fabrik/cli/internal/export"
)

func newExportCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export issue JSONL snapshot",
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := openRepoStore(cmd)
			if err != nil {
				return err
			}
			defer st.Close()
			count, err := export.Issues(cmd.Context(), st)
			if err != nil {
				return doltstore.StoreErr(err)
			}
			path := filepath.Join(".fabrik", "export", "issues.jsonl")
			if asJSON {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]interface{}{
					"issues":      count,
					"path_issues": path,
				})
			}
			fmt.Fprintf(cmd.OutOrStdout(), "exported %d issues\n", count)
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "machine-readable output")
	return cmd
}
