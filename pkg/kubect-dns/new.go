package kubect_dns

import (
	"fmt"
	"os"

	"github.com/miekg/dns"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"

	"go.linka.cloud/k8s/dns/pkg/record"
)

var (
	NewCmd = &cobra.Command{
		Use:   "create [record]",
		Short: "create a DNSRecord from bind record format and print it to stdout",
		Example: `
	kubectl dns create 'dns.google.com. IN A 8.8.8.8' | kubectl apply -f -`,
		Aliases:      []string{"new", "add"},
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			rr, err := dns.NewRR(args[0])
			if err != nil {
				return fmt.Errorf("invalid record: '%s': %v", args[0], err)
			}
			if rr == nil {
				return fmt.Errorf("invalid record: '%s'", args[0])
			}
			r := record.FromRR(rr)
			r.Namespace = ns
			b, err := yaml.Marshal(r)
			if err != nil {
				return err
			}
			_, err = os.Stdout.Write(b)
			return err
		},
	}
)

func init() {
	RootCmd.AddCommand(NewCmd)
	configFlags.AddFlags(NewCmd.Flags())
}
