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

// Package coredns contains the functions for starting CoreDNS.
package coredns

import (
	"log"
	"os"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/test"

	_ "go.linka.cloud/k8s/dns/pkg/coredns/k8s_dns"
)

// Various CoreDNS constants.
const (
	CoreVersion = "1.8.0"
	coreName    = "CoreDNS"
	serverType  = "dns"

	defautConfig = `
.:53 {
    k8s_dns
    forward . 8.8.8.8
    log
    errors
}
`
)

var Config = defautConfig

func init() {
	dnsserver.Directives = []string{
		"metadata",
		"cancel",
		"tls",
		"reload",
		"nsid",
		"bufsize",
		"root",
		"bind",
		"debug",
		"trace",
		"ready",
		"health",
		"pprof",
		"prometheus",
		"errors",
		"log",
		"dnstap",
		"dns64",
		"acl",
		"any",
		"chaos",
		"loadbalance",
		"cache",
		"rewrite",
		"dnssec",
		"autopath",
		"template",
		"transfer",
		"hosts",
		"route53",
		"azure",
		"clouddns",
		"k8s_dns",
		"k8s_external",
		"kubernetes",
		"file",
		"auto",
		"secondary",
		"etcd",
		"loop",
		"forward",
		"grpc",
		"erratic",
		"whoami",
		"on",
		"sign",
	}
}

// Run is CoreDNS's main() function.
func Run(config string) {
	if config != "" {
		Config = config
	}

	caddy.Quiet = true // don't show init stuff from caddy

	caddy.SetDefaultCaddyfileLoader("default", caddy.LoaderFunc(confLoader))

	caddy.AppName = coreName
	caddy.AppVersion = CoreVersion

	caddy.TrapSignals()

	log.SetOutput(os.Stdout)
	log.SetFlags(0) // Set to 0 because we're doing our own time, with timezone

	// Get Corefile input
	corefile, err := caddy.LoadCaddyfile(serverType)
	if err != nil {
		mustLogFatal(err)
	}

	// Start your engines
	instance, err := caddy.Start(corefile)
	if err != nil {
		mustLogFatal(err)
	}

	// Twiddle your thumbs
	instance.Wait()
}

// mustLogFatal wraps log.Fatal() in a way that ensures the
// output is always printed to stderr so the user can see it
// if the user is still there, even if the process log was not
// enabled. If this process is an upgrade, however, and the user
// might not be there anymore, this just logs to the process
// log and exits.
func mustLogFatal(args ...interface{}) {
	if !caddy.IsUpgrade() {
		log.SetOutput(os.Stderr)
	}
	log.Fatal(args...)
}

// confLoader loads the Caddyfile using the -conf flag.
func confLoader(_ string) (caddy.Input, error) {
	return test.NewInput(Config), nil
}
