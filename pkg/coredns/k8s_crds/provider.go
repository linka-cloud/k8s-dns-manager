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
	"go.linka.cloud/k8s/dns/pkg/record"
)

const (
	defaultTTL = 3600
	defaultHostmaster = "hostmaster"
	defaultApex = "dns"
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

func NewProvider(ctx context.Context) (Provider, error) {
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
		records:    make(map[string]dns.RR),
		hostmaster: defaultHostmaster,
		ttl:        defaultTTL,
		apex:       defaultApex,
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
		zone := strings.Join(parts[len(parts)-2:], ".") + "."
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
		if len(v.NS) == 0 {
			header := dns.RR_Header{Name: k, Rrtype: dns.TypeNS, Ttl: p.ttl, Class: dns.ClassINET}
			v.NS = append(v.NS, &dns.NS{Hdr: header, Ns: ns})
			// TODO(adphi): insert A Record for NS
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
			rr, err := makeRecord(obj)
			if err != nil {
				log.Error(err, "add func handler failed")
				return
			}
			log.Info("adding record", "record", rr.String())
			p.mu.Lock()
			p.records[fmt.Sprintf("%s:%d", rr.Header().Name, rr.Header().Rrtype)] = rr
			p.mu.Unlock()
			if err := p.sync(); err != nil {
				log.Error(err, "zones sync had errors")
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldRR, err := makeRecord(newObj)
			if err != nil {
				log.Error(err, "update func handler failed")
				return
			}
			newRR, err := makeRecord(newObj)
			if err != nil {
				log.Error(err, "update func handler failed")
				return
			}
			log.Info("replacing record", "old", oldRR.String(), "new", newRR.String())
			p.mu.Lock()
			delete(p.records, fmt.Sprintf("%s:%d", oldRR.Header().Name, oldRR.Header().Rrtype))
			p.records[fmt.Sprintf("%s:%d", newRR.Header().Name, newRR.Header().Rrtype)] = newRR
			p.mu.Unlock()
			if err := p.sync(); err != nil {
				log.Error(err, "zones sync had errors")
			}
		},
		DeleteFunc: func(obj interface{}) {
			rr, err := makeRecord(obj)
			if err != nil {
				log.Error(err, "delete func handler failed")
				return
			}
			log.Info("deleting record", "record", rr.String())
			p.mu.Lock()
			delete(p.records, fmt.Sprintf("%s:%d", rr.Header().Name, rr.Header().Rrtype))
			p.mu.Unlock()
			if err := p.sync(); err != nil {
				log.Error(err, "zones sync had errors")
			}
		},
	})
	return p.cache.Start(p.ctx)
}

func makeRecord(obj interface{}) (dns.RR, error) {
	r, ok := obj.(*v1alpha1.DNSRecord)
	if !ok || r == nil {
		return nil, errors.New("obj is nil or is not a DNSRecord")
	}
	rr, err := record.ToRR(*r)
	if err != nil {
		return nil, fmt.Errorf("record conversion: %w", err)
	}
	return rr, nil
}

func init() {
	_ = clientgoscheme.AddToScheme(scheme.Scheme)

	_ = v1alpha1.AddToScheme(scheme.Scheme)
	// +kubebuilder:scaffold:scheme
}
