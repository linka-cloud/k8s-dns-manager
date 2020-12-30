package kubect_dns

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes/scheme"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	client2 "sigs.k8s.io/controller-runtime/pkg/client"

	"go.linka.cloud/k8s/dns/api/v1alpha1"
)

var (
	configFlags = genericclioptions.NewConfigFlags(true)
	client      client2.Client
	ns          string
	RootCmd     = cobra.Command{
		Use:          "dns",
		Short:        "dns root command",
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			ns, _, _ = configFlags.ToRawKubeConfigLoader().Namespace()
			if ns == "" {
				ns = "default"
			}
			conf, err := configFlags.ToRESTConfig()
			if err != nil {
				return err
			}
			client, err = client2.New(conf, client2.Options{})
			if err != nil {
				return err
			}
			return nil
		},
	}
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme.Scheme)
	_ = v1alpha1.AddToScheme(scheme.Scheme)
	flags := pflag.NewFlagSet("kubectl-dns", pflag.ExitOnError)
	pflag.CommandLine = flags
}
