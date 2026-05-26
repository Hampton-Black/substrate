package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/Hampton-Black/substrate/internal/core"
	"github.com/spf13/cobra"
)

func newGapCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gap",
		Short: "Manage capability gaps",
	}
	cmd.AddCommand(gapAddCmd())
	cmd.AddCommand(gapListCmd())
	cmd.AddCommand(gapAcknowledgeCmd())
	cmd.AddCommand(gapResolveCmd())
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

func gapListCmd() *cobra.Command {
	var podID, status string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List capability gaps",
		RunE: func(cmd *cobra.Command, _ []string) error {
			defer closeConn()

			f := core.GapFilters{}
			if podID != "" {
				f.PodID = &podID
			}
			if status == "all" {
				// no status filter
			} else if status != "" {
				st := core.GapStatus(status)
				f.Status = &st
			} else {
				f.Statuses = []core.GapStatus{core.GapOpen, core.GapAcknowledged}
			}

			gaps, err := svc.ListCapabilityGaps(cmd.Context(), f)
			if err != nil {
				return err
			}
			return printGapTable(gaps)
		},
	}
	cmd.Flags().StringVar(&podID, "pod", "", "filter by pod id")
	cmd.Flags().StringVar(&status, "status", "", "filter by status (default: open,acknowledged; use 'all' for everything)")
	return cmd
}

func gapAcknowledgeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "acknowledge <id>",
		Short: "Acknowledge a capability gap",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			defer closeConn()
			gap, err := svc.AcknowledgeGap(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return printJSON(gap)
		},
	}
}

func gapResolveCmd() *cobra.Command {
	var ref string
	cmd := &cobra.Command{
		Use:   "resolve <id>",
		Short: "Resolve a capability gap",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			defer closeConn()
			gap, err := svc.ResolveGap(cmd.Context(), args[0], ref)
			if err != nil {
				return err
			}
			return printJSON(gap)
		},
	}
	cmd.Flags().StringVar(&ref, "ref", "", "resolution reference")
	return cmd
}

func printGapTable(gaps []core.CapabilityGap) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tCATEGORY\tPRIORITY\tFREQUENCY\tSTATUS\tDESCRIPTION")
	for _, g := range gaps {
		id := g.ID
		if len(id) > 8 {
			id = id[:8]
		}
		desc := g.Description
		if len(desc) > 60 {
			desc = desc[:60]
		}
		fmt.Fprintf(w, "%s\t%s\t%d\t%d\t%s\t%s\n",
			id, g.Category, g.Priority, g.Frequency, g.Status, desc)
	}
	if err := w.Flush(); err != nil {
		return err
	}
	if len(gaps) == 0 {
		fmt.Fprintln(os.Stderr, "no gaps found")
	}
	return nil
}
