package main

import (
	"github.com/spf13/cobra"
	"github.com/ur-wesley/fabrik/cli/internal/mcpserver"
)

func newMCPCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mcp",
		Short: "Stdio MCP server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return mcpserver.Run(cmd.Context())
		},
	}
}
