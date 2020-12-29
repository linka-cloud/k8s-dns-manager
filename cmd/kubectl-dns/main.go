package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/miekg/dns"
	"github.com/ryanuber/columnize"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes/scheme"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	client2 "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	"go.linka.cloud/k8s/dns/api/v1alpha1"
	"go.linka.cloud/k8s/dns/pkg/record"
)

var (
	configFlags = genericclioptions.NewConfigFlags(true)
	client      client2.Client
	ns          string
	quiet       = false
	rootCmd     = cobra.Command{
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

	importCmd = &cobra.Command{
		Use:          "import [file]",
		Short:        "import dns bind file zone and print the DNSRecordList to stdout",
		Aliases:      []string{"convert"},
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			i, err := os.Stat(args[0])
			if err != nil {
				return err
			}
			if i.IsDir() {
				return errors.New("input: expected a file, not a directory")
			}

			return parse(args[0], os.Stdout)
		},
	}
	newCmd = &cobra.Command{
		Use:          "create [record]",
		Short:        "create a DNSRecord from bind record format and print it to stdout",
		Aliases:      []string{"new", "add"},
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			rr, err := dns.NewRR(args[0])
			if err != nil {
				return fmt.Errorf("invalid record: '%s': %v", args[0], err)
			}
			r := record.FromRR(rr)
			r.Namespace = ns
			b, err := yaml.Marshal(r)
			if err != nil {
				return err
			}
			_, err = os.Stdout.Write(b)
			return err
		},
	}
	listCmd = &cobra.Command{
		Use:          "list",
		Short:        "list DNSRecords",
		Aliases:      []string{"ls", "l"},
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			var l v1alpha1.DNSRecordList
			if err := client.List(context.Background(), &l, client2.InNamespace(ns)); err != nil {
				return err
			}
			if len(l.Items) == 0 {
				fmt.Printf("No resources found in %s namespace.\n", ns)
				return nil
			}
			if quiet {
				for _, v := range l.Items {
					fmt.Printf("%s/%s\n", v.Namespace, v.Name)
				}
				return nil
			}
			output := []string{
				"NAME  | NAMESPACE | ACTIVE | RECORD | TTL | | TYPE | VALUE",
			}
			for _, v := range l.Items {
				parts := strings.Split(v.Status.Record, "\t")
				parts = append(append([]string{v.Name, ns, strconv.FormatBool(v.Status.Active)}, parts...))
				output = append(output, strings.Join(parts, " | "))
			}
			result := columnize.SimpleFormat(output)
			fmt.Println(result)
			return nil
		},
	}
	activateCmd = &cobra.Command{
		Use:          "activate [record-name]",
		Short:        "active DNSRecord",
		Aliases:      []string{"enable", "on"},
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			r := v1alpha1.DNSRecord{}
			if err := client.Get(context.Background(), client2.ObjectKey{Namespace: ns, Name: args[0]}, &r); err != nil {
				return err
			}
			if r.Spec.Active == nil || *r.Spec.Active {
				return nil
			}
			active := true
			r.Spec.Active = &active
			if err := client.Update(context.Background(), &r); err != nil {
				return err
			}
			return nil
		},
	}
	deactivateCmd = &cobra.Command{
		Use:          "deactivate [record-name]",
		Short:        "de-activate DNSRecord",
		Aliases:      []string{"disable", "off"},
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			r := v1alpha1.DNSRecord{}
			if err := client.Get(context.Background(), client2.ObjectKey{Namespace: ns, Name: args[0]}, &r); err != nil {
				return err
			}
			if r.Spec.Active != nil && !*r.Spec.Active {
				return nil
			}
			active := false
			r.Spec.Active = &active
			if err := client.Update(context.Background(), &r); err != nil {
				return err
			}
			return nil
		},
	}
)

func main() {
	flags := pflag.NewFlagSet("kubectl-dns", pflag.ExitOnError)
	pflag.CommandLine = flags
	configFlags.AddFlags(rootCmd.Flags())
	rootCmd.AddCommand(importCmd)
	configFlags.AddFlags(importCmd.Flags())
	rootCmd.AddCommand(newCmd)
	configFlags.AddFlags(newCmd.Flags())
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "display only names")
	configFlags.AddFlags(listCmd.Flags())
	rootCmd.AddCommand(activateCmd)
	configFlags.AddFlags(activateCmd.Flags())
	rootCmd.AddCommand(deactivateCmd)
	configFlags.AddFlags(deactivateCmd.Flags())
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	_ = clientgoscheme.AddToScheme(scheme.Scheme)

	_ = v1alpha1.AddToScheme(scheme.Scheme)
	// +kubebuilder:scaffold:scheme
}

func parse(file string, w io.Writer) error {
	if w == nil {
		return errors.New("writer cannot be nil")
	}
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()

	records := v1alpha1.DNSRecordList{
		TypeMeta: metav1.TypeMeta{
			APIVersion: record.APIVersion,
			Kind:       record.DNSRecordList,
		},
	}
	zp := dns.NewZoneParser(f, "", "")
	for r, ok := zp.Next(); ok; r, ok = zp.Next() {
		if r == nil {
			continue
		}
		logrus.Info(r)
		rec := record.FromRR(r)
		rec.Namespace = ns
		records.Items = append(records.Items, rec)
	}

	b, err := yaml.Marshal(records)
	if err != nil {
		return err
	}
	if _, err := w.Write(b); err != nil {
		return err
	}
	return nil
}
