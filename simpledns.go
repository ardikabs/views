package simpledns

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/pkg/fall"
	"github.com/coredns/coredns/plugin/pkg/upstream"
	"github.com/coredns/coredns/request"

	"github.com/miekg/dns"
)

type SimpleDNS struct {
	Next plugin.Handler
	Fall fall.F

	Reload         time.Duration
	ClientFilename string
	RecordFilename string

	ClientACLs  []*ClientACL
	ClientZones map[string]Zones
	Upstream    *upstream.Upstream
}

type (
	ClientACL struct {
		Name     string
		CIDRNets []*net.IPNet
	}

	Zones struct {
		Z     map[string]Zone
		Names []string
	}

	Zone struct {
		Name  string
		TTL   uint32
		Type  uint16
		Value string
	}
)

// ServeDNS implements the plugin.Handler interface.
func (s SimpleDNS) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	state := request.Request{W: w, Req: r}

	answers, err := s.lookup(ctx, state, state.QName())
	if err != nil {
		return plugin.NextOrFailure(s.Name(), s.Next, ctx, w, r)
	}

	m := new(dns.Msg)
	m.SetReply(r)
	m.Authoritative = true
	m.Answer = answers

	err = w.WriteMsg(m)
	if err != nil {
		log.Error(err)
		return plugin.NextOrFailure(s.Name(), s.Next, ctx, w, r)
	}

	return dns.RcodeSuccess, nil
}

// Name implements the Handler interface.
func (s SimpleDNS) Name() string { return "simpledns" }

func (s *SimpleDNS) lookup(ctx context.Context, state request.Request, qname string) ([]dns.RR, error) {
	var answers []dns.RR

Loop:
	for _, client := range s.ClientACLs {
		for _, cidrNet := range client.CIDRNets {
			if cidrNet.Contains(net.ParseIP(state.IP())) {
				log.Infof("match user IP with registered client (%s): %s (%s)", client.Name, state.IP(), qname)

				z, ok := s.ClientZones[client.Name].Z[qname]
				if !ok {
					return nil, fmt.Errorf("no client zone was found: %s", qname)
				}

				if z.Value == "" {
					return nil, fmt.Errorf("no record value was found: %s", z.Name)
				}

				rr := new(dns.CNAME)
				rr.Hdr = dns.RR_Header{Name: qname, Rrtype: dns.TypeCNAME, Class: state.QClass(), Ttl: z.TTL}
				rr.Target = z.Value

				answers = append(answers, rr)

				if state.QType() != dns.TypeCNAME {
					rrs := s.doLookup(ctx, state, z.Value, state.QType())
					answers = append(answers, rrs...)
				}

				break Loop
			}
		}
	}

	if len(answers) == 0 {
		return nil, fmt.Errorf("Skipping cause no record was found: %s", qname)
	}

	return answers, nil
}

func (s *SimpleDNS) doLookup(ctx context.Context, state request.Request, target string, qtype uint16) []dns.RR {
	m, e := s.Upstream.Lookup(ctx, state, target, qtype)
	if e != nil {
		return nil
	}
	if m == nil {
		return nil
	}
	if m.Rcode == dns.RcodeNameError {
		return m.Answer
	}
	if m.Rcode == dns.RcodeServerFailure {
		return m.Answer
	}
	if m.Rcode == dns.RcodeSuccess && len(m.Answer) == 0 {
		return m.Answer
	}
	return m.Answer
}
