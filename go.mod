module go.linka.cloud/k8s/dns

go 1.13

require (
	github.com/coredns/caddy v1.1.0
	github.com/coredns/coredns v1.8.0
	github.com/go-logr/logr v0.3.0
	github.com/miekg/dns v1.1.35
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.2
	github.com/ryanuber/columnize v2.1.2+incompatible
	github.com/sirupsen/logrus v1.6.0
	github.com/spf13/cobra v1.1.1
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.6.1
	github.com/weppos/publicsuffix-go v0.13.0
	go.uber.org/multierr v1.5.0
	go.uber.org/zap v1.15.0
	k8s.io/api v0.20.1
	k8s.io/apimachinery v0.20.1
	k8s.io/cli-runtime v0.20.1
	k8s.io/client-go v0.20.1
	sigs.k8s.io/controller-runtime v0.7.0
	sigs.k8s.io/yaml v1.2.0
)
