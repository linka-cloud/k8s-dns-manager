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

package ovh

import (
	"fmt"
	"os"

	"github.com/libdns/ovh"

	"go.linka.cloud/k8s/dns/pkg/provider"
	"go.linka.cloud/k8s/dns/pkg/provider/libdns"
)

const (
	EndpointEnv    = "OVH_ENDPOINT"
	AppKeyEnv      = "OVH_APPLICATION_KEY"
	AppSecretEnv   = "OVH_APPLICATION_SECRET"
	ConsumerKeyEnv = "OVH_CONSUMER_KEY"
)

func init() {
	provider.Register("ovh", func() (provider.Provider, error) {
		endpoint := os.Getenv(EndpointEnv)
		if endpoint == "" {
			return nil, fmt.Errorf("empty %s environment variable", EndpointEnv)
		}
		appKey := os.Getenv(AppKeyEnv)
		if appKey == "" {
			return nil, fmt.Errorf("empty %s environment variable", AppKeyEnv)
		}
		appSecret := os.Getenv(AppSecretEnv)
		if appSecret == "" {
			return nil, fmt.Errorf("empty %s environment variable", AppSecretEnv)
		}
		consumerKey := os.Getenv(ConsumerKeyEnv)
		if consumerKey == "" {
			return nil, fmt.Errorf("empty %s environment variable", ConsumerKeyEnv)
		}
		p := &ovh.Provider{
			Endpoint:          endpoint,
			ApplicationKey:    appKey,
			ApplicationSecret: appSecret,
			ConsumerKey:       consumerKey,
		}
		return libdns.New("ovh", p), nil
	})
}
