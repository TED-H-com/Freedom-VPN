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

// checkURL allows https URLs and, only when allowLoopback is set, http to
// a loopback address for tests. Loopback, link-local, multicast, unspecified,
// private, CGNAT (100.64.0.0/10) and metadata hosts are refused (§16.2,
// D-034f). The same address check runs again on every address DialContext
// resolves, so a public name that answers with a private address is refused.
func checkURL(raw string, allowLoopback bool) error {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.User != nil {
		return permanent(apierr.SignatureInvalid, "artifact url")
	}
	host := strings.ToLower(u.Hostname())
	if host == "metadata.google.internal" || host == "metadata.internal" {
		return permanent(apierr.SignatureInvalid, "artifact url host")
	}
	if ip := net.ParseIP(host); ip != nil {
		if err := rejectIP(ip, allowLoopback); err != nil {
			return err
		}
	}
	if u.Scheme != "https" {
		if allowLoopback && u.Scheme == "http" && isLoopbackName(host) {
			return nil
		}
		return permanent(apierr.SignatureInvalid, "artifact url scheme")
	}
	return nil
}

func rejectIP(ip net.IP, allowLoopback bool) error {
	if ip == nil {
		return permanent(apierr.SignatureInvalid, "artifact url address")
	}
	if ip.IsLoopback() {
		if allowLoopback {
			return nil
		}
		return permanent(apierr.SignatureInvalid, "artifact url loopback")
	}
	if ip.IsPrivate() || isCGNAT(ip) || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
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
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, permanent(apierr.SignatureInvalid, "artifact url address")
	}
	var ips []net.IP
	if ip := net.ParseIP(host); ip != nil {
		ips = []net.IP{ip}
	} else {
		found, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, err
		}
		if len(found) == 0 {
			return nil, permanent(apierr.SignatureInvalid, "artifact url address")
		}
		for _, one := range found {
			ips = append(ips, one.IP)
		}
	}
	for _, ip := range ips {
		if err := rejectIP(ip, allowLoopback); err != nil {
			return nil, err
		}
	}
	d := &net.Dialer{Timeout: 30 * time.Second}
	return d.DialContext(ctx, network, net.JoinHostPort(ips[0].String(), port))
}

func isLoopbackName(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
