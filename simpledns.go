package simpledns

import (
	"context"
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
	ClientACLs     []ClientACL
	Records        map[string][]Record
	Upstream       *upstream.Upstream
}

type (
	ClientACL struct {
		Name         string   `yaml:"name"`
		CIDRPrefixes []string `yaml:"prefix_list"`
	}

	Record struct {
		Name  string
		TTL   uint32
		Type  uint16
		Value string
	}
)

// ServeDNS implements the plugin.Handler interface.
func (s SimpleDNS) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	state := request.Request{W: w, Req: r}
	qname := state.Name()
	userIP := state.IP()

	var answers []dns.RR

	for _, client := range s.ClientACLs {
		for _, cidr := range client.CIDRPrefixes {
			_, cidrNet, _ := net.ParseCIDR(cidr)
			if cidrNet.Contains(net.ParseIP(userIP)) {
				log.Infof("User IP: %s\n", userIP)
				log.Infof("Question: %s\n", qname)

				zones, ok := s.Records[client.Name]
				if !ok {
					return plugin.NextOrFailure(s.Name(), s.Next, ctx, w, r)
				}

				for _, zone := range zones {
					if zone.Name == qname {
						rr := new(dns.CNAME)
						rr.Hdr = dns.RR_Header{Name: qname, Rrtype: dns.TypeCNAME, Class: state.QClass(), Ttl: 300}
						rr.Target = zone.Value
						answers = append(answers, rr)

						if state.QType() != dns.TypeCNAME {
							rrs := lookup(ctx, state, zone.Value, state.QType())
							answers = append(answers, rrs...)
						}
					}
				}
			}
		}
	}

	if len(answers) > 0 {
		m := new(dns.Msg)
		m.SetReply(r)
		m.Authoritative = true
		m.Answer = answers

		w.WriteMsg(m)
		return dns.RcodeSuccess, nil
	}
	return plugin.NextOrFailure(s.Name(), s.Next, ctx, w, r)
}

// Name implements the Handler interface.
func (s SimpleDNS) Name() string { return "simpledns" }

func lookup(ctx context.Context, state request.Request, target string, qtype uint16) []dns.RR {
	u := upstream.New()

	m, e := u.Lookup(ctx, state, target, qtype)
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
