package main

import (
	"os"

	kubect_dns "go.linka.cloud/k8s/dns/pkg/kubect-dns"
)


func main() {
	if err := kubect_dns.RootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
