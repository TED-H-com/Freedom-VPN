// Command fvpn-e2e-fixture-probe performs bounded, content-free TEST ONLY probes.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/streamstats"
	"io"
	"net"
	"net/netip"
	"os"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/dnspolicy"
	"golang.org/x/net/dns/dnsmessage"
)

type result struct {
	Endpoint       *streamstats.Sample `json:"endpoint,omitempty"`
	AuthorityTTL   uint32              `json:"authority_ttl"`
	HasSOA         bool                `json:"has_soa"`
	Available      bool                `json:"available"`
	RCode          int                 `json:"rcode"`
	Types          []uint16            `json:"types"`
	TTLs           []uint32            `json:"ttls"`
	Classes        []string            `json:"address_classes"`
	ExpectedAnswer bool                `json:"expected_answer"`
	Connected      bool                `json:"connected"`
	Bytes          int                 `json:"bytes"`
	Milliseconds   int64               `json:"milliseconds"`
	Error          string              `json:"error,omitempty"`
}

func class(ip netip.Addr) string {
	if ip.Is4() {
		if ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
			return "private_ipv4"
		}
		return "other_ipv4"
	}
	if ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
		return "private_ipv6"
	}
	return "other_ipv6"
}
func main() {
	mode := flag.String("mode", "dns", "dns, lookup, connect, listen, bulk")
	addr := flag.String("addr", "", "controlled address or empty for system nameserver")
	name := flag.String("name", "", "synthetic DNS name")
	typ := flag.Int("type", 1, "RR type")
	proto := flag.String("proto", "udp", "DNS transport")
	expected := flag.String("expect", "", "expected synthetic address; never emitted")
	device := flag.String("device", "", "member echo device ID")
	timeout := flag.Duration("timeout", 12*time.Second, "bounded operation deadline")
	flag.Parse()
	if *timeout <= 0 || *timeout > 30*time.Second {
		os.Exit(2)
	}
	if *mode == "listen" {
		ln, e := net.Listen("tcp", *addr)
		if e != nil {
			os.Exit(1)
		}
		defer ln.Close()
		for {
			c, e := ln.Accept()
			if e != nil {
				return
			}
			_ = c.Close()
		}
	}
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	out := result{RCode: -1, Types: []uint16{}, TTLs: []uint32{}, Classes: []string{}}
	var trace *streamstats.Conn
	switch *mode {
	case "dns":
		if !strings.HasSuffix(strings.TrimSuffix(strings.ToLower(*name), "."), ".example.invalid") {
			os.Exit(2)
		}
		server := *addr
		if server == "" {
			raw, e := os.ReadFile("/etc/resolv.conf")
			if e == nil {
				for _, line := range strings.Split(string(raw), "\n") {
					f := strings.Fields(line)
					if len(f) == 2 && f[0] == "nameserver" {
						server = net.JoinHostPort(f[1], "53")
						break
					}
				}
			}
		}
		response, e := dnspolicy.NewDNSClient(*timeout).QueryProto(ctx, *proto, server, *name, dnsmessage.Type(*typ))
		if e != nil {
			out.Error = "dns_unavailable"
			break
		}
		out.Available = true
		out.RCode = int(response.RCode)
		out.AuthorityTTL = response.AuthorityTTL
		out.HasSOA = response.HasSOA
		for _, a := range response.Answers {
			out.Types = append(out.Types, uint16(a.Type))
			out.TTLs = append(out.TTLs, a.TTL)
			if a.IP.IsValid() {
				out.Classes = append(out.Classes, class(a.IP))
				out.ExpectedAnswer = out.ExpectedAnswer || a.IP.String() == *expected
			}
		}
	case "lookup":
		if !strings.HasSuffix(*name, ".example.invalid") {
			os.Exit(2)
		}
		ips, e := net.DefaultResolver.LookupNetIP(ctx, "ip", *name)
		if e != nil {
			out.Error = "lookup_unavailable"
			break
		}
		out.Available = true
		for _, ip := range ips {
			out.Classes = append(out.Classes, class(ip))
			out.ExpectedAnswer = out.ExpectedAnswer || ip.String() == *expected
		}
		if *addr != "" {
			c, e := (&net.Dialer{}).DialContext(ctx, "tcp", net.JoinHostPort(*name, *addr))
			if e == nil {
				out.Connected = true
				_ = c.Close()
			}
		}
	case "connect":
		c, e := (&net.Dialer{}).DialContext(ctx, "tcp", *addr)
		if e == nil {
			out.Available = true
			out.Connected = true
			_ = c.Close()
		} else {
			out.Error = "connect_unavailable"
		}
	case "bulk":
		c, e := (&net.Dialer{}).DialContext(ctx, "tcp", *addr)
		if e != nil {
			out.Error = "bulk_connect"
			break
		}
		defer c.Close()
		trace = streamstats.Wrap(c)
		c = trace
		_ = c.SetDeadline(start.Add(*timeout))
		reader := bufio.NewReader(c)
		_, e = fmt.Fprintf(c, "MEMBER %s\n", *device)
		if e != nil {
			break
		}
		hello, e := reader.ReadString('\n')
		if e != nil || hello != "OK\n" {
			out.Error = "bulk_hello"
			break
		}
		payload := strings.Repeat("x", 1000) + "\n"
		if _, e = io.WriteString(c, strings.Repeat(payload, 64)); e != nil {
			out.Error = "bulk_write"
			break
		}
		for i := 0; i < 64; i++ {
			line, readErr := reader.ReadString('\n')
			trace.Line(len(line), readErr == nil && line == "ECHO "+payload, readErr)
			if readErr != nil || line != "ECHO "+payload {
				e = fmt.Errorf("echo mismatch")
				break
			}
			out.Bytes += len(payload)
		}
		out.Available = e == nil && out.Bytes == 64064
		if !out.Available {
			out.Error = "bulk_echo_incomplete"
		}
		out.Connected = true
	default:
		os.Exit(2)
	}
	if trace != nil {
		sample := trace.Snapshot()
		out.Endpoint = &sample
	}
	out.Milliseconds = time.Since(start).Milliseconds()
	_ = json.NewEncoder(os.Stdout).Encode(out)
}
