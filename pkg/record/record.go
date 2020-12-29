package record

import (
	"errors"
	"fmt"
	"net"
	"reflect"
	"strings"

	"github.com/miekg/dns"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"go.linka.cloud/k8s/dns/api/v1alpha1"
)

const (
	APIVersion    = "dns.linka.cloud/v1alpha1"
	DNSRecordList = "DNSRecordList"
	DNSRecord     = "DNSRecord"
)

func FromRR(r dns.RR) v1alpha1.DNSRecord {
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
			A: &v1alpha1.ARecord{
				Name:   rr.Hdr.Name,
				Class:  rr.Hdr.Class,
				Ttl:    rr.Hdr.Ttl,
				Target: rr.A.String(),
			},
		}
	case *dns.CNAME:
		spec = v1alpha1.DNSRecordSpec{
			CNAME: &v1alpha1.CNAMERecord{
				Name:   rr.Hdr.Name,
				Class:  rr.Hdr.Class,
				Ttl:    rr.Hdr.Ttl,
				Target: rr.Target,
			},
		}
	case *dns.SRV:
		spec = v1alpha1.DNSRecordSpec{
			SRV: &v1alpha1.SRVRecord{
				Name:     rr.Hdr.Name,
				Class:    rr.Hdr.Class,
				Ttl:      rr.Hdr.Ttl,
				Target:   rr.Target,
				Priority: rr.Priority,
				Weight:   rr.Weight,
				Port:     rr.Port,
			},
		}
	case *dns.TXT:
		spec = v1alpha1.DNSRecordSpec{
			TXT: &v1alpha1.TXTRecord{
				Name:    rr.Hdr.Name,
				Class:   rr.Hdr.Class,
				Ttl:     rr.Hdr.Ttl,
				Targets: rr.Txt,
			},
		}
	case *dns.MX:
		spec = v1alpha1.DNSRecordSpec{
			MX: &v1alpha1.MXRecord{
				Name:       rr.Hdr.Name,
				Class:      rr.Hdr.Class,
				Ttl:        rr.Hdr.Ttl,
				Preference: rr.Preference,
				Target:     rr.Mx,
			},
		}
	default:
		spec = v1alpha1.DNSRecordSpec{
			Raw: rr.String(),
		}
	}
	record.Spec = spec
	return record
}

func ToRR(r v1alpha1.DNSRecord) (dns.RR, error) {
	switch {
	case r.Spec.A != nil:
		h := dns.RR_Header{
			Name:   r.Spec.A.Name,
			Rrtype: dns.TypeA,
			Class:  r.Spec.A.Class,
			Ttl:    r.Spec.A.Ttl,
		}
		ip := net.ParseIP(r.Spec.A.Target)
		if ip == nil {
			return nil, fmt.Errorf("invalid ip: %s", r.Spec.A.Target)
		}
		return &dns.A{Hdr: h, A: ip}, nil
	case r.Spec.TXT != nil:
		h := dns.RR_Header{
			Name:   r.Spec.TXT.Name,
			Rrtype: dns.TypeTXT,
			Class:  r.Spec.TXT.Class,
			Ttl:    r.Spec.TXT.Ttl,
		}
		if len(r.Spec.TXT.Targets) == 0 {
			return nil, errors.New("empty TXT record")
		}
		return &dns.TXT{Hdr: h, Txt: r.Spec.TXT.Targets}, nil
	case r.Spec.SRV != nil:
		h := dns.RR_Header{
			Name:   r.Spec.SRV.Name,
			Rrtype: dns.TypeSRV,
			Class:  r.Spec.SRV.Class,
			Ttl:    r.Spec.SRV.Ttl,
		}
		if r.Spec.SRV.Target == "" {
			return nil, errors.New("'target' is required for SRV Records")
		}
		return &dns.SRV{
			Hdr:      h,
			Priority: r.Spec.SRV.Priority,
			Weight:   r.Spec.SRV.Weight,
			Port:     r.Spec.SRV.Port,
			Target:   r.Spec.SRV.Target,
		}, nil
	case r.Spec.MX != nil:
		h := dns.RR_Header{
			Name:   r.Spec.MX.Name,
			Rrtype: dns.TypeMX,
			Class:  r.Spec.MX.Class,
			Ttl:    r.Spec.MX.Ttl,
		}
		return &dns.MX{Hdr: h, Preference: r.Spec.MX.Preference, Mx: r.Spec.MX.Target}, nil
	case r.Spec.CNAME != nil:
		h := dns.RR_Header{
			Name:   r.Spec.CNAME.Name,
			Rrtype: dns.TypeCNAME,
			Class:  r.Spec.CNAME.Class,
			Ttl:    r.Spec.CNAME.Ttl,
		}
		if r.Spec.CNAME.Target == "" {
			return nil, errors.New("'target' is required for CNAME Records")
		}
		return &dns.CNAME{Hdr: h, Target: r.Spec.CNAME.Target}, nil
	default:
		if r.Spec.Raw == "" {
			return nil, errors.New("unknown record type")
		}
		rr, err := dns.NewRR(r.Spec.Raw)
		if err != nil {
			return nil, err
		}
		if rr == nil {
			return nil, fmt.Errorf("invalid record: '%s'", r.Spec.Raw)
		}
		return rr, nil
	}
}
