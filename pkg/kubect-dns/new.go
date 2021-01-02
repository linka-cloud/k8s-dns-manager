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
