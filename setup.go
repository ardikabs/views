package views

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	clog "github.com/coredns/coredns/plugin/pkg/log"
	"github.com/coredns/coredns/plugin/pkg/upstream"
	"gopkg.in/yaml.v2"
)

const (
	defaultReloadInterval = 30 * time.Second
)

var (
	log = clog.NewWithPlugin("views")
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

func parse(c *caddy.Controller) (*Views, error) {
	var (
		client       string
		clientSchema string
		record       string
		recordSchema string
		err          error
	)

	v := Views{
		ReloadInterval: defaultReloadInterval,
		Upstream:       upstream.New(),
	}

	for c.Next() {
		for c.NextBlock() {
			switch c.Val() {
			case "client":
				client = c.RemainingArgs()[0]
				clientSchema, err = schemaCheck(client)
				if err != nil {
					return nil, err
				}
			case "record":
				record = c.RemainingArgs()[0]
				recordSchema, err = schemaCheck(record)
				if err != nil {
					return nil, err
				}
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

	if err != nil {
		return nil, err
	}

	if client == "" {
		return nil, fmt.Errorf("required argument is missing: 'client'")
	} else if record == "" {
		return nil, fmt.Errorf("required argument is missing: 'record'")
	}

	v.Client = client
	v.Record = record
	v.ClientSchema = clientSchema
	v.RecordSchema = recordSchema

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
	var (
		rawClients []RawClientACL
		rawRecords []RawRecord
		err        error
	)

	switch v.ClientSchema {
	case SchemaYAML:
		err = parseFromYAML(v.Client, &rawClients)
	case SchemaHTTP:
		err = parseFromHTTP(v.Client, &rawClients)
	}

	if err != nil {
		log.Error(err)
	}

	switch v.RecordSchema {
	case SchemaYAML:
		err = parseFromYAML(v.Record, &rawRecords)
	case SchemaHTTP:
		err = parseFromHTTP(v.Record, &rawRecords)
	}
	if err != nil {
		log.Error(err)
	}

	v.ClientACLs = []*ClientACL{}

	for _, client := range rawClients {
		var cidrNets []*net.IPNet
		for _, cidr := range client.CIDRPrefixes {
			_, cidrNet, err := net.ParseCIDR(cidr)
			if err != nil {
				log.Warningf("(%s) invalid CIDR address: %s", client.Name, cidr)
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
	for _, raw := range rawRecords {
		zones := Zones{
			Names: []string{},
			Z:     make(map[string]Zone),
		}

		for _, record := range raw.Records {
			rr, err := NewZoneRecord(record)
			if err != nil {
				log.Warningf("(%s) %s", raw.Name, err)
				continue
			}

			zones.Names = append(zones.Names, rr.Name)
			zones.Z[rr.Name] = rr
		}

		v.ClientZones[raw.Name] = zones
	}
}

func parseFromYAML(filename string, out interface{}) error {
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(file, out)
	if err != nil {
		return err
	}

	return nil
}

func parseFromHTTP(endpoint string, out interface{}) (err error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return
	}

	req, err := http.NewRequest(
		http.MethodGet,
		u.String(),
		nil,
	)
	if err != nil {
		return
	}

	client := &http.Client{
		Timeout: time.Duration(60) * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	err = yaml.Unmarshal(body, out)
	if err != nil {
		return
	}
	return
}

func schemaCheck(str string) (string, error) {
	if strings.HasPrefix(str, "http://") || strings.HasPrefix(str, "https://") {
		return SchemaHTTP, nil
	} else if strings.HasSuffix(str, ".yaml") || strings.HasSuffix(str, ".yml") {
		return SchemaYAML, nil
	}
	return "", fmt.Errorf("unknown schema: %s", str)
}
