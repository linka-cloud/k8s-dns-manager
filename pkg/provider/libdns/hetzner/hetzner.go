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

package hetzner

import (
	"fmt"
	"os"

	"github.com/libdns/hetzner"

	"go.linka.cloud/k8s/dns/pkg/provider"
	"go.linka.cloud/k8s/dns/pkg/provider/libdns"
)

const (
	TokenEnv = "HETZNER_TOKEN"
)

func init() {
	provider.Register("hetzner", func() (provider.Provider, error) {
		t := os.Getenv(TokenEnv)
		if t == "" {
			return nil, fmt.Errorf("empty %s environment variable", TokenEnv)
		}
		p := &hetzner.Provider{
			AuthAPIToken: t,
		}
		// hetzner provider seems to have so issues with concurrent requests
		return libdns.NewSync("hetzner", p), nil
	})
}
