package main

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/ur-wesley/fabrik/cli/internal/tmpl"
)

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list <skills|agents|all>",
		Short: "List embedded skills, agents, or all show keys",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			switch args[0] {
			case "skills":
				for _, n := range skillNames() {
					fmt.Fprintln(out, n)
				}
				return nil
			case "agents":
				for _, n := range agentNames() {
					fmt.Fprintln(out, n)
				}
				return nil
			case "all":
				for _, k := range showKeys {
					if k == "prompt" {
						fmt.Fprintln(out, "prompt plan")
						fmt.Fprintln(out, "prompt build")
						continue
					}
					fmt.Fprintln(out, k)
				}
				for _, n := range skillNames() {
					fmt.Fprintln(out, "skill "+n)
				}
				for _, n := range agentNames() {
					fmt.Fprintln(out, "agent "+n)
				}
				return nil
			}
			return fmt.Errorf("unknown group %q: valid groups: skills, agents, all", args[0])
		},
	}
}

func stems(paths []string) []string {
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		name := strings.TrimSuffix(path.Base(p), ".md")
		if name != "" {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

func skillNames() []string { return stems(tmpl.List("skills")) }

func agentNames() []string { return stems(tmpl.List("agents")) }
