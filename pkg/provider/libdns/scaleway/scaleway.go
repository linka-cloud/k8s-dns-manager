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

package scaleway

import (
	"fmt"
	"os"

	"github.com/libdns/scaleway"

	"go.linka.cloud/k8s/dns/pkg/provider"
	"go.linka.cloud/k8s/dns/pkg/provider/libdns"
)

const (
	SecretKeyEnv      = "SCALEWAY_SECRET_KEY"
	OrganizationIDEnv = "SCALEWAY_ORGANIZATION_ID"
)

func init() {
	provider.Register("scaleway", func() (provider.Provider, error) {
		k := os.Getenv(SecretKeyEnv)
		if k == "" {
			return nil, fmt.Errorf("empty %s environment variable", SecretKeyEnv)
		}
		o := os.Getenv(OrganizationIDEnv)
		if o == "" {
			return nil, fmt.Errorf("empty %s environment variable", OrganizationIDEnv)
		}
		p := &scaleway.Provider{
			SecretKey:      k,
			OrganizationID: o,
		}
		return libdns.New("scaleway", p), nil
	})
}
