package config

import (
	"fmt"
	"net"
)

// ParseTrustedProxies checks trusted_proxies and returns the networks a TCP
// peer must belong to before X-Forwarded-For is read. An empty list trusts
// nobody. 0.0.0.0/0, ::/0, and any shorter prefix than /8 (IPv4) or /32
// (IPv6) are rejected: trusting the world lets any peer choose its bucket.
// A proxy that connects from several addresses is several CIDRs in this list.
func ParseTrustedProxies(cidrs []string) ([]*net.IPNet, error) {
	if len(cidrs) == 0 {
		return nil, nil
	}
	nets := make([]*net.IPNet, 0, len(cidrs))
	for _, raw := range cidrs {
		n, err := parseTrustedProxy(raw)
		if err != nil {
			return nil, err
		}
		nets = append(nets, n)
	}
	return nets, nil
}

func parseTrustedProxy(raw string) (*net.IPNet, error) {
	ip, network, err := net.ParseCIDR(raw)
	if err != nil || ip == nil || network == nil {
		return nil, fmt.Errorf("trusted_proxies: %q is not a CIDR", raw)
	}
	ones, bits := network.Mask.Size()
	if ones < 0 {
		return nil, fmt.Errorf("trusted_proxies: %q is not a CIDR", raw)
	}
	if ip.To4() != nil && bits == 128 {
		return nil, fmt.Errorf("trusted_proxies: %q is an IPv4-mapped CIDR; write it as an IPv4 CIDR", raw)
	}
	tooBroad := (bits == 32 && ones < 8) || (bits == 128 && ones < 32) || (bits != 32 && bits != 128)
	if tooBroad {
		return nil, fmt.Errorf("trusted_proxies: prefix %q is too broad; trusting the world lets any peer choose its bucket", raw)
	}
	if bits == 32 {
		v4 := ip.To4()
		if v4 == nil {
			return nil, fmt.Errorf("trusted_proxies: %q is not a CIDR", raw)
		}
		mask := net.CIDRMask(ones, 32)
		network.IP = v4.Mask(mask)
		network.Mask = mask
	}
	return network, nil
}

func (c Config) validateTrustedProxies() error {
	_, err := ParseTrustedProxies(c.TrustedProxies)
	return err
}
