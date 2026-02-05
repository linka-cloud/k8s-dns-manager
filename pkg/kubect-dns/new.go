/*
Copyright 2020 The Linka Cloud Team.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package kubect_dns

import (
	"fmt"
	"io"
	"os"

	"github.com/miekg/dns"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"

	"go.linka.cloud/k8s/dns/pkg/record"
)

var (
	name   string
	NewCmd = &cobra.Command{
		Use:   "create [record]",
		Short: "create a DNSRecord from bind record format and print it to stdout",
		Example: `
	kubectl dns create 'dns.google.com. IN A 8.8.8.8' | kubectl apply -f -`,
		Aliases:      []string{"new", "add"},
		Args:         cobra.MaximumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			var rec string
			if len(args) == 0 {
				b, err := io.ReadAll(cmd.InOrStdin())
				if err != nil {
					return err
				}
				rec = string(b)
			} else {
				rec = args[0]
			}
			rr, err := dns.NewRR(rec)
			if err != nil {
				return fmt.Errorf("invalid record: '%s': %v", rec, err)
			}
			if rr == nil {
				return fmt.Errorf("invalid record: '%s'", rec)
			}
			r := record.FromRR(rr)
			if name == "" {
				if dns.Fqdn(rr.Header().Name) == "." || rr.Header().Name == "" {
					r.Name = "root" + r.Name
				}
			} else {
				r.Name = name
			}
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
	NewCmd.Flags().StringVar(&name, "name", "", "name of the DNSRecord resource")
	RootCmd.AddCommand(NewCmd)
}
