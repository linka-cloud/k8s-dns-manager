package kubect_dns

import (
	"errors"
	"os"

	"github.com/miekg/dns"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	"go.linka.cloud/k8s/dns/api/v1alpha1"
	"go.linka.cloud/k8s/dns/pkg/record"
)

var (
	ImportCmd = &cobra.Command{
		Use:   "import [file]",
		Short: "import dns bind file zone and print the DNSRecordList to stdout",
		Example: `
	kubectl dns import example.org | kubectl apply -f -`,
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
			rrs, err := parse(args[0])
			if err != nil {
				return err
			}
			b, err := yaml.Marshal(rrs)
			if err != nil {
				return err
			}
			if _, err := os.Stdout.Write(b); err != nil {
				return err
			}
			return nil
		},
	}
)

func init() {
	RootCmd.AddCommand(ImportCmd)
	configFlags.AddFlags(ImportCmd.Flags())
}

func parse(file string) (*v1alpha1.DNSRecordList, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	records := &v1alpha1.DNSRecordList{
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
	return records, nil
}
