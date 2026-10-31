package agent

// recoveryDecision builds a local deny-all candidate, never an authorization.
// An old-generation or grant-less lease cannot open member forwarding. Clearing
// bindings and grants allows a full nft/tc Apply and read-back before the new
// generation ACK. The ordinary signed documents remain untouched, and a fresh
// matching lease plus another verified Apply are required to serve.
func recoveryDecision(in Input) (Decision, bool) {
	if in.Desired == nil || in.Lease == nil {
		return Decide(in), false
	}
	desired, lease := *in.Desired, *in.Lease
	desired.DeviceBindings = nil
	lease.Members = nil
	lease.ConfigGeneration = desired.ConfigGeneration
	in.Desired, in.Lease = &desired, &lease
	d := Decide(in)
	return d, d.Serve
}
