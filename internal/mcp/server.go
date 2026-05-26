package mcp

import (
	"context"
	"log/slog"

	"github.com/Hampton-Black/substrate/internal/core"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Serve runs the Substrate MCP server on stdio until the context is canceled.
func Serve(ctx context.Context, svc *core.Service, log *slog.Logger) error {
	if log == nil {
		log = slog.Default()
	}

	server := sdkmcp.NewServer(&sdkmcp.Implementation{
		Name:    "substrate",
		Version: "0.1.0",
	}, nil)

	registerReadTools(server, svc, log)

	return server.Run(ctx, &sdkmcp.StdioTransport{})
}
