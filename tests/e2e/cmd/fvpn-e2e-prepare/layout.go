package main

import (
	"fmt"
	"regexp"
)

// Port offsets from FVPN_E2E_PORT_BASE. The largest is postgres (5432), so two
// stacks need bases at least 6000 apart. Existing slots stay fixed; 12000 and 6000 use two new subnets.
const (
	portBaseOrigin = 18000
	portBaseStep   = 6000
	portBaseSlots  = 8
)

var portOffsets = map[string]int{
	"control":  443,
	"agent":    1443,
	"idp":      444,
	"origin":   445,
	"payment":  446,
	"gateway":  480,
	"stun":     3478,
	"postgres": 5432,
}

var prefixPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{0,20}$`)

type layout struct {
	Prefix       string
	PortBase     int
	LabPrefix    string
	OriginPrefix string
	OriginIP     string
	AdGuardIP    string
	CoreDNSIP    string
	Ports        map[string]int
}

func parseLayout(prefix string, portBase int) (layout, error) {
	if !prefixPattern.MatchString(prefix) || prefix[len(prefix)-1] == '-' {
		return layout{}, fmt.Errorf("prefix must be a short lowercase name")
	}
	if portBase < 6000 || (portBase-portBaseOrigin)%portBaseStep != 0 {
		return layout{}, fmt.Errorf("port base must be 6000 plus a multiple of %d", portBaseStep)
	}
	slot := (portBase - portBaseOrigin) / portBaseStep
	if slot >= portBaseSlots {
		return layout{}, fmt.Errorf("port base slot is out of range")
	}
	// Append the lower port slots; never renumber any running stack.
	if slot < 0 {
		slot = portBaseSlots - 1 - slot
	}
	lab := fmt.Sprintf("172.28.%d", 30+slot)
	origin := fmt.Sprintf("192.0.%d", 3+slot)
	ports := make(map[string]int, len(portOffsets))
	for name, offset := range portOffsets {
		ports[name] = portBase + offset
	}
	return layout{
		Prefix: prefix, PortBase: portBase,
		LabPrefix: lab, OriginPrefix: origin,
		OriginIP: origin + ".10", AdGuardIP: lab + ".53", CoreDNSIP: lab + ".54",
		Ports: ports,
	}, nil
}
