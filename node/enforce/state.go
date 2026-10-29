package enforce

// State is the part of table inet fvpn that a render must not recreate.
// Named counters are the important part: flushing them would look like a
// meter reset (§10.7).
type State struct {
	Table    bool
	Chains   map[string]bool
	Sets     map[string]bool
	Counters map[string]CounterValue
}

func (s State) hasChain(name string) bool { return s.Chains[name] }
func (s State) hasSet(name string) bool   { return s.Sets[name] }
func (s State) hasCounter(name string) bool {
	_, ok := s.Counters[name]
	return ok
}
