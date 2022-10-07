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

	"github.com/miekg/dns"
	"github.com/spf13/cobra"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	client2 "sigs.k8s.io/controller-runtime/pkg/client"

	"go.linka.cloud/k8s/dns/api/v1alpha1"
)

var (
	DeactivateCmd = &cobra.Command{
		Use:          "deactivate [record-name]",
		Short:        "de-activate DNSRecord",
		Aliases:      []string{"disable", "off", "dis"},
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := get(args[0])
			if err != nil {
				return err
			}
			if r.Spec.Active != nil && !*r.Spec.Active {
				return nil
			}
			active := false
			r.Spec.Active = &active
			if err := client.Update(context.Background(), r); err != nil {
				return err
			}
			return nil
		},
	}
)

func get(name string) (*v1alpha1.DNSRecord, error) {
	r := v1alpha1.DNSRecord{}
	err := client.Get(context.Background(), client2.ObjectKey{Namespace: ns, Name: name}, &r)
	if err == nil {
		return &r, nil
	}
	if !apierrors.IsNotFound(err) {
		return nil, err
	}
	rr, err := dns.NewRR(name)
	if err != nil || rr == nil {
		return nil, fmt.Errorf("'%s' is not a DNSRecord or a valid dns record", name)
	}
	records := &v1alpha1.DNSRecordList{}
	if err := client.List(context.Background(), records); err != nil {
		return nil, err
	}
	rrs := rr.String()
	for _, v := range records.Items {
		if v.Status.Record == rrs {
			return &v, nil
		}
	}
	return nil, fmt.Errorf("'%s': record not found", name)
}

func init() {
	RootCmd.AddCommand(DeactivateCmd)
}
