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

package crds

import (
	"context"
	"net"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/file"
	"github.com/coredns/coredns/request"
	"github.com/miekg/dns"
	"github.com/sirupsen/logrus"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	name = "k8s_dns"
)

var log = logf.Log.WithName("coredns-k8s-dns")

func init() {
	plugin.Register(name, setup)
}

type CRDS struct {
	Next     plugin.Handler
	provider Provider
	external string
}

func New(external string) (*CRDS, error) {
	provider, err := NewProvider(context.Background(), net.ParseIP(external))
	if err != nil {
		return nil, err
	}
	go func() {
		if err := provider.Run(); err != nil {
			log.Error(err, "run provider failed")
		}
	}()
	return &CRDS{provider: provider}, nil
}

// ServeDNS implements the plugin.Handle interface.
func (p *CRDS) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	state := request.Request{W: w, Req: r}

	qname := state.Name()

	zone := plugin.Zones(p.provider.Zones().Names).Matches(qname)
	if zone == "" {
		return plugin.NextOrFailure(p.Name(), p.Next, ctx, w, r)
	}

	z, ok := p.provider.Zones().Z[zone]
	if !ok || z == nil {
		return dns.RcodeServerFailure, nil
	}

	z.RLock()
	exp := z.Expired
	z.RUnlock()
	if exp {
		logrus.Errorf("Zone %s is expired", zone)
		return dns.RcodeServerFailure, nil
	}

	answer, ns, extra, result := z.Lookup(ctx, state, qname)

	m := new(dns.Msg)
	m.SetReply(r)
	m.Authoritative = true
	m.Answer, m.Ns, m.Extra = answer, ns, extra

	switch result {
	case file.Success:
	case file.NoData:
	case file.NameError:
		m.Rcode = dns.RcodeNameError
	case file.Delegation:
		m.Authoritative = false
	case file.ServerFailure:
		return dns.RcodeServerFailure, nil
	}

	// Figure out the max response size
	if _, tcp := w.RemoteAddr().(*net.TCPAddr); !tcp {
		m.Truncate(dns.MinMsgSize)
	}

	if err := w.WriteMsg(m); err != nil {
		logrus.Errorf("Failed to write response: %v", err)
		return dns.RcodeServerFailure, nil
	}
	return dns.RcodeSuccess, nil
}

func (p *CRDS) Name() string {
	return name
}

func setup(c *caddy.Controller) error {
	var external string
	for c.NextBlock() {
		switch c.Val() {
		case "external":
			args := c.RemainingArgs()
			if len(args) == 0 {
				return c.ArgErr()
			}
			external = args[0]
		default:
			return c.Errf("unknown property '%s'", c.Val())
		}
	}
	p, err := New(external)
	if err != nil {
		return plugin.Error(name, err)
	}
	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		p.Next = next
		return p
	})
	return nil
}
