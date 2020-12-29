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

func ToRR(r v1alpha1.DNSRecord) (dns.RR, error) {
	h := dns.RR_Header{
		Name:  r.Spec.Name,
		Class: r.Spec.Class,
		Ttl:   r.Spec.Ttl,
	}
	switch {
	case r.Spec.ARecord != nil:
		h.Rrtype = dns.TypeA
		ip := net.ParseIP(r.Spec.ARecord.A)
		if ip == nil {
			return nil, fmt.Errorf("invalid ip: %s", r.Spec.ARecord.A)
		}
		return &dns.A{Hdr: h, A: ip}, nil
	case r.Spec.TXTRecord != nil:
		h.Rrtype = dns.TypeTXT
		if len(r.Spec.Txt) == 0 {
			return nil, errors.New("empty TXT record")
		}
		return &dns.TXT{Hdr: h, Txt: r.Spec.TXTRecord.Txt}, nil
	case r.Spec.SRVRecord != nil:
		h.Rrtype = dns.TypeSRV
		if r.Spec.Target == "" {
			return nil, errors.New("'target' is required for SRV Records")
		}
		return &dns.SRV{
			Hdr:      h,
			Priority: r.Spec.SRVRecord.Priority,
			Weight:   r.Spec.SRVRecord.Weight,
			Port:     r.Spec.SRVRecord.Port,
			Target:   r.Spec.Target,
		}, nil
	case r.Spec.MXRecord != nil:
		h.Rrtype = dns.TypeMX
		return &dns.MX{Hdr: h, Preference: r.Spec.MXRecord.Preference, Mx: r.Spec.MXRecord.Mx}, nil
	case r.Spec.Target != "":
		h.Rrtype = dns.TypeCNAME
		return &dns.CNAME{Hdr: h, Target: r.Spec.Target}, nil
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
