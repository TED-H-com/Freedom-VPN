package enforce

import "github.com/google/uuid"

// State is the part of table inet fvpn that a render must not recreate.
// Named counters are the important part: flushing them would look like a
// meter reset (§10.7).
type State struct {
	Table    bool
	Chains   map[string]bool
	Sets     map[string]bool
	Counters map[string]CounterValue
	// Classes are member minors read back from per-member fvpn_class comments
	// so a later render does not move a member (§11.4). nft comments are
	// at most 128 bytes, so each member has its own marker rule.
	Classes map[uuid.UUID]uint16
}

func (s State) hasChain(name string) bool { return s.Chains[name] }
func (s State) hasSet(name string) bool   { return s.Sets[name] }
func (s State) hasCounter(name string) bool {
	_, ok := s.Counters[name]
	return ok
}
