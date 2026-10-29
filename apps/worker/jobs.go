package worker

// Domains is the only registration list. Add exactly one line per domain;
// constructors and wiring belong in jobs_<area>.go. See docs/control/COMPOSITION.md.
func Domains() []Domain {
	return []Domain{
		members(),
		dns(),
		policies(),
		usage(),
	}
}
