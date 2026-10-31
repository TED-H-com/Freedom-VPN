// Package enforce renders and checks the node data-plane policy.
//
// The agent (a later wave) feeds a typed model. This package turns it into one
// atomic nftables transaction for table inet fvpn and a tc batch that updates
// HTB classes in place. It does not flush the host ruleset, write fwmark, or
// install a flowtable, and it never writes any table other than inet fvpn.
// Verification reads `nft -j list ruleset` and tc -j and returns a stable
// reason when the kernel state does not match the model. A base chain outside
// inet fvpn is part of those bytes. Accept, continue, counter, and log are
// tolerated, because an accept does not stop a later base chain. A statement
// that can rewrite, mark, offload, queue, or drop member traffic on a hook
// that traffic crosses fails closed. A native match that restricts the
// packet to 127.0.0.0/8 or to ::1 is classed loopback_scoped and does not
// fail closed; an xt match does not prove that scope.
// VerifyServing also reads tc qdiscs and every filter attachment on both
// member-path interfaces. Foreign or unreadable tc state is TC_FOREIGN_FILTER;
// Apply records the evidence and restores the nft baseline. No tc filters are
// installed, and foreign clsact state is not removed by that baseline.
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
// RenderBaseline admit only that peer's /32, for at most ProbeMaxDestinations
// exact sockets that do not overlap a member binding, and, when ResolverDNS
// is set, TCP/UDP port 53 to the model's resolvers. The upload tuple sits
// behind the invalid, IPv6, deny, and host drops and before any member accept.
// Resolver DNS is the same early hole as member DNS, because the resolver
// prefix is inside the deny set, and a drop of every other port to that set
// follows it. The download accept is established,related back to the probe
// /32. Both sets carry the kernel timeout from ExpiresAt, at most
// ProbeMaxSeconds. TimeoutCeilSeconds and the remaining lifetime already in
// State cap a re-render so a live set is not given a fresh full window.
// Verify rejects a prefix wider than /32, an element it cannot parse, an
// original timeout above ProbeMaxSeconds, or a remaining lifetime past the
// model window, as ReasonProbeUnbounded.
// Apply and ApplyBaselineModel remove the exception when Probe is nil; there
// is no separate withdraw call. ApplyBaseline is that path with a nil probe,
// which is what the agent and the watchdog call, including after a failed
// serving apply. probe_up and probe_down are diagnostic_v1 and are not member
// usage. The caller authenticates the test peer before it puts the address
// in SourceV4.
package enforce
