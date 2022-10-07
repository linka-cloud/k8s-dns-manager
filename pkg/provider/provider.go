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

package provider

import (
	"context"
	"errors"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	dnsv1alpha1 "go.linka.cloud/k8s/dns/api/v1alpha1"
)

var (
	providers = make(map[string]Factory)

	ErrProviderNotFound = errors.New("provider not found")
)

func Register(name string, factory Factory) {
	providers[name] = factory
}

func New(name string) (Provider, error) {
	factory, ok := providers[name]
	if !ok {
		return nil, fmt.Errorf("%s: %w", name, ErrProviderNotFound)
	}
	p, err := factory()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	return p, nil
}

type Factory func() (Provider, error)

type Provider interface {
	Reconcile(ctx context.Context, rec *dnsv1alpha1.DNSRecord) (ctrl.Result, bool, error)
}

type Func func(ctx context.Context, rec *dnsv1alpha1.DNSRecord) (ctrl.Result, bool, error)

func (f Func) Reconcile(ctx context.Context, rec *dnsv1alpha1.DNSRecord) (ctrl.Result, bool, error) {
	return f(ctx, rec)
}
