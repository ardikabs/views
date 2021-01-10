package main

import (
	_ "github.com/ardikabs/simpledns"

	_ "github.com/coredns/coredns/plugin/errors"
	_ "github.com/coredns/coredns/plugin/file"
	_ "github.com/coredns/coredns/plugin/forward"
	_ "github.com/coredns/coredns/plugin/health"
	_ "github.com/coredns/coredns/plugin/loadbalance"
	_ "github.com/coredns/coredns/plugin/log"
	_ "github.com/coredns/coredns/plugin/loop"
	_ "github.com/coredns/coredns/plugin/metrics"
	_ "github.com/coredns/coredns/plugin/ready"
	_ "github.com/coredns/coredns/plugin/reload"
	_ "github.com/coredns/coredns/plugin/transfer"

	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/coremain"
)

var directives = []string{
	"reload",
	"ready",
	"health",
	"prometheus",
	"errors",
	"log",
	"transfer",
	"simpledns",
	"loadbalance",
	"file",
	"loop",
	"forward",
}

func init() {
	dnsserver.Directives = directives
}

func main() {
	coremain.Run()
}
