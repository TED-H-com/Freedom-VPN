package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/netip"
	"os"

	"github.com/TED-H-com/Freedom-VPN/tests/netlab"
)

func injectCmd(args []string) error {
	fs := flag.NewFlagSet("inject", flag.ContinueOnError)
	iface := fs.String("iface", "", "member interface whose ingress receives the frame")
	src := fs.String("src", "", "source IPv4 carried as the member address")
	dst := fs.String("dst", "", "destination IPv4 beyond the node")
	port := fs.Uint("port", 9, "UDP destination port")
	mechanism := fs.String("mechanism", "", "empty selects from the interface type; veth-mirred or tun-mirred forces one")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *port > 65535 {
		return fmt.Errorf("port %d is out of range", *port)
	}
	source, err := netip.ParseAddr(*src)
	if err != nil {
		return fmt.Errorf("src: %w", err)
	}
	dest, err := netip.ParseAddr(*dst)
	if err != nil {
		return fmt.Errorf("dst: %w", err)
	}
	obs, err := netlab.InjectMemberIngress(context.Background(), netlab.IngressSpec{
		MemberIface: *iface,
		Source:      source,
		Dest:        dest,
		DestPort:    uint16(*port),
		Mechanism:   *mechanism,
	})
	if err != nil {
		return err
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(obs)
}
