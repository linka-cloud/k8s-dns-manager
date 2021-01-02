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
