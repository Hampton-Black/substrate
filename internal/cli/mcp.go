package cli

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/Hampton-Black/substrate/internal/mcp"
	"github.com/spf13/cobra"
)

func newMCPCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "MCP server commands",
	}
	cmd.AddCommand(mcpServeCmd())
	return cmd
}

func mcpServeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Run the Substrate MCP server on stdio",
		RunE: func(cmd *cobra.Command, _ []string) error {
			defer closeConn()

			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			return mcp.Serve(ctx, svc, logger)
		},
	}
}
