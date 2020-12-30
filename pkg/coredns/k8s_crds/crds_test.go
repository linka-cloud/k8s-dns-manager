package crds

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/coredns/coredns/plugin/pkg/dnstest"
	"github.com/coredns/coredns/plugin/test"
	"github.com/miekg/dns"
)

var miekAuth = []dns.RR{
	test.NS("miek.nl.	1800	IN	NS	ext.ns.whyscream.net."),
	test.NS("miek.nl.	1800	IN	NS	linode.atoom.net."),
	test.NS("miek.nl.	1800	IN	NS	ns-ext.nlnetlabs.nl."),
	test.NS("miek.nl.	1800	IN	NS	omval.tednet.nl."),
}

var tests = []test.Case{
	{
		Qname: "www.miek.nl.", Qtype: dns.TypeA,
		Answer: []dns.RR{
			test.A("a.miek.nl.	1800	IN	A	139.162.196.78"),
			test.CNAME("www.miek.nl.	1800	IN	CNAME	a.miek.nl."),
		},
		Ns: miekAuth,
	},
	{
		Qname: "www.miek.nl.", Qtype: dns.TypeAAAA,
		Answer: []dns.RR{
			test.AAAA("a.miek.nl.	1800	IN	AAAA	2a01:7e00::f03c:91ff:fef1:6735"),
			test.CNAME("www.miek.nl.	1800	IN	CNAME	a.miek.nl."),
		},
		Ns: miekAuth,
	},
	{
		Qname: "miek.nl.", Qtype: dns.TypeSOA,
		Answer: []dns.RR{
			test.SOA("miek.nl.	1800	IN	SOA	linode.atoom.net. miek.miek.nl. 1282630057 14400 3600 604800 14400"),
		},
		Ns: miekAuth,
	},
	{
		Qname: "miek.nl.", Qtype: dns.TypeAAAA,
		Answer: []dns.RR{
			test.AAAA("miek.nl.	1800	IN	AAAA	2a01:7e00::f03c:91ff:fef1:6735"),
		},
		Ns: miekAuth,
	},
	{
		Qname: "mIeK.NL.", Qtype: dns.TypeAAAA,
		Answer: []dns.RR{
			test.AAAA("miek.nl.	1800	IN	AAAA	2a01:7e00::f03c:91ff:fef1:6735"),
		},
		Ns: miekAuth,
	},
	{
		Qname: "miek.nl.", Qtype: dns.TypeMX,
		Answer: []dns.RR{
			test.MX("miek.nl.	1800	IN	MX	1 aspmx.l.google.com."),
			test.MX("miek.nl.	1800	IN	MX	10 aspmx2.googlemail.com."),
			test.MX("miek.nl.	1800	IN	MX	10 aspmx3.googlemail.com."),
			test.MX("miek.nl.	1800	IN	MX	5 alt1.aspmx.l.google.com."),
			test.MX("miek.nl.	1800	IN	MX	5 alt2.aspmx.l.google.com."),
		},
		Ns: miekAuth,
	},
	{
		Qname: "a.miek.nl.", Qtype: dns.TypeSRV,
		Ns: []dns.RR{
			test.SOA("miek.nl.	1800	IN	SOA	linode.atoom.net. miek.miek.nl. 1282630057 14400 3600 604800 14400"),
		},
	},
	{
		Qname: "b.miek.nl.", Qtype: dns.TypeA,
		Rcode: dns.RcodeNameError,
		Ns: []dns.RR{
			test.SOA("miek.nl.	1800	IN	SOA	linode.atoom.net. miek.miek.nl. 1282630057 14400 3600 604800 14400"),
		},
	},
	{
		Qname: "srv.miek.nl.", Qtype: dns.TypeSRV,
		Answer: []dns.RR{
			test.SRV("srv.miek.nl.	1800	IN	SRV	10 10 8080  a.miek.nl."),
		},
		Extra: []dns.RR{
			test.A("a.miek.nl.	1800	IN	A       139.162.196.78"),
			test.AAAA("a.miek.nl.	1800	IN	AAAA	2a01:7e00::f03c:91ff:fef1:6735"),
		},
		Ns: miekAuth,
	},
	{
		Qname: "mx.miek.nl.", Qtype: dns.TypeMX,
		Answer: []dns.RR{
			test.MX("mx.miek.nl.	1800	IN	MX	10 a.miek.nl."),
		},
		Extra: []dns.RR{
			test.A("a.miek.nl.	1800	IN	A       139.162.196.78"),
			test.AAAA("a.miek.nl.	1800	IN	AAAA	2a01:7e00::f03c:91ff:fef1:6735"),
		},
		Ns: miekAuth,
	},
}

const dbMiekNL = `
$TTL    30M
$ORIGIN miek.nl.
@       IN      SOA     linode.atoom.net. miek.miek.nl. (
                             1282630057 ; Serial
                             4H         ; Refresh
                             1H         ; Retry
                             7D         ; Expire
                             4H )       ; Negative Cache TTL
                IN      NS      linode.atoom.net.
                IN      NS      ns-ext.nlnetlabs.nl.
                IN      NS      omval.tednet.nl.
                IN      NS      ext.ns.whyscream.net.

                IN      MX      1  aspmx.l.google.com.
                IN      MX      5  alt1.aspmx.l.google.com.
                IN      MX      5  alt2.aspmx.l.google.com.
                IN      MX      10 aspmx2.googlemail.com.
                IN      MX      10 aspmx3.googlemail.com.

		IN      A       139.162.196.78
		IN      AAAA    2a01:7e00::f03c:91ff:fef1:6735

a               IN      A       139.162.196.78
                IN      AAAA    2a01:7e00::f03c:91ff:fef1:6735
www             IN      CNAME   a
archive         IN      CNAME   a

srv		IN	SRV     10 10 8080 a.miek.nl.
mx		IN	MX      10 a.miek.nl.`

func TestCRDS(t *testing.T) {
	records := make(map[string]dns.RR)
	zp := dns.NewZoneParser(strings.NewReader(dbMiekNL), "", "")
	for r, ok := zp.Next(); ok; r, ok = zp.Next() {
		if r == nil {
			continue
		}
		records[fmt.Sprintf("%s", r.String())] = r
	}
	prov := &provider{records: records}
	if err := prov.sync(); err != nil {
		t.Fatal(err)
	}
	p := &CRDS{provider: prov}
	p.Next = test.NextHandler(dns.RcodeSuccess, nil)
	ctx := context.TODO()
	for i, tc := range tests {
		r := tc.Msg()
		w := dnstest.NewRecorder(&test.ResponseWriter{})

		_, err := p.ServeDNS(ctx, w, r)
		if err != tc.Error {
			t.Errorf("Test %d expected no error, got %v", i, err)
			return
		}
		if tc.Error != nil {
			continue
		}

		resp := w.Msg

		if resp == nil {
			t.Fatalf("Test %d, got nil message and no error for %q", i, r.Question[0].Name)
		}
		if err = test.SortAndCheck(resp, tc); err != nil {
			t.Errorf("Test %d: %v", i, err)
		}
	}
}
