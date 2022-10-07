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

package main

import (
	"fmt"
	"net"
	"os"

	"github.com/spf13/cobra"
	zap2 "go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	dnsv1alpha1 "go.linka.cloud/k8s/dns/api/v1alpha1"
	"go.linka.cloud/k8s/dns/controllers"
	"go.linka.cloud/k8s/dns/pkg/coredns"
	"go.linka.cloud/k8s/dns/pkg/coredns/config"
	"go.linka.cloud/k8s/dns/pkg/provider"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")

	metricsAddr           string
	enableLeaderElection  bool
	enableWebhook         bool
	noDNSServer           bool
	dnsLog                bool
	dnsForward            []string
	dnsMetrics            bool
	dnsCache              int
	dnsAny                bool
	externalAddress       net.IP
	dnsVerificationServer net.IP

	dnsProvider string

	Root = &cobra.Command{
		Use:   "k8s-dns",
		Short: "k8s-dns is a DNS Controller allowing to manage DNS Records from within a Kubernetes cluster",
		Run: func(cmd *cobra.Command, args []string) {
			ctrl.SetLogger(zap.New(zap.UseDevMode(true), zap.StacktraceLevel(zap2.NewAtomicLevelAt(zapcore.FatalLevel))))

			mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
				Scheme:             scheme,
				MetricsBindAddress: metricsAddr,
				Port:               9443,
				LeaderElection:     enableLeaderElection,
				LeaderElectionID:   "aa75d9c6.linka.cloud",
			})
			if err != nil {
				setupLog.Error(err, "unable to start manager")
				os.Exit(1)
			}

			if dnsProvider == "" && noDNSServer {
				setupLog.Error(err, "no dns provider specified and no in-process dns server")
				os.Exit(1)
			}

			if externalAddress == nil {
				externalAddress = net.ParseIP("127.0.0.1")
			}

			if dnsVerificationServer == nil {
				dnsVerificationServer = net.ParseIP("1.1.1.1")
			}

			if !noDNSServer && dnsProvider != "" && dnsProvider != "coredns" {
				setupLog.Error(fmt.Errorf("dns provider must be coredns when running in-process dns server"), "invalid dns provider")
				os.Exit(1)
			}

			if dnsProvider == "" {
				dnsProvider = "coredns"
			}

			prov, err := provider.New(dnsProvider)
			if err != nil {
				setupLog.Error(err, "unable to create provider")
				os.Exit(1)
			}
			c := &controllers.DNSRecordReconciler{
				Client:                mgr.GetClient(),
				Log:                   ctrl.Log.WithName("controllers").WithName("DNSRecord"),
				Scheme:                mgr.GetScheme(),
				Provider:              prov,
				DNSVerificationServer: dnsVerificationServer.String() + ":53",
			}
			if err = c.SetupWithManager(mgr); err != nil {
				setupLog.Error(err, "unable to create controller", "controller", "DNSRecord")
				os.Exit(1)
			}

			if enableWebhook {
				setupLog.Info("registering webhook")
				if err = (&dnsv1alpha1.DNSRecord{}).SetupWebhookWithManager(mgr); err != nil {
					setupLog.Error(err, "unable to create webhook", "webhook", "DNSRecord")
					os.Exit(1)
				}
			}

			if !noDNSServer {
				c.DNSVerificationServer = "127.0.0.1:53"
				conf, err := config.Config{
					Forward:         dnsForward,
					Log:             dnsLog,
					Errors:          true,
					Cache:           dnsCache,
					Metrics:         dnsMetrics,
					Any:             dnsAny,
					ExternalAddress: externalAddress.String(),
				}.Render()
				setupLog.Info("coredns config", "corefile", conf)
				if err != nil {
					setupLog.Error(err, "failed to configure coredns server")
					os.Exit(1)
				}
				setupLog.Info("starting dns server")
				go coredns.Run(conf)
			}

			setupLog.Info("starting manager")
			if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
				setupLog.Error(err, "problem running manager")
				os.Exit(1)
			}
		},
	}
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)
	_ = dnsv1alpha1.AddToScheme(scheme)

	Root.Flags().StringVar(&metricsAddr, "metrics-addr", ":4299", "The address the metric endpoint binds to.")
	Root.Flags().BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	Root.Flags().BoolVar(&enableWebhook, "enable-webhook", false, "Enable the validation webhook")

	Root.Flags().IPVar(&dnsVerificationServer, "dns-verification-server", net.ParseIP("1.1.1.1"), "DNS server to use for verification")

	Root.Flags().StringVarP(&dnsProvider, "provider", "p", "coredns", "DNS provider to use")

	Root.Flags().BoolVar(&noDNSServer, "no-dns", false, "Do not run in process coredns server")
	Root.Flags().BoolVar(&dnsLog, "dns-log", false, "Enable coredns query logs")
	Root.Flags().StringSliceVar(&dnsForward, "dns-forward", nil, "Dns forward servers")
	Root.Flags().BoolVar(&dnsMetrics, "dns-metrics", false, "Enable coredns metrics on 0.0.0.0:9153")
	Root.Flags().BoolVar(&dnsAny, "dns-any", false, "Enable coredns 'any' plugin")
	Root.Flags().IntVar(&dnsCache, "dns-cache", 0, "Enable coredns cache with ttl (in seconds)")
	Root.Flags().IPVarP(&externalAddress, "external-address", "a", net.ParseIP("127.0.0.1"), "The external dns server address, e.g the loadbalancer service IP")
}

func main() {
	Root.Execute()
}
