package kubect_dns

import (
	"fmt"
	"os"

	"github.com/miekg/dns"
	"github.com/spf13/cobra"
)

var (
	keyMode   string
	outputDir string
	DNSSecCmd = &cobra.Command{
		Use:   "dnssec",
		Short: "base command for DNSSEC operations",
	}
	DNSSecGenerateCmd = &cobra.Command{
		Use:          "generate [domain]",
		Short:        "generate DNSSEC public/private key pair for a domain",
		Aliases:      []string{"gen", "new"},
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if keyMode != "ksk" && keyMode != "zsk" {
				return fmt.Errorf("invalid key mode: %s", keyMode)
			}
			key := new(dns.DNSKEY)
			key.Hdr.Rrtype = dns.TypeDNSKEY
			key.Hdr.Name = dns.Fqdn(args[0])
			key.Hdr.Class = dns.ClassINET
			key.Hdr.Ttl = 14400
			if keyMode == "ksk" {
				key.Flags = 257
			} else {
				key.Flags = 256
			}
			key.Protocol = 3
			key.Algorithm = dns.ECDSAP256SHA256
			privkey, err := key.Generate(256)
			if err != nil {
				return err
			}
			name := fmt.Sprintf("K%s+%03d+%05d", dns.Fqdn(args[0]), key.Algorithm, key.KeyTag())
			if _, err := os.Stat(outputDir); err != nil {
				return err
			}
			if err = os.WriteFile(fmt.Sprintf("%s/%s.key", outputDir, name), []byte(key.String()), 0600); err != nil {
				return err
			}
			if err = os.WriteFile(fmt.Sprintf("%s/%s.private", outputDir, name), []byte(key.PrivateKeyString(privkey)), 0600); err != nil {
				return err
			}
			return nil
		},
	}
	DNSSecDSCmd = &cobra.Command{
		Use:          "ds",
		Short:        "generate DS record from DNSKEY",
		Aliases:      []string{"dsrecord"},
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			b, err := os.ReadFile(args[0])
			if err != nil {
				return err
			}
			rr, err := dns.NewRR(string(b))
			if err != nil {
				return err
			}
			k, ok := rr.(*dns.DNSKEY)
			if !ok {
				return fmt.Errorf("not a DNSKEY record")
			}
			if k.Flags != 257 {
				return fmt.Errorf("DNSKEY is not a KSK key")
			}
			ds := k.ToDS(dns.SHA256)
			fmt.Println(ds.String())
			return nil
		},
	}
)

func init() {
	DNSSecGenerateCmd.Flags().StringVarP(&outputDir, "output", "o", ".", "output directory for generated keys")
	DNSSecGenerateCmd.Flags().StringVarP(&keyMode, "mode", "k", "ksk", "key mode: ksk (key signing key) or zsk (zone signing key)")
	DNSSecCmd.AddCommand(DNSSecGenerateCmd, DNSSecDSCmd)
	RootCmd.AddCommand(DNSSecCmd)
}
