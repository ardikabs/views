package views

import (
	"context"
	"fmt"
	"net"
	"sync"
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

	var (
		wg      sync.WaitGroup
		answers []dns.RR
	)

	resultChan := make(chan []dns.RR)
	errChan := make(chan error)
	doneChan := make(chan bool, 1)

	go func() {
		wg.Wait()
		close(resultChan)
		close(errChan)
		doneChan <- true
	}()

	for _, client := range v.ClientACLs {
		wg.Add(1)
		go client.lookup(ctx, state, &v, &wg, resultChan, errChan)
	}

	for {
		select {
		case answers = <-resultChan:
			if len(answers) > 0 {
				goto Message
			}
			return plugin.NextOrFailure(v.Name(), v.Next, ctx, w, r)
		case err := <-errChan:
			log.Error(err)
			return plugin.NextOrFailure(v.Name(), v.Next, ctx, w, r)
		case <-doneChan:
			return plugin.NextOrFailure(v.Name(), v.Next, ctx, w, r)
		}

	}

Message:
	m := new(dns.Msg)
	m.SetReply(r)
	m.Authoritative = true
	m.Answer = answers

	err := w.WriteMsg(m)
	if err != nil {
		log.Error(err)
		return plugin.NextOrFailure(v.Name(), v.Next, ctx, w, r)
	}

	return dns.RcodeSuccess, nil
}

func (c *ClientACL) lookup(ctx context.Context, state request.Request, v *Views, wg *sync.WaitGroup, resultCh chan []dns.RR, errCh chan error) {
	defer wg.Done()

	qname := state.QName()
	qtype := state.QType()
	userIP := net.ParseIP(state.IP())

	for _, cidrNet := range c.CIDRNets {
		zone, ok := v.ClientZones[c.Name].Z[state.QName()]
		if !ok {
			errCh <- fmt.Errorf("no client zone was found. Client: %s, Zone: %s", c.Name, state.QName())
		}

		wg.Add(1)
		go func(z Zone, cidrNet *net.IPNet) {
			defer wg.Done()

			if cidrNet.Contains(userIP) {
				log.Infof("found match for user IP (%s) with registered client (%s) \"%s\"", userIP.String(), c.Name, qname)
				var answers []dns.RR
				rr := new(dns.CNAME)
				rr.Hdr = dns.RR_Header{Name: qname, Rrtype: z.Type, Class: state.QClass(), Ttl: z.TTL}
				rr.Target = z.Value

				answers = append(answers, rr)

				if qtype != dns.TypeCNAME {
					rrs := v.doLookup(ctx, state, z.Value, qtype)
					answers = append(answers, rrs...)
				}
				resultCh <- answers
			}
		}(zone, cidrNet)
	}
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

// Name implements the Handler interface.
func (v Views) Name() string { return "views" }
