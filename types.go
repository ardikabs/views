package views

type (
	Origin struct {
		Name       string
		DefaultTTL uint32
		SOA        SOA
		Records    []Record
	}

	SOA struct {
		MName            string
		RName            string
		Serial           uint
		Refresh          uint16
		Retry            uint16
		Expire           uint32
		NegativeCacheTTL uint16
	}

	Record struct {
		Name   string
		TTL    uint32
		QType  string
		QClass string
		Value  string
	}
)

const (
	TypeA     = "A"
	TypeAAAA  = "AAAA"
	TypeCNAME = "CNAME"
	TypeTXT   = "TXT"
	TypeSOA   = "SOA"
	TypeNS    = "NS"

	ClassINET = "IN"
)
