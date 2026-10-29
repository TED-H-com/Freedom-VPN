package config

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"os"
	"strconv"
	"strings"
)

// DNS is the resolver deployment file (decision D-028). Upstreams are injected
// here. There is no production default and the compiler never invents one.
type DNS struct {
	Upstreams    []string `json:"upstreams"`
	BootstrapDNS []string `json:"bootstrap_dns"`
}

// Validate requires at least one upstream. Each address is an IPv4 or IPv6
// literal with a port in 1..65535. An empty bootstrap list means the publisher
// copies the configured upstreams; it is not a public resolver.
func (c *DNS) Validate() error {
	upstreams, err := resolverList("upstreams", c.Upstreams, true)
	if err != nil {
		return err
	}
	bootstrap, err := resolverList("bootstrap_dns", c.BootstrapDNS, false)
	if err != nil {
		return err
	}
	c.Upstreams = upstreams
	c.BootstrapDNS = bootstrap
	return nil
}

// LoadDNS reads one strict JSON file. The path itself is not included in errors.
func LoadDNS(path string) (DNS, error) {
	var c DNS
	f, err := os.Open(path)
	if err != nil {
		return c, errors.New("cannot open dns configuration")
	}
	defer f.Close()
	d := json.NewDecoder(io.LimitReader(f, 1<<20))
	d.DisallowUnknownFields()
	if d.Decode(&c) != nil {
		return c, errors.New("invalid dns configuration")
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return c, errors.New("trailing dns configuration")
	}
	return c, c.Validate()
}

func resolverList(field string, in []string, required bool) ([]string, error) {
	if required && len(in) == 0 {
		return nil, errors.New(field + " requires at least one resolver")
	}
	out := make([]string, 0, len(in))
	seen := make(map[string]struct{}, len(in))
	for i, raw := range in {
		addr, err := resolverAddr(raw)
		if err != nil {
			return nil, errors.New(field + "[" + strconv.Itoa(i) + "] must be an IPv4 or IPv6 literal with a port")
		}
		if _, ok := seen[addr]; ok {
			return nil, errors.New(field + " contains a duplicate resolver")
		}
		seen[addr] = struct{}{}
		out = append(out, addr)
	}
	return out, nil
}

func resolverAddr(raw string) (string, error) {
	host, port, err := net.SplitHostPort(strings.TrimSpace(raw))
	if err != nil {
		return "", err
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return "", errors.New("not an IP literal")
	}
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 {
		return "", errors.New("port out of range")
	}
	if v4 := ip.To4(); v4 != nil {
		return v4.String() + ":" + strconv.Itoa(n), nil
	}
	return "[" + ip.String() + "]:" + strconv.Itoa(n), nil
}

// String and LogValue keep deployment addresses out of accidental logs.
func (c DNS) String() string       { return "[REDACTED dns configuration]" }
func (c DNS) LogValue() slog.Value { return slog.StringValue(c.String()) }
