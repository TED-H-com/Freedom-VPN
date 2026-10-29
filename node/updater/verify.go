package updater

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/TED-H-com/Freedom-VPN/contracts"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

const releaseSchemaID = "urn:fvpn:contracts:schema:release-manifest-envelope:v1"

var (
	relSchemaOnce sync.Once
	relSchema     *jsonschema.Schema
	relSchemaErr  error
)

func releaseSchema() (*jsonschema.Schema, error) {
	relSchemaOnce.Do(func() {
		raw, err := fs.ReadFile(contracts.Schemas(), "release-manifest-envelope.schema.json")
		if err != nil {
			relSchemaErr = err
			return
		}
		doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
		if err != nil {
			relSchemaErr = err
			return
		}
		c := jsonschema.NewCompiler()
		c.DefaultDraft(jsonschema.Draft2020)
		c.AssertFormat()
		if err := c.AddResource(releaseSchemaID, doc); err != nil {
			relSchemaErr = err
			return
		}
		relSchema, relSchemaErr = c.Compile(releaseSchemaID)
	})
	return relSchema, relSchemaErr
}

func validateReleaseEnvelope(raw []byte) error {
	s, err := releaseSchema()
	if err != nil {
		return err
	}
	inst, err := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
	if err != nil {
		return err
	}
	return s.Validate(inst)
}

// VerifyManifest checks the envelope hash, the release signature and the
// schema, then returns the signed manifest (§13.4). manifestSHA is the
// SHA-256 of the canonical envelope.
func VerifyManifest(raw []byte, manifestSHA string, ts *envelope.TrustStore, now time.Time) (api.ReleaseManifest, error) {
	canon, err := canonjson.Canonicalize(raw)
	if err != nil {
		return api.ReleaseManifest{}, permanent(apierr.SignatureInvalid, "manifest is not valid JSON")
	}
	sum := sha256.Sum256(canon)
	if hex.EncodeToString(sum[:]) != strings.ToLower(manifestSHA) {
		return api.ReleaseManifest{}, permanent(apierr.HashMismatch, "manifest envelope")
	}
	if err := validateReleaseEnvelope(canon); err != nil {
		return api.ReleaseManifest{}, permanent(apierr.SignatureInvalid, "manifest schema")
	}
	signed, _, err := envelope.Verify(canon, ts, envelope.PurposeRelease, now)
	if err != nil {
		if errors.Is(err, envelope.ErrKeyUntrusted) {
			return api.ReleaseManifest{}, permanent(apierr.KeyUntrusted, "release signature")
		}
		return api.ReleaseManifest{}, permanent(apierr.SignatureInvalid, "release signature")
	}
	var m api.ReleaseManifest
	if err := json.Unmarshal(signed, &m); err != nil {
		return api.ReleaseManifest{}, permanent(apierr.SignatureInvalid, "manifest body")
	}
	if err := requireBothArch(m); err != nil {
		return api.ReleaseManifest{}, err
	}
	return m, nil
}

// requireBothArch is the rule the schema cannot express (§13.4): a release
// without both Ubuntu 24.04 architectures is not installable.
func requireBothArch(m api.ReleaseManifest) error {
	need := map[string]bool{"amd64": false, "arm64": false}
	for _, a := range m.Artifacts {
		if string(a.OsId) != "ubuntu" || string(a.Format) != "tar.zst" {
			continue
		}
		if _, ok := need[string(a.Architecture)]; !ok {
			continue
		}
		for _, v := range a.OsVersions {
			if v == "24.04" {
				need[string(a.Architecture)] = true
			}
		}
	}
	for arch, ok := range need {
		if !ok {
			return permanent(apierr.UnsupportedPlatform, "missing ubuntu 24.04 "+arch)
		}
	}
	return nil
}

// checkCompat rejects a manifest this node cannot run or roll back (§13.4,
// §13.7). currentSchema 0 means there is no previous state (first install).
func checkCompat(m api.ReleaseManifest, p Platform, currentSchema int, hasPrevious bool) error {
	if m.Compatibility.ControlApiMajor != p.ControlAPIMajor {
		return permanent(apierr.ControlAPIIncompatible, fmt.Sprintf("control api %d", m.Compatibility.ControlApiMajor))
	}
	if !intsContain(m.Compatibility.PolicySchemaVersions, p.PolicySchema) ||
		!intsContain(m.Compatibility.UsageEventSchemaVersions, p.UsageSchema) {
		return permanent(apierr.SchemaIncompatible, "policy or usage schema")
	}
	if !versionAtLeast(p.UpdaterVersion, m.Compatibility.MinimumUpdaterVersion) {
		return permanent(apierr.SchemaIncompatible, "minimum updater version")
	}
	if hasPrevious && !intsContain(m.Compatibility.StateSchemaFrom, currentSchema) {
		return permanent(apierr.SchemaIncompatible, "state schema from")
	}
	if hasPrevious && m.Compatibility.StateSchemaTo != currentSchema && !m.Rollback.StateReadableByPrevious {
		return permanent(apierr.SchemaIncompatible, "previous release cannot read the new state")
	}
	if _, err := selectArtifact(m, p.OSID, p.OSVersion, p.Architecture); err != nil {
		return err
	}
	return nil
}

func intsContain(xs []int, v int) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}

// urlPolicy is the artifact, manifest and result URL guard (D-034f, D-046).
// AllowedOrigins lists exact scheme://host[:port] origins. A matching origin
// is exempt from the private and CGNAT checks only. The default is empty,
// so the guard stands. allow_loopback still permits loopback and nothing else.
type urlPolicy struct {
	AllowLoopback  bool
	AllowedOrigins []string
}

// lookupIPs resolves a host. Tests replace it to prove the dial is pinned.
var lookupIPs = func(ctx context.Context, host string) ([]net.IP, error) {
	found, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}
	out := make([]net.IP, 0, len(found))
	for _, one := range found {
		out = append(out, one.IP)
	}
	return out, nil
}

// dialContextFn opens the connection to the already-checked address.
var dialContextFn = func(ctx context.Context, network, addr string) (net.Conn, error) {
	d := &net.Dialer{Timeout: 30 * time.Second}
	return d.DialContext(ctx, network, addr)
}

// checkURL allows https URLs and, only when allowLoopback is set, http to
// a loopback address for tests. Loopback, link-local, multicast, unspecified,
// private, CGNAT (100.64.0.0/10) and metadata hosts are refused (§16.2,
// D-034f). The same address check runs again on every address DialContext
// resolves, so a public name that answers with a private address is refused.
func checkURL(raw string, allowLoopback bool) error {
	return checkURLPolicy(raw, urlPolicy{AllowLoopback: allowLoopback})
}

func checkURLPolicy(raw string, policy urlPolicy) error {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.User != nil {
		return permanent(apierr.SignatureInvalid, "artifact url")
	}
	host := strings.ToLower(u.Hostname())
	if isMetadataHost(host) {
		return permanent(apierr.SignatureInvalid, "artifact url host")
	}
	allowPrivate := originAllowed(u, policy.AllowedOrigins)
	if ip := net.ParseIP(host); ip != nil {
		if err := rejectIP(ip, policy.AllowLoopback, allowPrivate); err != nil {
			return err
		}
	}
	if u.Scheme != "https" {
		if policy.AllowLoopback && u.Scheme == "http" && isLoopbackName(host) {
			return nil
		}
		return permanent(apierr.SignatureInvalid, "artifact url scheme")
	}
	return nil
}

func rejectIP(ip net.IP, allowLoopback, allowPrivate bool) error {
	if ip == nil {
		return permanent(apierr.SignatureInvalid, "artifact url address")
	}
	if ip.IsLoopback() {
		if allowLoopback {
			return nil
		}
		return permanent(apierr.SignatureInvalid, "artifact url loopback")
	}
	if !allowPrivate && (ip.IsPrivate() || isCGNAT(ip)) {
		return permanent(apierr.SignatureInvalid, "artifact url address")
	}
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
		return permanent(apierr.SignatureInvalid, "artifact url address")
	}
	return nil
}

// isCGNAT reports the IPv4 shared address space 100.64.0.0/10 (RFC 6598).
func isCGNAT(ip net.IP) bool {
	v4 := ip.To4()
	if v4 == nil {
		return false
	}
	return v4[0] == 100 && v4[1]&0xc0 == 0x40
}

// dialChecked resolves addr and refuses any private, CGNAT, link-local or
// otherwise forbidden address before it connects (D-034f). allowLoopback
// permits a loopback destination and nothing else.
func dialChecked(ctx context.Context, network, addr string, allowLoopback bool) (net.Conn, error) {
	return dialPolicy(ctx, network, addr, urlPolicy{AllowLoopback: allowLoopback})
}

// dialPolicy resolves addr, applies urlPolicy, and connects to the first
// resolved address. An allowlisted origin skips only the private and CGNAT
// checks when the request scheme, host and effective port match that entry,
// and the dial stays pinned to that resolution (D-046). The scheme comes
// from the request context. A missing scheme does not match.
func dialPolicy(ctx context.Context, network, addr string, policy urlPolicy) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, permanent(apierr.SignatureInvalid, "artifact url address")
	}
	var ips []net.IP
	if ip := net.ParseIP(host); ip != nil {
		ips = []net.IP{ip}
	} else {
		found, err := lookupIPs(ctx, host)
		if err != nil {
			return nil, err
		}
		if len(found) == 0 {
			return nil, permanent(apierr.SignatureInvalid, "artifact url address")
		}
		ips = found
	}
	allowPrivate := hostPortAllowed(dialSchemeFrom(ctx), host, port, policy.AllowedOrigins)
	for _, ip := range ips {
		if err := rejectIP(ip, policy.AllowLoopback, allowPrivate); err != nil {
			return nil, err
		}
	}
	return dialContextFn(ctx, network, net.JoinHostPort(ips[0].String(), port))
}

func isMetadataHost(host string) bool {
	return host == "metadata.google.internal" || host == "metadata.internal"
}

// validateOrigins rejects an allowed_artifact_origins entry that is not an
// exact scheme://host[:port] with no user, path, query or fragment.
func validateOrigins(list []string) error {
	for _, raw := range list {
		u, err := url.Parse(raw)
		if err != nil || !exactOrigin(raw) || isMetadataHost(strings.ToLower(u.Hostname())) {
			return errors.New("updater: allowed_artifact_origins entry must be scheme://host[:port]")
		}
	}
	return nil
}

// exactOrigin reports whether raw is scheme://host[:port] with no user, path,
// query or fragment. url.Parse leaves RawQuery empty when ForceQuery is set
// by a trailing "?", and it leaves Fragment empty for a bare trailing "#",
// so both are rejected from the raw string as well as from the parsed URL.
func exactOrigin(raw string) bool {
	if strings.ContainsAny(raw, "?#") {
		return false
	}
	u, err := url.Parse(raw)
	if err != nil || u == nil || u.ForceQuery || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Path != "" {
		return false
	}
	return u.Scheme == "https" || u.Scheme == "http"
}

// effectivePort is the port of a URL or allowlist entry. A missing port is
// the scheme default: 443 for https and 80 for http.
func effectivePort(scheme, port string) string {
	if port != "" {
		return port
	}
	switch strings.ToLower(scheme) {
	case "https":
		return "443"
	case "http":
		return "80"
	default:
		return ""
	}
}

// originsMatch reports whether one entry matches scheme, host and effective
// port. Comparison is case-insensitive. An empty scheme does not match.
func originsMatch(scheme, host, port string, origins []string) bool {
	scheme = strings.ToLower(scheme)
	host = strings.ToLower(host)
	port = effectivePort(scheme, port)
	if scheme == "" || host == "" || port == "" {
		return false
	}
	for _, raw := range origins {
		u, err := url.Parse(strings.TrimSpace(raw))
		if err != nil || u.Host == "" {
			continue
		}
		entryScheme := strings.ToLower(u.Scheme)
		if entryScheme != scheme || strings.ToLower(u.Hostname()) != host {
			continue
		}
		if effectivePort(entryScheme, u.Port()) == port {
			return true
		}
	}
	return false
}

// originAllowed reports whether one entry matches u. An entry matches when
// the scheme, host and effective port are equal. The effective port of a URL
// or entry without an explicit port is the scheme default: 443 for https and
// 80 for http.
func originAllowed(u *url.URL, origins []string) bool {
	if u == nil {
		return false
	}
	return originsMatch(u.Scheme, u.Hostname(), u.Port(), origins)
}

// hostPortAllowed applies the same scheme, host and effective-port rule to a
// dial address. port is the port DialContext was given. scheme comes from the
// request; an empty scheme does not match.
func hostPortAllowed(scheme, host, port string, origins []string) bool {
	return originsMatch(scheme, host, port, origins)
}

type dialSchemeKey struct{}

func withDialScheme(ctx context.Context, scheme string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, dialSchemeKey{}, scheme)
}

func dialSchemeFrom(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	scheme, _ := ctx.Value(dialSchemeKey{}).(string)
	return scheme
}

func isLoopbackName(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
