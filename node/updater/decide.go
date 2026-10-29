package updater

import (
	"strconv"
	"strings"
	"unicode"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/google/uuid"
)

// RollbackPlan is the §13.9 decision before the current pointer moves back.
type RollbackPlan struct {
	Restore bool
	Code    apierr.Code
}

// RollbackInput is the evidence DecideRollback is allowed to use. Version
// strings are not compared; the serving allowlist is (§13.2, UPD04).
type RollbackInput struct {
	HasLastGood    bool
	TreeOK         bool
	SignatureOK    bool
	InServingSet   bool
	Revoked        bool
	SchemaReadable bool
}

// DecideRollback chooses restore or recovery_required (§13.9 steps 2-4).
func DecideRollback(in RollbackInput) RollbackPlan {
	switch {
	case !in.HasLastGood || !in.TreeOK:
		return RollbackPlan{Code: apierr.RollbackFailed}
	case !in.SignatureOK:
		return RollbackPlan{Code: apierr.SignatureInvalid}
	case in.Revoked:
		return RollbackPlan{Code: apierr.ReleaseNotApproved}
	case !in.InServingSet:
		return RollbackPlan{Code: apierr.MinimumVersionViolation}
	case !in.SchemaReadable:
		return RollbackPlan{Code: apierr.SchemaIncompatible}
	default:
		return RollbackPlan{Restore: true}
	}
}

// versionAtLeast reports whether have is at least minimum. Both are
// dotted component versions; comparison is numeric per component, not
// lexical (§13.4 minimum_updater_version).
func versionAtLeast(have, minimum string) bool {
	if have == "" || minimum == "" {
		return false
	}
	h, m := splitVer(have), splitVer(minimum)
	for i := 0; i < 3; i++ {
		if h[i] > m[i] {
			return true
		}
		if h[i] < m[i] {
			return false
		}
	}
	return true
}

func splitVer(s string) [3]int {
	var out [3]int
	parts := strings.SplitN(s, ".", 3)
	for i := 0; i < len(parts) && i < 3; i++ {
		out[i] = leadingInt(parts[i])
	}
	return out
}

func leadingInt(s string) int {
	i := 0
	for i < len(s) && unicode.IsDigit(rune(s[i])) {
		i++
	}
	if i == 0 {
		return 0
	}
	n, err := strconv.Atoi(s[:i])
	if err != nil {
		return 0
	}
	return n
}

func containsID(ids []uuid.UUID, id uuid.UUID) bool {
	for _, v := range ids {
		if v == id {
			return true
		}
	}
	return false
}

func healthPassed(h api.UpdateHealth) bool {
	return h.Processes == api.HealthCheckStatusPassed &&
		h.IdentityAndState == api.HealthCheckStatusPassed &&
		h.InternetOnly == api.HealthCheckStatusPassed &&
		h.DnsPolicy == api.HealthCheckStatusPassed &&
		h.TunnelProbe == api.HealthCheckStatusPassed
}

func selectArtifact(m api.ReleaseManifest, osID, osVer, arch string) (api.ManifestArtifact, error) {
	for _, a := range m.Artifacts {
		if string(a.OsId) != osID || string(a.Architecture) != arch {
			continue
		}
		for _, v := range a.OsVersions {
			if v == osVer {
				return a, nil
			}
		}
	}
	return api.ManifestArtifact{}, permanent(apierr.UnsupportedPlatform, osID+" "+osVer+" "+arch)
}
