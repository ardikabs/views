package simpledns

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"time"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	clog "github.com/coredns/coredns/plugin/pkg/log"
	"github.com/miekg/dns"
	"gopkg.in/yaml.v2"
)

var (
	log = clog.NewWithPlugin("simpledns")
)

type (
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

func init() { plugin.Register("simpledns", setup) }

func setup(c *caddy.Controller) error {
	simpleDNS, err := parse(c)
	if err != nil {
		return plugin.Error("simpledns", err)
	}

	reloadChan := simpleDNS.reload()

	c.OnStartup(func() error {
		simpleDNS.loadConfig()
		return nil
	})

	c.OnShutdown(func() error {
		close(reloadChan)
		return nil
	})

	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		simpleDNS.Next = next
		return &simpleDNS
	})

	return nil
}

func parse(c *caddy.Controller) (SimpleDNS, error) {
	var (
		clientFilename string
		recordFilename string
	)

	simpleDNS := SimpleDNS{
		Reload: 5,
	}

	for c.Next() {
		// To check whether there is argument exist or not
		// if !c.NextArg() {
		// 	return c.ArgErr()
		// }
		for c.NextBlock() {
			switch c.Val() {
			case "clients":
				clientFilename = c.RemainingArgs()[0]
			case "records":
				recordFilename = c.RemainingArgs()[0]
			case "reload":
				duration, err := strconv.Atoi(c.RemainingArgs()[0])
				if err != nil {
					return SimpleDNS{}, fmt.Errorf("wrong format for reload duration: %v", err)
				}
				simpleDNS.Reload = time.Duration(duration)
			default:
				return SimpleDNS{}, fmt.Errorf("unknown argument: %s", c.Val())
			}
		}
	}

	if clientFilename == "" && recordFilename == "" {
		return SimpleDNS{}, fmt.Errorf("required argument is missing: (client: '%s') and (records: '%s')", clientFilename, recordFilename)
	}

	simpleDNS.ClientFilename = clientFilename
	simpleDNS.RecordFilename = recordFilename

	return simpleDNS, nil
}

func (s *SimpleDNS) reload() chan bool {
	reloadChan := make(chan bool)

	go func() {
		ticker := time.NewTicker(time.Second * s.Reload)
		for {
			select {
			case <-reloadChan:
				return
			case <-ticker.C:
				s.loadConfig()
			}
		}
	}()

	return reloadChan
}

func (s *SimpleDNS) loadConfig() {
	var rawClients []ClientACL
	var rawRecords []RecordFile

	file, err := ioutil.ReadFile(s.ClientFilename)
	err = yaml.Unmarshal(file, &rawClients)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	file, err = ioutil.ReadFile(s.RecordFilename)
	err = yaml.Unmarshal(file, &rawRecords)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	s.ClientACLs = rawClients
	s.Records = make(map[string][]Record)

	for _, r := range rawRecords {
		records := []Record{}
		for _, rawRecord := range r.Records {
			xtype := strings.ToUpper(rawRecord.Type)
			var t uint16

			switch xtype {
			case "A":
				t = dns.TypeA
			case "AAAA":
				t = dns.TypeAAAA
			case "CNAME":
				t = dns.TypeCNAME
			}

			record := Record{
				Name:  plugin.Host(rawRecord.Name).Normalize(),
				TTL:   rawRecord.TTL,
				Type:  t,
				Value: plugin.Host(rawRecord.Value).Normalize(),
			}

			records = append(records, record)
		}

		s.Records[r.Name] = records
	}

}
