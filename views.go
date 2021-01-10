package views

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

// Views represent of plugin that route dns resolving based on user IP
type Views struct {
	Next plugin.Handler
	Fall fall.F

	ReloadInterval time.Duration
	ClientFilename string
	RecordFilename string

	ClientACLs  []*ClientACL
	ClientZones map[string]Zones
	Upstream    *upstream.Upstream
}

type (
	// ClientACL represent Client definition and their CIDR Prefix list
	ClientACL struct {
		Name     string
		CIDRNets []*net.IPNet
	}

	// Zones represent list of zones available
	Zones struct {
		Z     map[string]Zone
		Names []string
	}

	// Zone represent of single zone record definition
	Zone struct {
		Name  string
		TTL   uint32
		Type  uint16
		Value string
	}
)

// ServeDNS implements the plugin.Handler interface.
func (v Views) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	state := request.Request{W: w, Req: r}
	answers, err := v.lookup(ctx, state)
	if err != nil {
		return plugin.NextOrFailure(v.Name(), v.Next, ctx, w, r)
	}

	m := new(dns.Msg)
	m.SetReply(r)
	m.Authoritative = true
	m.Answer = answers

	err = w.WriteMsg(m)
	if err != nil {
		log.Error(err)
		return plugin.NextOrFailure(v.Name(), v.Next, ctx, w, r)
	}

	return dns.RcodeSuccess, nil
}

// Name implements the Handler interface.
func (v Views) Name() string { return "views" }

func (v *Views) lookup(ctx context.Context, state request.Request) (answers []dns.RR, err error) {
	qname := state.QName()
	userIP := net.ParseIP(state.IP())

Loop:
	for _, client := range v.ClientACLs {
		for _, cidrNet := range client.CIDRNets {
			if cidrNet.Contains(userIP) {
				log.Infof("found match for user IP (%s) with registered client (%s) \"%s\"", userIP.String(), client.Name, qname)

				z, ok := v.ClientZones[client.Name].Z[qname]
				if !ok {
					err = fmt.Errorf("no client zone was found. Client: %s, Zone: %s", client.Name, qname)
				}

				if z.Value == "" {
					err = fmt.Errorf("no record value was found. Zone: %s", qname)
				}

				rr := new(dns.CNAME)
				rr.Hdr = dns.RR_Header{Name: qname, Rrtype: z.Type, Class: state.QClass(), Ttl: z.TTL}
				rr.Target = z.Value

				answers = append(answers, rr)

				if state.QType() != dns.TypeCNAME {
					rrs := v.doLookup(ctx, state, z.Value, state.QType())
					answers = append(answers, rrs...)
				}
				break Loop
			}
		}
	}

	return
}

func (v *Views) doLookup(ctx context.Context, state request.Request, target string, qtype uint16) []dns.RR {
	m, e := v.Upstream.Lookup(ctx, state, target, qtype)
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
