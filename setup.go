package views

import (
	"fmt"
	"io/ioutil"
	"net"
	"strings"
	"time"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	clog "github.com/coredns/coredns/plugin/pkg/log"
	"github.com/coredns/coredns/plugin/pkg/upstream"
	"github.com/miekg/dns"
	"gopkg.in/yaml.v2"
)

var (
	log = clog.NewWithPlugin("views")
)

type (
	// ClientACLFile represent specification of Client ACL YAML-file
	ClientACLFile struct {
		Name         string   `yaml:"name"`
		CIDRPrefixes []string `yaml:"prefix_list"`
	}

	// RecordFile represent specification of Record YAML-file
	RecordFile struct {
		Name    string `yaml:"name"`
		Records []struct {
			Name  string `yaml:"name"`
			TTL   uint32 `yaml:"ttl"`
			Type  string `yaml:"type"`
			Value string `yaml:"value"`
		} `yaml:"records"`
	}
)

func init() { plugin.Register("views", setup) }

func setup(c *caddy.Controller) error {
	v, err := parse(c)
	if err != nil {
		return plugin.Error("views", err)
	}

	reloadChan := v.reload()

	c.OnStartup(func() error {
		v.loadConfig()
		return nil
	})

	c.OnShutdown(func() error {
		close(reloadChan)
		return nil
	})

	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		v.Next = next
		return v
	})

	return nil
}

const (
	defaultReloadInterval = 30 * time.Second
)

func parse(c *caddy.Controller) (*Views, error) {
	var (
		clientFilename string
		recordFilename string
	)

	v := Views{
		ReloadInterval: defaultReloadInterval,
		Upstream:       upstream.New(),
	}

	for c.Next() {
		for c.NextBlock() {
			switch c.Val() {
			case "clients":
				clientFilename = c.RemainingArgs()[0]
			case "records":
				recordFilename = c.RemainingArgs()[0]
			case "reload":
				d, err := time.ParseDuration(c.RemainingArgs()[0])
				if err != nil {
					return nil, err
				}
				v.ReloadInterval = d
			default:
				return nil, fmt.Errorf("unknown argument: %s", c.Val())
			}
		}
	}

	if clientFilename == "" && recordFilename == "" {
		return nil, fmt.Errorf("required argument is missing: (client: '%s') and (records: '%s')", clientFilename, recordFilename)
	}

	v.ClientFilename = clientFilename
	v.RecordFilename = recordFilename

	return &v, nil
}

func (v *Views) reload() chan bool {
	reloadChan := make(chan bool)

	go func() {
		ticker := time.NewTicker(v.ReloadInterval)
		for {
			select {
			case <-reloadChan:
				return
			case <-ticker.C:
				v.loadConfig()
			}
		}
	}()

	return reloadChan
}

func (v *Views) loadConfig() {
	var rawClients []ClientACLFile
	var rawRecords []RecordFile

	file, err := ioutil.ReadFile(v.ClientFilename)
	err = yaml.Unmarshal(file, &rawClients)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	file, err = ioutil.ReadFile(v.RecordFilename)
	err = yaml.Unmarshal(file, &rawRecords)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	v.ClientACLs = []*ClientACL{}

	for _, client := range rawClients {
		var cidrNets []*net.IPNet
		for _, cidr := range client.CIDRPrefixes {
			_, cidrNet, err := net.ParseCIDR(cidr)
			if err != nil {
				log.Warningf("invalid CIDR address: %s (%s)", client.Name, cidr)
				continue
			}

			cidrNets = append(cidrNets, cidrNet)
		}

		v.ClientACLs = append(v.ClientACLs, &ClientACL{
			Name:     client.Name,
			CIDRNets: cidrNets,
		})
	}

	v.ClientZones = make(map[string]Zones)
	for _, r := range rawRecords {
		zones := Zones{
			Names: []string{},
			Z:     make(map[string]Zone),
		}

		for _, rawRecord := range r.Records {
			t := strings.ToUpper(rawRecord.Type)
			var rrtype uint16

			switch t {
			case "A":
				rrtype = dns.TypeA
			case "AAAA":
				rrtype = dns.TypeAAAA
			case "CNAME":
				rrtype = dns.TypeCNAME
			case "TXT":
				rrtype = dns.TypeTXT
			}

			rr := Zone{
				Name:  plugin.Host(rawRecord.Name).Normalize(),
				TTL:   rawRecord.TTL,
				Type:  rrtype,
				Value: plugin.Host(rawRecord.Value).Normalize(),
			}

			zones.Names = append(zones.Names, rr.Name)
			zones.Z[rr.Name] = rr
		}

		v.ClientZones[r.Name] = zones
	}
}
