package main

import (
	_ "go.linka.cloud/k8s/dns/pkg/provider/coredns"
	_ "go.linka.cloud/k8s/dns/pkg/provider/libdns/cloudflare"
	_ "go.linka.cloud/k8s/dns/pkg/provider/libdns/hetzner"
	_ "go.linka.cloud/k8s/dns/pkg/provider/libdns/ovh"
	_ "go.linka.cloud/k8s/dns/pkg/provider/libdns/scaleway"
)
