package updater

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/netip"
	"net/url"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/internal/healthattest"
)

const defaultProbeAgentConfig = "/etc/vpn-node/agent.json"

// These seams keep tests independent of host interfaces and external DNS.
type probeAddressSources struct {
	interfaces func() ([]net.Addr, error)
	lookup     func(context.Context, string, string) ([]netip.Addr, error)
}

type probeEnvironment struct {
	agentConfig, resultURL string
	sources                probeAddressSources
}

func (c Config) probeEnvironment() probeEnvironment {
	return probeEnvironment{agentConfig: c.AgentConfig, resultURL: c.ResultURL, sources: c.probeSources}
}

// Read only the address-bearing subset of agent.json. Unknown agent fields
// belong to the agent; in particular, resolver credentials are never opened.
type probeAgentConfig struct {
	ControlURL string `json:"control_url"`
	DNS        *struct {
		Resolvers []struct {
			AdminURL string `json:"admin_url"`
			DNSAddr  string `json:"dns_addr"`
		} `json:"resolvers"`
	} `json:"dns"`
}

func readProbeAgentConfig(path string) (probeAgentConfig, error) {
	var cfg probeAgentConfig
	if path == "" {
		path = defaultProbeAgentConfig
	}
	lst, err := os.Lstat(path)
	if err != nil || !lst.Mode().IsRegular() {
		return cfg, probeRefusal("agent_config_unavailable")
	}
	return readProbeAgentFile(path, lst)
}

// lst is the pre-open identity. NOFOLLOW rejects a substituted symlink;
// NONBLOCK prevents a substituted FIFO from hanging before the fstat check.
func readProbeAgentFile(path string, lst os.FileInfo) (probeAgentConfig, error) {
	var cfg probeAgentConfig
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return cfg, probeRefusal("agent_config_unavailable")
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil || !st.Mode().IsRegular() || !os.SameFile(lst, st) {
		return cfg, probeRefusal("agent_config_unavailable")
	}
	const maxBytes = 1 << 20
	b, err := io.ReadAll(io.LimitReader(f, maxBytes+1))
	if err != nil || len(b) > maxBytes || json.Unmarshal(b, &cfg) != nil {
		return cfg, probeRefusal("agent_config_unavailable")
	}
	return cfg, nil
}

func (env probeEnvironment) validate(ctx context.Context, p *healthattest.ChallengeProbe) error {
	if p == nil {
		return nil
	}
	// Refuse malformed/denied sockets before reading files or resolving names.
	if err := validateProbePeer(p, probeAddresses{}); err != nil {
		return err
	}
	known, err := env.addresses(ctx)
	if err != nil {
		return err
	}
	return validateProbePeer(p, known)
}

func (env probeEnvironment) addresses(ctx context.Context) (probeAddresses, error) {
	var known probeAddresses
	// One deadline covers all DNS lookups, including multiple resolver hosts.
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	cfg, err := readProbeAgentConfig(env.agentConfig)
	if err != nil {
		return known, err
	}
	interfaces := env.sources.interfaces
	if interfaces == nil {
		interfaces = net.InterfaceAddrs
	}
	addrs, err := interfaces()
	if err != nil {
		return known, probeRefusal("node_addresses_unavailable")
	}
	for _, a := range addrs {
		var ip net.IP
		switch v := a.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		}
		addr, ok := netip.AddrFromSlice(ip)
		if !ok {
			return known, probeRefusal("node_addresses_unavailable")
		}
		addr = addr.Unmap()
		if addr.Is4() && addr.IsGlobalUnicast() {
			known.node = append(known.node, addr)
		}
	}
	if len(known.node) == 0 {
		return known, probeRefusal("node_addresses_unavailable")
	}
	lookup := env.sources.lookup
	if lookup == nil {
		lookup = net.DefaultResolver.LookupNetIP
	}
	resolve := func(host string) ([]netip.Addr, error) {
		if addr, err := netip.ParseAddr(host); err == nil {
			return []netip.Addr{addr.Unmap()}, nil
		}
		addrs, err := lookup(ctx, "ip", host)
		if err != nil || len(addrs) == 0 {
			return nil, errProbePeer
		}
		out := make([]netip.Addr, 0, len(addrs))
		for _, addr := range addrs {
			if !addr.IsValid() {
				return nil, errProbePeer
			}
			out = append(out, addr.Unmap())
		}
		return out, nil
	}
	controlURLs := []string{cfg.ControlURL}
	if env.resultURL != "" {
		controlURLs = append(controlURLs, env.resultURL)
	}
	for _, raw := range controlURLs {
		host, ok := probeURLHost(raw)
		if !ok {
			return known, probeRefusal("control_addresses_unavailable")
		}
		addrs, err := resolve(host)
		if err != nil {
			return known, probeRefusal("control_addresses_unavailable")
		}
		known.control = append(known.control, addrs...)
	}
	if cfg.DNS != nil {
		for _, r := range cfg.DNS.Resolvers {
			admin, ok := probeURLHost(r.AdminURL)
			dns, port, err := net.SplitHostPort(r.DNSAddr)
			n, perr := strconv.Atoi(port)
			if !ok || err != nil || dns == "" || perr != nil || n < 1 || n > 65535 {
				return known, probeRefusal("resolver_addresses_unavailable")
			}
			for _, host := range []string{admin, dns} {
				addrs, err := resolve(host)
				if err != nil {
					return known, probeRefusal("resolver_addresses_unavailable")
				}
				known.resolvers = append(known.resolvers, addrs...)
			}
		}
	}
	return known, nil
}

func probeURLHost(raw string) (string, bool) {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil {
		return "", false
	}
	return u.Hostname(), true
}

func (e *Engine) challengeProbe(ctx context.Context) (*healthattest.ChallengeProbe, *healthattest.ProbeOmission) {
	p := cloneProbePeer(e.ProbePeer)
	if err := e.probeEnv.validate(ctx, p); err != nil {
		// validate returns only fixed probeRefusal values; no raw file or DNS
		// error, address, peer ID or config path may enter the log.
		reason := err.(probeRefusal)
		if e.probeRefusals == nil {
			e.probeRefusals = make(map[probeRefusal]bool)
		}
		if !e.probeRefusals[reason] {
			e.log().Warn("health challenge probe omitted", "reason", string(reason))
			e.probeRefusals[reason] = true
		}
		peerID := p.PeerID
		// Loaded configurations have bounded peer IDs. Also bound diagnostics
		// when an embedded caller supplies a malformed peer directly.
		if strings.TrimSpace(peerID) == "" || len(peerID) > 128 || strings.ContainsFunc(peerID, func(r rune) bool { return r < 0x20 || r == 0x7f }) {
			peerID = "invalid_peer_id"
		}
		return nil, &healthattest.ProbeOmission{PeerID: peerID, Reason: string(reason)}
	}
	return p, nil
}
