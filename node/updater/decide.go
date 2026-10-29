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

// versionAtLeast reports whether have is at least minimum under SemVer 2.0.0
// precedence (spec section 11). Build metadata is ignored. A pre-release is
// lower than the same core without one, so 1.0.0-alpha.1 < 1.0.0. An invalid
// version fails closed.
func versionAtLeast(have, minimum string) bool {
	if have == "" || minimum == "" {
		return false
	}
	h, okH := parseSemver(have)
	m, okM := parseSemver(minimum)
	if !okH || !okM {
		return false
	}
	return compareSemver(h, m) >= 0
}

type semver struct {
	major int
	minor int
	patch int
	pre   []string
}

func parseSemver(s string) (semver, bool) {
	if i := strings.IndexByte(s, '+'); i >= 0 {
		s = s[:i]
	}
	var pre string
	if i := strings.IndexByte(s, '-'); i >= 0 {
		pre = s[i+1:]
		s = s[:i]
		if pre == "" {
			return semver{}, false
		}
	}
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return semver{}, false
	}
	nums := [3]int{}
	for i, p := range parts {
		n, ok := semverCore(p)
		if !ok {
			return semver{}, false
		}
		nums[i] = n
	}
	var idents []string
	if pre != "" {
		idents = strings.Split(pre, ".")
		for _, id := range idents {
			if id == "" || !semverIdent(id) {
				return semver{}, false
			}
		}
	}
	return semver{major: nums[0], minor: nums[1], patch: nums[2], pre: idents}, true
}

func semverCore(s string) (int, bool) {
	if s == "" || (len(s) > 1 && s[0] == '0') {
		return 0, false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, false
		}
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, false
	}
	return n, true
}

func semverIdent(s string) bool {
	for _, c := range s {
		if c != '-' && !unicode.IsDigit(c) && !unicode.IsLetter(c) {
			return false
		}
	}
	if numericIdent(s) && len(s) > 1 && s[0] == '0' {
		return false
	}
	return true
}

func numericIdent(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func compareSemver(a, b semver) int {
	if c := cmpInt(a.major, b.major); c != 0 {
		return c
	}
	if c := cmpInt(a.minor, b.minor); c != 0 {
		return c
	}
	if c := cmpInt(a.patch, b.patch); c != 0 {
		return c
	}
	if len(a.pre) == 0 && len(b.pre) == 0 {
		return 0
	}
	if len(a.pre) == 0 {
		return 1
	}
	if len(b.pre) == 0 {
		return -1
	}
	n := len(a.pre)
	if len(b.pre) < n {
		n = len(b.pre)
	}
	for i := 0; i < n; i++ {
		if c := cmpIdent(a.pre[i], b.pre[i]); c != 0 {
			return c
		}
	}
	return cmpInt(len(a.pre), len(b.pre))
}

func cmpInt(a, b int) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

func cmpIdent(a, b string) int {
	an, aNum := identNum(a)
	bn, bNum := identNum(b)
	switch {
	case aNum && bNum:
		return cmpInt(an, bn)
	case aNum && !bNum:
		return -1
	case !aNum && bNum:
		return 1
	default:
		if a < b {
			return -1
		}
		if a > b {
			return 1
		}
		return 0
	}
}

func identNum(s string) (int, bool) {
	if !numericIdent(s) || (len(s) > 1 && s[0] == '0') {
		return 0, false
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, false
	}
	return n, true
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
