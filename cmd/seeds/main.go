package main

import (
	"github.com/ardikabs/views"
)

func main() {
	soa := views.SOA{
		MName:            "ns1.example.internal",
		RName:            "admin.example.internal",
		Serial:           2021010806,
		Refresh:          10800,
		Retry:            3600,
		Expire:           2419200,
		NegativeCacheTTL: 3600,
	}

	records := []views.Record{
		views.Record{
			Name:   "@",
			TTL:    500,
			QClass: views.ClassINET,
			QType:  views.TypeNS,
			Value:  "ns1.example.internal",
		},
		views.Record{
			Name:   "ns1",
			TTL:    300,
			QClass: views.ClassINET,
			QType:  views.TypeA,
			Value:  "100.64.0.28",
		},
		views.Record{
			Name:   "db",
			TTL:    60,
			QClass: views.ClassINET,
			QType:  views.TypeA,
			Value:  "100.64.0.100",
		},
		views.Record{
			Name:   "db-dc1",
			TTL:    60,
			QClass: views.ClassINET,
			QType:  views.TypeA,
			Value:  "100.64.1.100",
		},
		views.Record{
			Name:   "web-dc1",
			TTL:    60,
			QClass: views.ClassINET,
			QType:  views.TypeA,
			Value:  "100.64.1.101",
		},
		views.Record{
			Name:   "db-dc2",
			TTL:    60,
			QClass: views.ClassINET,
			QType:  views.TypeAAAA,
			Value:  "2001:db8::68",
		},
		views.Record{
			Name:   "db-dc3",
			TTL:    60,
			QClass: views.ClassINET,
			QType:  views.TypeAAAA,
			Value:  "fe80::ac5d:ccff:fee2:881",
		},
	}

	origins := []views.Origin{
		views.Origin{
			Name:       "example.internal",
			DefaultTTL: 300,
			SOA:        soa,
			Records:    records,
		},
	}

	filepath := string("/Users/supermonster/Workspaces/workshop/coredns/views/data/generated")

	for _, o := range origins {
		views.Render(filepath, o)
	}
}
