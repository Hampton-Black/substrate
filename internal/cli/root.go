package cli

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"

	"github.com/Hampton-Black/substrate/internal/config"
	"github.com/Hampton-Black/substrate/internal/core"
	"github.com/Hampton-Black/substrate/internal/db"
	gitrepo "github.com/Hampton-Black/substrate/internal/git"
	"github.com/spf13/cobra"
)

var (
	configPath string
	cfg        config.Config
	conn       *sql.DB
	svc        *core.Service
	logger     *slog.Logger
)

// Execute runs the substrate CLI.
func Execute() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "substrate",
		Short: "Substrate — shared world model for developer+agent pods",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			if cmd.Name() == "init" {
				return nil
			}
			return loadRuntime(cmd.Context())
		},
	}

	root.PersistentFlags().StringVar(&configPath, "config", "", "path to config file (default ~/.substrate/config.yaml or SUBSTRATE_CONFIG)")

	root.AddCommand(initCmd())
	root.AddCommand(newPodCmd())
	root.AddCommand(newWorkstreamCmd())
	root.AddCommand(newGapCmd())
	root.AddCommand(newMCPCmd())

	return root
}

func loadRuntime(ctx context.Context) error {
	path, err := config.ResolvePath(configPath)
	if err != nil {
		return err
	}

	cfg, err = config.Load(path)
	if err != nil {
		return err
	}
	if err := cfg.ExpandStoragePaths(); err != nil {
		return err
	}

	logger = slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	conn, err = db.Open(ctx, cfg.Storage.DBPath)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}

	var git *gitrepo.Repo
	if _, err := os.Stat(cfg.Storage.GitRepoDir); err == nil {
		git = gitrepo.Open(cfg.Storage.GitRepoDir)
	}

	svc = core.NewService(conn, git, logger)
	return nil
}

func closeConn() {
	if conn != nil {
		_ = conn.Close()
		conn = nil
	}
}
