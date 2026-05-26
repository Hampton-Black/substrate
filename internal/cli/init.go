package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/Hampton-Black/substrate/internal/config"
	"github.com/Hampton-Black/substrate/internal/db"
	gitrepo "github.com/Hampton-Black/substrate/internal/git"
	"github.com/spf13/cobra"
)

func initCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize Substrate database and git document repository",
		RunE: func(cmd *cobra.Command, _ []string) error {
			defer closeConn()

			path, err := config.ResolvePath(configPath)
			if err != nil {
				return err
			}

			cfg = config.Default()
			if existing, err := config.Load(path); err == nil {
				cfg = existing
			}

			if err := config.Save(path, cfg); err != nil {
				return fmt.Errorf("write config: %w", err)
			}
			fmt.Fprintf(os.Stderr, "config: %s\n", path)

			if err := cfg.ExpandStoragePaths(); err != nil {
				return err
			}

			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			c, err := db.Open(ctx, cfg.Storage.DBPath)
			if err != nil {
				return fmt.Errorf("initialize database: %w", err)
			}
			_ = c.Close()
			fmt.Fprintf(os.Stderr, "database: %s\n", cfg.Storage.DBPath)

			if _, err := gitrepo.Init(cfg.Storage.GitRepoDir); err != nil {
				return fmt.Errorf("initialize git repo: %w", err)
			}
			fmt.Fprintf(os.Stderr, "git repo: %s\n", cfg.Storage.GitRepoDir)

			fmt.Fprintln(os.Stderr, "substrate initialized")
			return nil
		},
	}
}
