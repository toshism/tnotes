package cmd

import (
	"github.com/spf13/cobra"
	"github.com/tosh/tnotes/internal/mcp"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Run as MCP server",
	Long: `Runs tnotes as an MCP (Model Context Protocol) server.

This mode allows AI assistants like Claude to interact with your notes
directly via the MCP protocol over stdio.

Configure in Claude Code settings:
{
  "mcpServers": {
    "tnotes": {
      "command": "tnotes",
      "args": ["mcp", "--dir", "/path/to/notes"]
    }
  }
}`,
	RunE: func(cmd *cobra.Command, args []string) error {
		server := mcp.NewServer()
		return server.Run()
	},
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}
