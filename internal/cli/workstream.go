package cli

import (
	"github.com/Hampton-Black/substrate/internal/core"
	"github.com/spf13/cobra"
)

func newWorkstreamCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workstream",
		Short: "Manage workstreams",
	}
	cmd.AddCommand(workstreamAddCmd())
	cmd.AddCommand(workstreamStatusCmd())
	return cmd
}

func workstreamAddCmd() *cobra.Command {
	var podID, title, intent, branch string
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a workstream",
		RunE: func(cmd *cobra.Command, _ []string) error {
			defer closeConn()
			ws, err := svc.AddWorkstream(cmd.Context(), core.AddWorkstreamInput{
				PodID:  podID,
				Title:  title,
				Intent: intent,
				Branch: branch,
			})
			if err != nil {
				return err
			}
			return printJSON(ws)
		},
	}
	cmd.Flags().StringVar(&podID, "pod", "", "pod id")
	cmd.Flags().StringVar(&title, "title", "", "workstream title")
	cmd.Flags().StringVar(&intent, "intent", "", "workstream intent")
	cmd.Flags().StringVar(&branch, "branch", "", "git branch")
	_ = cmd.MarkFlagRequired("pod")
	_ = cmd.MarkFlagRequired("title")
	return cmd
}

func workstreamStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status <id> <status>",
		Short: "Update workstream status",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			defer closeConn()
			status := core.WorkstreamStatus(args[1])
			ws, err := svc.SetWorkstreamStatus(cmd.Context(), args[0], status)
			if err != nil {
				return err
			}
			return printJSON(ws)
		},
	}
}
