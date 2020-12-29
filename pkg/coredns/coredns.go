// Package coremain contains the functions for starting CoreDNS.
package coredns

import (
	"log"
	"os"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/test"

	_ "go.linka.cloud/k8s/dns/pkg/coredns/k8s_crds"
)

// Various CoreDNS constants.
const (
	CoreVersion = "1.8.0"
	coreName    = "CoreDNS"
	serverType  = "dns"

	defautConfig = `
.:53 {
    crds
    forward . 8.8.8.8
    log
    errors
}
`
)

var Config = defautConfig

func init() {
	dnsserver.Directives = append(dnsserver.Directives, "crds")
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
