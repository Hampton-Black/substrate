package cli

import (
	"encoding/json"
	"os"

	"github.com/Hampton-Black/substrate/internal/core"
	"github.com/spf13/cobra"
)

func newPodCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pod",
		Short: "Manage pods",
	}
	cmd.AddCommand(podRegisterCmd())
	cmd.AddCommand(podListCmd())
	return cmd
}

func podRegisterCmd() *cobra.Command {
	var name, owner string
	cmd := &cobra.Command{
		Use:   "register",
		Short: "Register a new pod",
		RunE: func(cmd *cobra.Command, _ []string) error {
			defer closeConn()
			pod, err := svc.RegisterPod(cmd.Context(), core.RegisterPodInput{
				Name:  name,
				Owner: owner,
			})
			if err != nil {
				return err
			}
			return printJSON(pod)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "pod display name (used to derive id)")
	cmd.Flags().StringVar(&owner, "owner", "", "human owner email")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("owner")
	return cmd
}

func podListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List registered pods",
		RunE: func(cmd *cobra.Command, _ []string) error {
			defer closeConn()
			pods, err := svc.ListPods(cmd.Context())
			if err != nil {
				return err
			}
			if pods == nil {
				pods = []core.Pod{}
			}
			return printJSON(pods)
		},
	}
}

func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
