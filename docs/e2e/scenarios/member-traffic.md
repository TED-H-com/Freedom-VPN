# Native member forwarding — D-052/D-057

Use one public-subject issuer for Portal and native NetBird login. Create the
member's plan and exit group through the product APIs, grant or pay the member
through the product paths, then admit the real native device. Poll the
production identity mapping, device authorization and binding projections.
Match the plan's group to the published exit's `group_ids`.

Read the desired-state device binding before attempting a flow. Authorized
device projection alone does not establish forwarding; neither does an `fvnb0`
accept rule. The serving policy must contain `fvmb0` member accepts and a real
lab-local flow must traverse the exit. Lease expiry requires a successful
positive flow before control loss, then measured withdrawal after the lease
expires.

See [member traffic](../runs/2026-10-30/member-traffic.md),
[forwarding](../runs/2026-10-30/forwarding.md) and
[lease expiry](../runs/2026-10-30/lease-expiry.md). The D-057 group/publication
prerequisite now passes. The current lease-refresh blocker is described below.
Resolver enforcement separately requires an issued D-056 document reference;
a central feed document is not an on-node ACK.

## Follow-up 7

The native peer and portal use the same public subject. The shared native helper
checks the exact peer identity and polls the authorized device; the desired
state now contains its real device binding and non-empty member rate policy.
The plan's TEST ONLY 100 Mbps rate is explicit. NetBird uses `fvmb0` on the exit
with `--disable-firewall`; the product agent owns `inet fvpn`. The real
origin-facing interface is renamed `fvup0` inside the node and the configured
SNAT is that interface's address. The member has only the lab bridge and cannot
reach the controlled origin through a directly attached origin bridge.

Traffic passes only after HTTPS to the controlled origin returns 200 and the
same device's admission counters increase. The persistent boot recovery config
is created from real enrollment IDs; it is separate from the one-shot updater
trigger. Missing/failed recovery still blocks forwarding.

The current remaining blocker is `node/agent` lease renewal: the agent retains
its initial lease (generation 1, zero members, 24-hour deadline) after publication
and member binding advance desired state. The bounded probe sees `serving 0`,
`policy_expires_mono_ns 0`, `reason lease_expired`, and no admission counters.
`needRenew()` does not consider the changed desired/member generation.
**Owner: w2-agent-core.** The runner records the exact stored lease, desired
counts, current node HTTP response/checks and heartbeat, and does not restart
the process to force renewal. No counted flow, usage or expiry success is claimed.
