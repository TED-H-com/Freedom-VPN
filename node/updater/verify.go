package updater

import (
	"bytes"
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
// a loopback address for tests. Link-local and metadata hosts are refused
// (§16.2). The signature is what authorizes the URL; this stops a manifest
// from pointing at the cloud metadata service.
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
		if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
			return permanent(apierr.SignatureInvalid, "artifact url address")
		}
		if ip.IsLoopback() {
			if allowLoopback {
				return nil
			}
			return permanent(apierr.SignatureInvalid, "artifact url loopback")
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

func isLoopbackName(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
