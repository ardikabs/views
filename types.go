package views

type (
	// Origin represent of zone origin following on RFC 1035-style
	Origin struct {
		Name       string
		DefaultTTL uint32
		SOA        SOA
		Records    []Record
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
	// SchemaYAML represent of HTTP schema
	SchemaHTTP = "http"
)
