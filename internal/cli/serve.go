package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/Hampton-Black/substrate/internal/server"
	"github.com/spf13/cobra"
)

func serveCmd() *cobra.Command {
	var host string
	var port int
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the HTTP dashboard server",
		RunE: func(cmd *cobra.Command, _ []string) error {
			defer closeConn()

			srv, err := server.New(svc, logger)
			if err != nil {
				return err
			}

			addr := fmt.Sprintf("%s:%d", host, port)
			fmt.Fprintf(os.Stderr, "substrate serving at http://%s\n", addr)

			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			return srv.ListenAndServe(ctx, addr)
		},
	}
	cmd.Flags().StringVar(&host, "host", "127.0.0.1", "host to bind")
	cmd.Flags().IntVar(&port, "port", 7777, "port to listen on")
	return cmd
}
