// Package enforce renders and checks the node data-plane policy.
//
// The agent (a later wave) feeds a typed model. This package turns it into one
// atomic nftables transaction for table inet fvpn and a tc batch that updates
// HTB classes in place. It does not flush the host ruleset, write fwmark, or
// install a flowtable. Verification reads nft -j and tc -j and returns a
// stable reason when the kernel state does not match the model.
//
// Metering is forward_admission_v1 (§10.1, §10.2): one named counter per
// device per direction, incremented only after the lease and isolation checks,
// on the inner IP packet, before SNAT. DNS-to-resolver and diagnostic drops
// use other counter names and are not usage.
//
// DetectResets compares counter values only: a backwards or missing counter
// is a meter epoch, and a counter that stays at zero is idle. DetectMeterResets
// is that comparison plus the ruleset. Use it when the caller holds a State,
// so a table, forward chain, or generation cleared outside the agent is still
// one epoch when the numbers did not move.
//
// A synthetic probe (§13.9, D-063) is an optional Model.Probe. Render and
// RenderBaseline admit only that peer's /32, for the listed destination
// sockets and, when ResolverDNS is set, TCP/UDP port 53 to the model's
// resolvers. The upload tuple sits behind the invalid, IPv6, deny, and host
// drops and before any member accept. Resolver DNS is the same early hole as
// member DNS, because the resolver prefix is inside the deny set, and a drop
// of every other port to that set follows it. The download accept is
// established,related back to the probe /32. Both sets carry the kernel
// timeout from ExpiresAt, at most ProbeMaxSeconds. Apply and
// ApplyBaselineModel remove the exception when Probe is nil; there is no
// separate withdraw call. ApplyBaseline is that path with a nil probe, which
// is what the agent and the watchdog call, including after a failed serving
// apply. probe_up and probe_down are diagnostic_v1 and are not member usage.
// The caller authenticates the test peer before it puts the address in
// SourceV4.
package enforce
