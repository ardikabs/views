package main

import (
	_ "github.com/ardikabs/views"
	_ "github.com/coredns/coredns/core/plugin"

	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/coremain"
)

func init() {

	var i int
	for i = 0; i < len(dnsserver.Directives); i++ {
		if dnsserver.Directives[i] == "loadbalance" {
			break
		}
	}

	dnsserver.Directives = append(dnsserver.Directives, "")
	copy(dnsserver.Directives[i+1:], dnsserver.Directives[i:])
	dnsserver.Directives[i] = "views"
}

func main() {
	coremain.Run()
}
