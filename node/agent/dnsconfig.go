package agent

import (
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"

	"github.com/TED-H-com/Freedom-VPN/node/dns"
)

const (
	maxDNSResolvers      = 8
	maxResolverCredBytes = 4096
	maxResolverSecret    = 256

	// resolverCredDetail is the only operator text for a credential file
	// the process will not use. It has no path, URL, username, or password.
	resolverCredDetail = "Resolver credentials are not usable."
)

var (
	errDNSBoth = errors.New("agent: dns resolvers and resolver_credentials_file must both be set")
	errDNSHost = errors.New("agent: dns resolver is not a host-local address")
	errDNSID   = errors.New("agent: dns resolver id is not usable")
	errDNSMany = errors.New("agent: dns has too many resolvers")
	errDNSPath = errors.New("agent: dns resolver_credentials_file is not usable")

	errResolverCred = errors.New("agent: resolver credentials are not usable")

	resolverIDRE = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,31}$`)
)

// DNSConfig is the optional dns object in agent.json (D-061).
// Absent leaves the resolver applier unset. A referenced document then
// fails closed and does not open an AdGuard client.
type DNSConfig struct {
	Resolvers               []DNSResolver `json:"resolvers"`
	ResolverCredentialsFile string        `json:"resolver_credentials_file"`
}

// DNSResolver is one AdGuard target. AdminURL is the admin origin.
// DNSAddr is host:port of the DNS listener. dns.New requires both, and
// the admin port is not the DNS port.
type DNSResolver struct {
	ID       string `json:"id,omitempty"`
	AdminURL string `json:"admin_url"`
	DNSAddr  string `json:"dns_addr"`
}

type resolverCredentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (c Config) validateDNS() error {
	if c.DNS == nil {
		return nil
	}
	d := c.DNS
	if len(d.Resolvers) == 0 || strings.TrimSpace(d.ResolverCredentialsFile) == "" {
		return errDNSBoth
	}
	if len(d.Resolvers) > maxDNSResolvers {
		return errDNSMany
	}
	if err := validCredPath(d.ResolverCredentialsFile); err != nil {
		return err
	}
	seen := map[string]struct{}{}
	for _, r := range d.Resolvers {
		if r.ID != "" {
			if !resolverIDRE.MatchString(r.ID) {
				return errDNSID
			}
			if _, ok := seen[r.ID]; ok {
				return errDNSID
			}
			seen[r.ID] = struct{}{}
		}
		if err := parseAdminOrigin(r.AdminURL); err != nil {
			return errDNSHost
		}
		if err := parseDNSAddr(r.DNSAddr); err != nil {
			return errDNSHost
		}
	}
	return nil
}

func validCredPath(p string) error {
	if p == "" || len(p) > 4096 || strings.ContainsRune(p, 0) || strings.Contains(p, "://") || !filepath.IsAbs(p) {
		return errDNSPath
	}
	return nil
}

func parseAdminOrigin(raw string) error {
	if raw == "" || strings.TrimSpace(raw) != raw || strings.ContainsAny(raw, " \t") {
		return errDNSHost
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.Hostname() == "" {
		return errDNSHost
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return errDNSHost
	}
	if u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Opaque != "" {
		return errDNSHost
	}
	if u.Path != "" && u.Path != "/" {
		return errDNSHost
	}
	_, port, err := net.SplitHostPort(u.Host)
	if err != nil || !portOK(port) {
		return errDNSHost
	}
	return parseHostIP(u.Hostname())
}

func parseDNSAddr(raw string) error {
	if raw == "" || strings.TrimSpace(raw) != raw {
		return errDNSHost
	}
	host, port, err := net.SplitHostPort(raw)
	if err != nil || !portOK(port) {
		return errDNSHost
	}
	return parseHostIP(host)
}

func portOK(port string) bool {
	n, err := strconv.Atoi(port)
	return err == nil && n >= 1 && n <= 65535 && strconv.Itoa(n) == port
}

func parseHostIP(host string) error {
	if host == "" || strings.Contains(host, "%") {
		return errDNSHost
	}
	ip, err := netip.ParseAddr(host)
	if err != nil || !ip.IsValid() || ip.Zone() != "" || ip.Is4In6() {
		return errDNSHost
	}
	if !hostAssigned(ip) {
		return errDNSHost
	}
	return nil
}

// hostAssigned allows loopback, and a link-local or private address only
// when that address is configured on a host interface. A public,
// documentation, multicast, or unspecified address is rejected. If the
// interface list cannot be read, only loopback is allowed.
func hostAssigned(ip netip.Addr) bool {
	if !ip.IsValid() || ip.Zone() != "" || ip.Is4In6() || ip.IsUnspecified() || ip.IsMulticast() {
		return false
	}
	if ip.IsLoopback() {
		return true
	}
	if !ip.IsPrivate() && !ip.IsLinkLocalUnicast() {
		return false
	}
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return false
	}
	for _, a := range addrs {
		var parsed netip.Addr
		var ok bool
		switch v := a.(type) {
		case *net.IPNet:
			parsed, ok = netip.AddrFromSlice(v.IP)
		case *net.IPAddr:
			parsed, ok = netip.AddrFromSlice(v.IP)
		default:
			continue
		}
		if ok && parsed.IsValid() && parsed.Zone() == "" && parsed.Unmap() == ip.Unmap() {
			return true
		}
	}
	return false
}

// reloadResolvers loads the credential file and builds the node/dns applier.
// A bad mode or owner drops any applier already held. Options.DNS, when
// set by a test, is left in place. The URL and the secret are not logged.
func (e *Engine) reloadResolvers() {
	if e == nil || e.dnsPinned {
		return
	}
	if e.cfg.DNS == nil {
		e.dnsApply = nil
		e.dnsCredDetail = ""
		return
	}
	user, pass, err := readResolverCredentials(e.cfg.DNS.ResolverCredentialsFile)
	if err != nil {
		e.refuseResolvers()
		return
	}
	targets := make([]dns.Target, len(e.cfg.DNS.Resolvers))
	for i, r := range e.cfg.DNS.Resolvers {
		targets[i] = dns.Target{
			ID:       r.ID,
			AdminURL: r.AdminURL,
			Username: user,
			Password: pass,
			DNSAddr:  r.DNSAddr,
		}
	}
	ap, err := dns.New(targets)
	if err != nil {
		e.refuseResolvers()
		return
	}
	e.dnsApply = ap
	e.dnsCredDetail = ""
}

func (e *Engine) refuseResolvers() {
	e.dnsApply = nil
	e.dnsCredDetail = resolverCredDetail
	if e.log != nil {
		e.log.Warn("resolver credentials are not usable")
	}
}

func readResolverCredentials(path string) (string, string, error) {
	lst, err := os.Lstat(path)
	if err != nil || !credentialMode(lst) {
		return "", "", errResolverCred
	}
	f, err := os.Open(path)
	if err != nil {
		return "", "", errResolverCred
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil || !credentialMode(st) || !sameStat(lst, st) {
		return "", "", errResolverCred
	}
	b, err := io.ReadAll(io.LimitReader(f, maxResolverCredBytes+1))
	if err != nil {
		return "", "", errResolverCred
	}
	return parseResolverCredentials(b)
}

func parseResolverCredentials(b []byte) (string, string, error) {
	if len(b) == 0 || len(b) > maxResolverCredBytes {
		return "", "", errResolverCred
	}
	dec := json.NewDecoder(bytesReader(b))
	dec.DisallowUnknownFields()
	var cred resolverCredentials
	if err := dec.Decode(&cred); err != nil {
		return "", "", errResolverCred
	}
	var extra any
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		return "", "", errResolverCred
	}
	if !credentialSecret(cred.Username) || !credentialSecret(cred.Password) {
		return "", "", errResolverCred
	}
	return cred.Username, cred.Password, nil
}

func credentialSecret(s string) bool {
	if s == "" || len(s) > maxResolverSecret {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < 0x20 || s[i] == 0x7f {
			return false
		}
	}
	return true
}

func credentialMode(fi os.FileInfo) bool {
	if fi == nil || !fi.Mode().IsRegular() || fi.Mode().Perm() != 0o600 {
		return false
	}
	if fi.Mode()&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) != 0 {
		return false
	}
	st, ok := fi.Sys().(*syscall.Stat_t)
	return ok && st != nil && st.Uid == 0 && st.Gid == 0 && st.Nlink == 1
}

func sameStat(a, b os.FileInfo) bool {
	as, aok := a.Sys().(*syscall.Stat_t)
	bs, bok := b.Sys().(*syscall.Stat_t)
	if !aok || !bok || as == nil || bs == nil {
		return false
	}
	return as.Dev == bs.Dev && as.Ino == bs.Ino
}
