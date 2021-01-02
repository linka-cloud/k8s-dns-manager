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
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/ryanuber/columnize"
	"github.com/spf13/cobra"
	client2 "sigs.k8s.io/controller-runtime/pkg/client"

	"go.linka.cloud/k8s/dns/api/v1alpha1"
	"go.linka.cloud/k8s/dns/pkg/ptr"
)

var (
	quiet   = false
	ListCmd = &cobra.Command{
		Use:          "list",
		Short:        "list DNSRecords",
		Aliases:      []string{"ls", "l"},
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			var l v1alpha1.DNSRecordList
			if err := client.List(context.Background(), &l, client2.InNamespace(ns)); err != nil {
				return err
			}
			if len(l.Items) == 0 {
				fmt.Printf("No resources found in %s namespace.\n", ns)
				return nil
			}
			if quiet {
				for _, v := range l.Items {
					fmt.Printf("%s/%s\n", v.Namespace, v.Name)
				}
				return nil
			}
			output := []string{
				"NAME  | NAMESPACE | ACTIVE | RECORD | TTL | | TYPE | VALUE",
			}
			for _, v := range l.Items {
				parts := strings.Split(v.Status.Record, "\t")
				parts = append(append([]string{v.Name, ns, strconv.FormatBool(ptr.ToBool(v.Status.Active))}, parts...))
				output = append(output, strings.Join(parts, " | "))
			}
			result := columnize.SimpleFormat(output)
			fmt.Println(result)
			return nil
		},
	}
)

func init() {
	RootCmd.AddCommand(ListCmd)
	ListCmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "display only names")
	configFlags.AddFlags(ListCmd.Flags())
}
