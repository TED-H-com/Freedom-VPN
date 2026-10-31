package e2e

import "github.com/google/uuid"

// ReleaseOptions controls rollback compatibility, independently of the fleet's
// serving support policy. Empty means no additional restriction (D-034e).
type ReleaseOptions struct {
	AllowedReleaseIDs []uuid.UUID
	// InstallerLayout packages root binaries and the unmodified systemd units.
	InstallerLayout bool
}

// RollbackAllowlist includes last-good and removes duplicates. The returned
// slice owns its storage; callers can reuse their input for other candidates.
func RollbackAllowlist(lastGood uuid.UUID, allowed ...uuid.UUID) []uuid.UUID {
	out := []uuid.UUID{}
	seen := map[uuid.UUID]bool{}
	for _, id := range append([]uuid.UUID{lastGood}, allowed...) {
		if id != uuid.Nil && !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}
