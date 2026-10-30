# Dev-stack owner requests — follow-up 9

Current evidence: [verification](followup9-verification.md),
[forwarding trace](followup9-forwarding-trace.md), and the complete dated records.

## Native route selection — resolved by w3-devstack-e2e

Evidence before selection: `Network: 0.0.0.0/0` / `Status: Not Selected`.
The diagnostic origin route used `eth0`; the request timed out. Explicitly
selecting the API-published `fvpn-exit-...` identifier installs `dev fvnb0`.
The fresh runs require HTTPS 200, device admission-byte growth, tunnel captures
and origin arrival from the node SNAT address. The original first stopped hop
belongs to the lab's native client configuration. It is not an agent lease or
nftables drop finding.

**Request to harness owners:** use the explicit native exit selection documented
in [RUN.md](RUN.md#native-exit-selection-and-traffic-trace), then require a counted
flow. Remove the superseded empty-member lease blocker where the new evidence
applies. `netbird networks select all` does not select this exit on 0.82.0.

## Resolver response corruption — resolved by w3-devstack-e2e

The loopback mTLS fault hook records HTTP 200 with changed bytes and
`fault_injected=true`. The node projects `failed/HASH_MISMATCH`; last-good
resolver rules and the synthetic block remain, and the same document reaches
`applied` after disarming. The worker and policy both use a TEST ONLY 300-second
cadence. No projector row, signed reference or agent cache is edited.

**Request to harness owners:** reuse the local transport hook for negative
resolver tests; central wrong-pin rejection is a different assertion.

## Recent admin authentication — resolved by w3-devstack-e2e

A completed diagnostic run reached `revoke: 403 FORBIDDEN` after the long
control-loss test passed the configured 900-second recent-auth window. The
harness now performs a fresh admin OIDC login before revoke. It does not extend
that window, forge auth time, or bypass the product's authorization checks.

## Measurement limits

The product SNAT rule has no counter expression. This run retains that rule and
proves translation using uplink packet headers, conntrack's reply tuple and the
authenticated origin source. No counter was invented or rule added. Physical
arm64 execution, full 20 × 3 native load, and the full AT29 crash/clock/grace
campaign remain operator/acceptance-harness work. The `lease-expiry` lab record
is limited to its measured control-loss withdrawal; it does not claim the full
AT29 campaign or that a 24-hour node lease elapsed.
