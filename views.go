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
	Next           plugin.Handler
	Fall           fall.F
	Upstream       *upstream.Upstream
	ReloadInterval time.Duration

	Client       string
	ClientSchema string
	Record       string
	RecordSchema string

	ClientACLs  []*ClientACL
	ClientZones map[string]Zones
}

// ServeDNS implements the plugin.Handler interface.
func (v Views) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	state := request.Request{W: w, Req: r}

	var (
		wg      sync.WaitGroup
		answers []dns.RR
	)

	answersCh := make(chan []dns.RR)
	errCh := make(chan error)
	doneCh := make(chan bool, 1)

	go func() {
		defer close(answersCh)
		defer close(errCh)

		wg.Wait()
		doneCh <- true
	}()

	for _, client := range v.ClientACLs {
		wg.Add(1)
		go client.lookup(ctx, state, &v, &wg, answersCh, errCh)
	}

	for {
		select {
		case answers = <-answersCh:
			// when we got an answers, response with the message
			if len(answers) > 0 {
				goto Message
			}
			// if answers is empty, then go to the next plugin
			return plugin.NextOrFailure(v.Name(), v.Next, ctx, w, r)
		case err := <-errCh:
			// when we caught an error,
			// then go to the next plugin
			log.Error(err)
			return plugin.NextOrFailure(v.Name(), v.Next, ctx, w, r)
		case <-doneCh:
			// when the process is done and not giving any result,
			// then go to the next plugin
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

func (c *ClientACL) lookup(ctx context.Context, state request.Request, v *Views, wg *sync.WaitGroup, answersCh chan []dns.RR, errCh chan error) {
	defer wg.Done()

	qname := state.QName()
	qtype := state.QType()
	userIP := net.ParseIP(state.IP())

	for _, cidrNet := range c.CIDRNets {
		wg.Add(1)
		go func(cidrNet *net.IPNet) {
			defer wg.Done()

			if cidrNet.Contains(userIP) {
				z, ok := v.ClientZones[c.Name].Z[qname]
				if !ok {
					errCh <- fmt.Errorf("no zone was found. Zone: %s", qname)
				}

				log.Infof("(%s) found match for user IP (%s) with registered client CIDR prefixes: %s (%s)", c.Name, userIP.String(), cidrNet.String(), qname)

				var answers []dns.RR
				rr := new(dns.CNAME)
				rr.Hdr = dns.RR_Header{Name: qname, Rrtype: z.Type, Class: state.QClass(), Ttl: z.TTL}
				rr.Target = z.Value

				answers = append(answers, rr)

				switch qtype {
				case dns.TypeCNAME:
				default:
					rrs := v.doLookup(ctx, state, z.Value, qtype)
					answers = append(answers, rrs...)
				}

				answersCh <- answers
			}
		}(cidrNet)
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
