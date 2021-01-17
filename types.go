package views

import (
	"fmt"
	"net"
	"strings"

	"github.com/coredns/coredns/plugin"
	"github.com/miekg/dns"
)

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

	// SOA represent of SOA record
	SOA struct {
		MName            string
		RName            string
		Serial           uint
		Refresh          uint16
		Retry            uint16
		Expire           uint32
		NegativeCacheTTL uint16
	}

	// Record represent of single record on origin
	Record struct {
		Name   string
		TTL    uint32
		QType  string
		QClass string
		Value  string
	}

	// RawClientACL represent specification of Client ACL YAML-file
	RawClientACL struct {
		Name         string   `yaml:"name" json:"name"`
		CIDRPrefixes []string `yaml:"prefixes" json:"prefixes"`
	}

	// RawRecord represent specification of Record YAML-file
	RawRecord struct {
		Name    string          `yaml:"name" json:"name"`
		Records []RawRecordUnit `yaml:"records" json:"records"`
	}

	// RawRecordUnit represent a smallest unit of Record YAML-file
	RawRecordUnit struct {
		Name  string `yaml:"name" json:"name"`
		TTL   uint32 `yaml:"ttl" json:"ttl"`
		Type  string `yaml:"type" json:"type"`
		Value string `yaml:"value" json:"value"`
	}
)

const (
	// TypeA represent of DNS RR of A
	TypeA = "A"
	// TypeAAAA represent of DNS RR of AAAA
	TypeAAAA = "AAAA"
	// TypeCNAME represent of DNS RR of CNAME
	TypeCNAME = "CNAME"
	// TypeTXT represent of DNS RR of TXT
	TypeTXT = "TXT"
	// TypeSOA represent of DNS RR of SOA
	TypeSOA = "SOA"
	// TypeNS represent of DNS RR of NS
	TypeNS = "NS"

	// ClassINET represent of DNS RR Class of IN
	ClassINET = "IN"

	// SchemaYAML represent of YAML schema
	SchemaYAML = "yaml"

	// SchemaHTTP represent of HTTP schema
	SchemaHTTP = "http"
)

// NewZoneRecord is method to create new zone record from raw record unit
func NewZoneRecord(record RawRecordUnit) (Zone, error) {
	t := strings.ToUpper(record.Type)
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
	default:
		return Zone{}, fmt.Errorf("unknown type for record %s: \"%s\"", record.Name, t)
	}

	return Zone{
		Name:  plugin.Host(record.Name).Normalize(),
		TTL:   record.TTL,
		Type:  rrtype,
		Value: plugin.Host(record.Value).Normalize(),
	}, nil
}
