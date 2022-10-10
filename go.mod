module go.linka.cloud/k8s/dns

go 1.13

require (
	github.com/coredns/caddy v1.1.1
	github.com/coredns/coredns v1.10.0
	github.com/go-logr/logr v1.2.3
	github.com/libdns/cloudflare v0.1.1-0.20221006221909-9d3ab3c3cddd
	github.com/libdns/hetzner v0.0.1
	github.com/libdns/libdns v0.2.2-0.20221006221142-3ef90aee33fd
	github.com/libdns/ovh v0.0.1
	github.com/libdns/scaleway v0.1.0
	github.com/miekg/dns v1.1.50
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/gomega v1.19.0
	github.com/ryanuber/columnize v2.1.2+incompatible
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.4.0
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.8.0
	github.com/weppos/publicsuffix-go v0.13.0
	go.uber.org/multierr v1.6.0
	go.uber.org/zap v1.21.0
	golang.org/x/sys v0.0.0-20220928140112-f11e5e49a4ec // indirect
	k8s.io/api v0.25.0
	k8s.io/apimachinery v0.25.0
	k8s.io/cli-runtime v0.20.1
	k8s.io/client-go v0.25.0
	sigs.k8s.io/controller-runtime v0.13.0
	sigs.k8s.io/yaml v1.3.0
)

replace github.com/libdns/cloudflare => github.com/linka-cloud/cloudflare v0.0.0-20221007092341-f65599da475a

replace github.com/libdns/hetzner => github.com/linka-cloud/hetzner v0.0.2-0.20221007151052-8f56df8a2bdf
