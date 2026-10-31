// Package installerjob derives the bounded installer receipt identity (D-110).
package installerjob

import (
	"strings"

	"github.com/google/uuid"
)

// ID returns one installer job ID per release manifest and node incarnation.
// The caller must validate the release ID, SHA-256 hex digest and incarnation.
// Neither a staging path nor a local request ID participates in this identity.
func ID(release uuid.UUID, manifestSHA256 string, incarnation uuid.UUID) uuid.UUID {
	digest := strings.ToLower(manifestSHA256)
	return uuid.NewSHA1(claimNamespace(release, digest), []byte(digest+"|"+incarnation.String()))
}

// Keep the claim namespace used by node/updater's installerClaim. D-110 uses
// the verified manifest digest for the final UUID name, replacing the local
// pending-record digest that central never receives.
func claimNamespace(release uuid.UUID, manifestSHA256 string) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte("fvpn-installer-apply|"+release.String()+"|"+strings.ToLower(manifestSHA256)))
}
