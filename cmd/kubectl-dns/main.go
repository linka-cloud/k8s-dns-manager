package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"

	"github.com/miekg/dns"
	"github.com/ryanuber/columnize"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	client2 "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/yaml"

	"go.linka.cloud/k8s/dns/api/v1alpha1"
)

const (
	APIVersion    = "dns.linka.cloud/v1alpha1"
	DNSRecordList = "DNSRecordList"
	DNSRecord     = "DNSRecord"
)

var (
	rootCmd = cobra.Command{
		Use:   "dns",
		Short: "dns root command",
	}

	importCmdOut string
	importCmd    = &cobra.Command{
		Use:     "import [file]",
		Short:   "import dns bind file zone and print the DNSRecordList to stdout",
		Aliases: []string{"convert"},
		Args:    cobra.ExactArgs(1),
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
		Use:     "create [record]",
		Short:   "create a DNSRecord from bind record format and print it to stdout",
		Aliases: []string{"new", "add"},
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rr, err := dns.NewRR(args[0])
			if err != nil {
				return fmt.Errorf("invalid record: '%s': %v", args[0], err)
			}
			r := newDNSRecord(rr)
			b, err := yaml.Marshal(r)
			if err != nil {
				return err
			}
			_, err = os.Stdout.Write(b)
			return err
		},
	}
	listCmd = &cobra.Command{
		Use:     "list",
		Short:   "list DNSRecords",
		Aliases: []string{"ls", "l"},
		RunE: func(cmd *cobra.Command, args []string) error {
			conf, err := config.GetConfig()
			if err != nil {
			    return err
			}
			client, err := client2.New(conf, client2.Options{})
			if err != nil {
			    return err
			}
			var l v1alpha1.DNSRecordList
			if err := client.List(context.Background(), &l); err != nil {
				return err
			}
			output := []string{
				"NAME | TTL | | TYPE | VALUE",
			}
			for _, v := range l.Items {
				parts := strings.Split(v.Status.Record, "\t")
				output = append(output, strings.Join(parts, " | "))
			}
			result := columnize.SimpleFormat(output)
			fmt.Println(result)
			return nil
		},
	}
)

func main() {
	rootCmd.AddCommand(importCmd)
	rootCmd.AddCommand(newCmd)
	rootCmd.AddCommand(listCmd)
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
			APIVersion: APIVersion,
			Kind:       DNSRecordList,
		},
	}
	zp := dns.NewZoneParser(f, "", "")
	for r, ok := zp.Next(); ok; r, ok = zp.Next() {
		if r == nil {
			continue
		}
		logrus.Info(r)
		records.Items = append(records.Items, newDNSRecord(r))
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

func newDNSRecord(r dns.RR) v1alpha1.DNSRecord {
	name := strings.TrimSuffix(r.Header().Name, ".")
	name = strings.Replace(name, ".", "-", -1)
	name = strings.Replace(name, "_", "", -1)
	name = strings.Replace(name, "*", "wildcard", -1)
	if r.Header().Rrtype != dns.TypeA {
		typeName := strings.ToLower(strings.TrimPrefix(reflect.TypeOf(r).String(), "*dns."))
		name = fmt.Sprintf("%s-%s", name, typeName)
	}
	record := v1alpha1.DNSRecord{
		TypeMeta: metav1.TypeMeta{
			APIVersion: APIVersion,
			Kind:       DNSRecord,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	var spec v1alpha1.DNSRecordSpec
	switch rr := r.(type) {
	case *dns.A:
		spec = v1alpha1.DNSRecordSpec{
			Name:  rr.Hdr.Name,
			Class: rr.Hdr.Class,
			Ttl:   rr.Hdr.Ttl,
			ARecord: &v1alpha1.ARecord{
				A: rr.A.String(),
			}}
	case *dns.CNAME:
		spec = v1alpha1.DNSRecordSpec{
			Name:   rr.Hdr.Name,
			Class:  rr.Hdr.Class,
			Ttl:    rr.Hdr.Ttl,
			Target: rr.Target,
		}
	case *dns.SRV:
		spec = v1alpha1.DNSRecordSpec{
			Name:   rr.Hdr.Name,
			Class:  rr.Hdr.Class,
			Ttl:    rr.Hdr.Ttl,
			Target: rr.Target,
			SRVRecord: &v1alpha1.SRVRecord{
				Priority: rr.Priority,
				Weight:   rr.Weight,
				Port:     rr.Port,
			}}
	case *dns.TXT:
		spec = v1alpha1.DNSRecordSpec{
			Name:  rr.Hdr.Name,
			Class: rr.Hdr.Class,
			Ttl:   rr.Hdr.Ttl,
			TXTRecord: &v1alpha1.TXTRecord{
				Txt: rr.Txt,
			}}
	case *dns.MX:
		spec = v1alpha1.DNSRecordSpec{
			Name:  rr.Hdr.Name,
			Class: rr.Hdr.Class,
			Ttl:   rr.Hdr.Ttl,
			MXRecord: &v1alpha1.MXRecord{
				Preference: rr.Preference,
				Mx:         rr.Mx,
			}}
	default:
		spec = v1alpha1.DNSRecordSpec{
			Name: rr.Header().Name,
			Raw:  rr.String(),
		}
	}
	record.Spec = spec
	return record
}
