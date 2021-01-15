package main

import (
	"github.com/ardikabs/views"
)

func main() {
	soa := views.SOA{
		MName:            "ns1.example.internal",
		RName:            "admin.example.internal",
		Serial:           2021010806,
		Refresh:          10000,
		Retry:            2400,
		Expire:           604800,
		NegativeCacheTTL: 3600,
	}

	records := []views.Record{
		{
			Name:   "@",
			TTL:    500,
			QClass: views.ClassINET,
			QType:  views.TypeNS,
			Value:  "ns1.example.internal",
		},
		{
			Name:   "ns1",
			TTL:    300,
			QClass: views.ClassINET,
			QType:  views.TypeA,
			Value:  "100.64.0.28",
		},
		{
			Name:   "db",
			TTL:    60,
			QClass: views.ClassINET,
			QType:  views.TypeA,
			Value:  "100.64.0.100",
		},
		{
			Name:   "db-dc1",
			TTL:    60,
			QClass: views.ClassINET,
			QType:  views.TypeA,
			Value:  "100.64.1.100",
		},
		{
			Name:   "web-dc1",
			TTL:    60,
			QClass: views.ClassINET,
			QType:  views.TypeA,
			Value:  "100.64.1.101",
		},
		{
			Name:   "db-dc2",
			TTL:    60,
			QClass: views.ClassINET,
			QType:  views.TypeAAAA,
			Value:  "2001:db8::68",
		},
		{
			Name:   "db-dc3",
			TTL:    60,
			QClass: views.ClassINET,
			QType:  views.TypeAAAA,
			Value:  "fe80::ac5d:ccff:fee2:881",
		},
		{
			Name:   "*.cdn",
			TTL:    60,
			QClass: views.ClassINET,
			QType:  views.TypeA,
			Value:  "192.168.100.1",
		},
	}

	origins := []views.Origin{
		{
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
