package cli

import (
	"context"
	"os"

	"github.com/pauloborges/orb-cli/internal/mcpserver"
	"github.com/spf13/cobra"
)

// NewRootCmd creates the orb-cli command tree.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "orb-cli",
		Short:         "Command-line tools for Orb",
		Long:          "orb-cli provides command-line tools for Orb.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(newMCPCmd(mcpserver.Run))
	return root
}

type runMCPServer func(context.Context, mcpserver.Config) error

func newMCPCmd(run runMCPServer) *cobra.Command {
	return &cobra.Command{
		Use:   "mcp",
		Short: "Run an Orb read-only MCP server over stdio",
		Long: `Start a local, read-only MCP server for Orb.

The server uses standard input and standard output for MCP messages. Do not
write log data to standard output.

Set ORB_LIVE_API_KEY and ORB_TEST_API_KEY before you run this command. The
command fails if either key is not set. Each MCP tool request must set its
environment to live or test. The server does not select an environment by
default.

Press Ctrl+C to stop the server. The command cancels active requests before it
exits.`,
		Example: `  ORB_LIVE_API_KEY=your-live-key ORB_TEST_API_KEY=your-test-key orb-cli mcp`,
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return run(cmd.Context(), mcpserver.Config{
				LiveAPIKey: os.Getenv(mcpserver.LiveAPIKeyEnv),
				TestAPIKey: os.Getenv(mcpserver.TestAPIKeyEnv),
			})
		},
	}
}
