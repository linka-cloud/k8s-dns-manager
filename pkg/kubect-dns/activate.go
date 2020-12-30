package kubect_dns

import (
	"context"

	"github.com/spf13/cobra"
)

var (
	ActivateCmd = &cobra.Command{
		Use:          "activate [record-name]",
		Short:        "active DNSRecord",
		Aliases:      []string{"enable", "on", "en"},
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := get(args[0])
			if err != nil {
				return err
			}
			if r.Spec.Active == nil || *r.Spec.Active {
				return nil
			}
			active := true
			r.Spec.Active = &active
			if err := client.Update(context.Background(), r); err != nil {
				return err
			}
			return nil
		},
	}
)

func init() {
	RootCmd.AddCommand(ActivateCmd)
	configFlags.AddFlags(ActivateCmd.Flags())
}
