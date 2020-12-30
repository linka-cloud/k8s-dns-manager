package crds

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/file"
	"github.com/coredns/coredns/plugin/pkg/dnsutil"
	"github.com/miekg/dns"
	"go.uber.org/multierr"
	"k8s.io/client-go/kubernetes/scheme"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	toolscache "k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	"go.linka.cloud/k8s/dns/api/v1alpha1"
	"go.linka.cloud/k8s/dns/pkg/ptr"
	"go.linka.cloud/k8s/dns/pkg/record"
)

const (
	defaultTTL        = 3600
	defaultHostmaster = "hostmaster"
	defaultApex       = "dns"
)

type Provider interface {
	Zones() file.Zones
	Run() error
}

type provider struct {
	ctx             context.Context
	cache           cache.Cache
	zones           file.Zones
	records         map[string]dns.RR
	externalAddress net.IP
	mu              sync.RWMutex

	hostmaster string
	ttl        uint32
	apex       string
}

func NewProvider(ctx context.Context, externalAddress net.IP) (Provider, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	conf, err := config.GetConfig()
	if err != nil {
		return nil, err
	}
	c, err := cache.New(conf, cache.Options{})
	if err != nil {
		return nil, err
	}
	p := &provider{
		ctx:   ctx,
		cache: c,
		zones: file.Zones{
			Z: map[string]*file.Zone{},
		},
		records:         make(map[string]dns.RR),
		hostmaster:      defaultHostmaster,
		ttl:             defaultTTL,
		apex:            defaultApex,
		externalAddress: externalAddress,
	}
	return p, nil
}

func (p *provider) Zones() file.Zones {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.zones
}

func (p *provider) sync() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.zones = file.Zones{
		Z: map[string]*file.Zone{},
	}
	if p.hostmaster == "" {
		p.hostmaster = defaultHostmaster
	}
	if p.ttl == 0 {
		p.ttl = defaultTTL
	}
	if p.apex == "" {
		p.apex = defaultApex
	}
	var merr error

	for _, v := range p.records {
		parts := dns.SplitDomainName(v.Header().Name)
		if len(parts) < 2 {
			merr = multierr.Append(merr, fmt.Errorf("malformed name: %s", v.Header().Name))
			continue
		}
		zone := dns.Fqdn(strings.Join(parts[len(parts)-2:], "."))
		zone = plugin.Name(zone).Normalize()
		z, ok := p.zones.Z[zone]
		if !ok {
			z = file.NewZone(zone, "")
			p.zones.Z[zone] = z
			p.zones.Names = append(p.zones.Names, zone)
		}
		if err := z.Insert(v); err != nil {
			merr = multierr.Append(merr, err)
		}
	}
	for k, v := range p.zones.Z {
		ns := dnsutil.Join("ns0", p.apex, k)
		if v.SOA == nil {
			v.SOA = &dns.SOA{
				Hdr:     dns.RR_Header{Name: k, Rrtype: dns.TypeSOA, Ttl: p.ttl, Class: dns.ClassINET},
				Mbox:    dnsutil.Join(p.hostmaster, p.apex, k),
				Ns:      ns,
				Serial:  1499347823, // Also dynamic?
				Refresh: 7200,
				Retry:   1800,
				Expire:  86400,
				Minttl:  5,
			}
		}
		if len(v.NS) != 0 {
			continue
		}
		header := dns.RR_Header{Name: k, Rrtype: dns.TypeNS, Ttl: p.ttl, Class: dns.ClassINET}
		v.NS = append(v.NS, &dns.NS{Hdr: header, Ns: ns})
		if p.externalAddress == nil {
			continue
		}
		nsRecord := &dns.A{
			Hdr: dns.RR_Header{
				Name:   ns,
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
				Ttl:    p.ttl,
			},
			A: p.externalAddress,
		}
		if err := v.Insert(nsRecord); err != nil {
			return err
		}

	}
	return merr
}

func (p *provider) Run() error {
	i, err := p.cache.GetInformer(p.ctx, &v1alpha1.DNSRecord{})
	if err != nil {
		return err
	}
	i.AddEventHandler(toolscache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			rr, r, err := makeRecord(obj)
			if err != nil {
				log.Error(err, "add func handler failed")
				return
			}
			if ptr.ToBoolD(r.Spec.Active, true) {
				log.Info("skip adding inactive record", "record", rr.String())
				return
			}
			log.Info("adding record", "record", rr.String())
			p.mu.Lock()
			p.records[rr.String()] = rr
			p.mu.Unlock()
			if err := p.sync(); err != nil {
				log.Error(err, "zones sync had errors")
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldRR, r, err := makeRecord(oldObj)
			if err != nil {
				log.Error(err, "update func handler failed")
				return
			}
			newRR, r, err := makeRecord(newObj)
			if err != nil {
				log.Error(err, "update func handler failed")
				return
			}
			log.Info("deleting record", "old", oldRR.String())
			p.mu.Lock()
			delete(p.records, oldRR.String())
			if ptr.ToBoolD(r.Spec.Active, true) {
				log.Info("adding record", "new", newRR.String())
				p.records[newRR.String()] = newRR
			} else {
				log.Info("skip adding inactive record", "record", newRR.String())
			}
			p.mu.Unlock()
			if err := p.sync(); err != nil {
				log.Error(err, "zones sync had errors")
			}
		},
		DeleteFunc: func(obj interface{}) {
			rr, _, err := makeRecord(obj)
			if err != nil {
				log.Error(err, "delete func handler failed")
				return
			}
			log.Info("deleting record", "record", rr.String())
			p.mu.Lock()
			delete(p.records, rr.String())
			p.mu.Unlock()
			if err := p.sync(); err != nil {
				log.Error(err, "zones sync had errors")
			}
		},
	})
	return p.cache.Start(p.ctx)
}

func makeRecord(obj interface{}) (dns.RR, *v1alpha1.DNSRecord, error) {
	r, ok := obj.(*v1alpha1.DNSRecord)
	if !ok || r == nil {
		return nil, nil, errors.New("obj is nil or is not a DNSRecord")
	}
	rr, err := record.ToRR(*r)
	if err != nil {
		return nil, nil, fmt.Errorf("record conversion: %w", err)
	}
	return rr, r, nil
}

func init() {
	_ = clientgoscheme.AddToScheme(scheme.Scheme)

	_ = v1alpha1.AddToScheme(scheme.Scheme)
	// +kubebuilder:scaffold:scheme
}
