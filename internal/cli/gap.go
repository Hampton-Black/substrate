package cli

import (
	"github.com/Hampton-Black/substrate/internal/core"
	"github.com/spf13/cobra"
)

func newGapCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gap",
		Short: "Manage capability gaps",
	}
	cmd.AddCommand(gapAddCmd())
	return cmd
}

func gapAddCmd() *cobra.Command {
	var podID, category, description string
	var priority int
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Register a capability gap",
		RunE: func(cmd *cobra.Command, _ []string) error {
			defer closeConn()
			gap, err := svc.AddGap(cmd.Context(), core.AddGapInput{
				PodID:       podID,
				Category:    core.GapCategory(category),
				Description: description,
				Priority:    priority,
			})
			if err != nil {
				return err
			}
			return printJSON(gap)
		},
	}
	cmd.Flags().StringVar(&podID, "pod", "", "pod id")
	cmd.Flags().StringVar(&category, "category", "", "gap category")
	cmd.Flags().StringVar(&description, "description", "", "gap description")
	cmd.Flags().IntVar(&priority, "priority", 3, "priority 1 (high) to 5 (low)")
	_ = cmd.MarkFlagRequired("pod")
	_ = cmd.MarkFlagRequired("category")
	_ = cmd.MarkFlagRequired("description")
	return cmd
}
