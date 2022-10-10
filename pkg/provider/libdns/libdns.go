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

package libdns

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/libdns/libdns"
	"github.com/miekg/dns"
	"github.com/weppos/publicsuffix-go/publicsuffix"
	ctrl "sigs.k8s.io/controller-runtime"

	"go.linka.cloud/k8s/dns/api/v1alpha1"
	"go.linka.cloud/k8s/dns/pkg/provider"
	"go.linka.cloud/k8s/dns/pkg/record"
)

type Client interface {
	libdns.RecordGetter
	libdns.RecordAppender
	libdns.RecordDeleter
}

type prov struct {
	name string
	c    Client
}

func New(name string, c Client) provider.Provider {
	return &prov{name: name, c: c}
}

func NewSync(name string, c Client) provider.Provider {
	return &prov{name: name, c: &syncClient{c: c}}
}

func (p prov) Reconcile(ctx context.Context, rec *v1alpha1.DNSRecord) (ctrl.Result, bool, error) {
	log := ctrl.LoggerFrom(ctx).WithValues("provider", p.name)
	// we don't own this record, so we should not reconcile it
	if rec.Status.Provider != "" && rec.Status.Provider != p.name {
		log.Info("skipping record, not for this provider", "recordProvider", rec.Status.Provider)
		return ctrl.Result{}, false, nil
	}
	rr, err := record.ToRR(*rec)
	if err != nil {
		return ctrl.Result{}, false, err
	}
	name := rr.Header().Name
	parts := dns.SplitDomainName(name)
	if len(parts) < 2 {
		return ctrl.Result{}, false, fmt.Errorf("malformed name: %s", rr.Header().Name)
	}
	d, err := publicsuffix.Domain(strings.Join(parts, "."))
	if err != nil {
		log.Error(err, "parse domain", "domain", name)
		return ctrl.Result{}, false, err
	}
	zone := dns.Fqdn(d)
	recs, err := p.c.GetRecords(ctx, zone)
	if err != nil {
		log.Error(err, "get records", "zone", zone)
		return ctrl.Result{}, false, err
	}
	want := makeRecord(rr, zone, rec.Status.ID)
	var r *libdns.Record
	for _, v := range recs {
		if v.ID == rec.Status.ID {
			r = &v
			break
		}
	}

	if !rec.DeletionTimestamp.IsZero() || (rec.Spec.Active != nil && !*rec.Spec.Active) {
		if r == nil {
			rec.Status.ID = ""
			rec.Status.Provider = ""
			return ctrl.Result{}, true, nil
		}
		log.Info("delete record", "record", r.Name, "type", r.Type, "value", r.Value)
		if _, err := p.c.DeleteRecords(ctx, zone, []libdns.Record{*r}); err != nil {
			log.Error(err, "delete records", "zone", zone, "record", rec.Name)
			return ctrl.Result{}, false, err
		}
		rec.Status.ID = ""
		rec.Status.Provider = ""
		return ctrl.Result{}, true, nil
	}

	if r == nil {
		log.Info("create record")
		// AppendRecords implementation should prevent from creating duplicate records or overriding existing ones
		rs, err := p.c.AppendRecords(ctx, zone, []libdns.Record{*want})
		if err != nil {
			log.Error(err, "create record", "zone", zone, "record", want.Name)
			return ctrl.Result{}, false, err
		}
		if len(rs) != 1 {
			return ctrl.Result{}, false, fmt.Errorf("expected 1 record, got %d", len(rs))
		}
		rec.Status.Provider = p.name
		rec.Status.ID = rs[0].ID
		return ctrl.Result{}, true, nil
	}

	FqdnRec(r, zone)
	if r.Name == want.Name &&
		r.Value == want.Value &&
		r.Type == want.Type &&
		r.TTL == want.TTL {
		log.Info("record up to date")
		return ctrl.Result{}, true, nil
	}

	log.Info("update record: delete")
	if _, err := p.c.DeleteRecords(ctx, zone, []libdns.Record{*r}); err != nil {
		log.Error(err, "delete record", "zone", zone, "record", r.Name)
		return ctrl.Result{}, false, err
	}
	log.Info("update record: create")
	rs, err := p.c.AppendRecords(ctx, zone, []libdns.Record{*want})
	if err != nil {
		log.Error(err, "append record", "zone", zone, "record", want.Name)
		return ctrl.Result{}, false, err
	}
	if len(rs) != 1 {
		return ctrl.Result{}, false, fmt.Errorf("expected 1 record, got %d", len(rs))
	}

	rec.Status.Provider = p.name
	rec.Status.ID = rs[0].ID
	return ctrl.Result{}, true, nil
}

func makeRecord(rr dns.RR, zone string, id string) *libdns.Record {
	rec := &libdns.Record{
		ID:   id,
		Name: rr.Header().Name,
		Type: dns.TypeToString[rr.Header().Rrtype],
		TTL:  time.Duration(rr.Header().Ttl) * time.Second,
	}
	switch r := rr.(type) {
	case *dns.A:
		rec.Value = r.A.String()
	case *dns.AAAA:
		rec.Value = r.AAAA.String()
	case *dns.CNAME:
		rec.Value = r.Target
	case *dns.MX:
		rec.Value = fmt.Sprintf("%d %s", r.Preference, r.Mx)
	case *dns.NS:
		rec.Value = r.Ns
	case *dns.SRV:
		rec.Value = fmt.Sprintf("%d %d %d %s", r.Priority, r.Weight, r.Port, r.Target)
	case *dns.TXT:
		rec.Value = strings.Join(r.Txt, "")
	}
	return rec
}

func FqdnRec(rec *libdns.Record, zone string) {
	rec.Name = Fqdn(rec.Name, zone)
	switch rec.Type {
	case "A", "AAAA", "TXT":
		rec.Value = Unquote(strings.Replace(rec.Value, "\" \"", "", -1))
	case "CNAME", "MX", "NS", "SRV":
		rec.Value = Unquote(Fqdn(rec.Value, zone))
	}
}

func Fqdn(name string, zone string) string {
	if strings.HasSuffix(name, UnFqdn(zone)) {
		return dns.Fqdn(name)
	}
	if !strings.HasSuffix(name, dns.Fqdn(zone)) {
		return libdns.AbsoluteName(name, zone)
	}
	return name
}

func UnFqdn(name string) string {
	return strings.TrimSuffix(name, ".")
}

func Unquote(name string) string {
	return strings.TrimRight(strings.TrimLeft(strings.TrimSpace(name), "\""), "\"")
}
