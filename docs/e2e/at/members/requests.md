# Follow-up 3 requests to orchestrator

Assigned stack: `w4-members / 24000`. Both attempts are retained under
[history/follow-up-3](history/follow-up-3/attempt-1/attempt.json).
D-050 was respected: no identity, entitlement, subscription, device, binding,
observation or publication projection was seeded or repaired.

## F3-01 — select the native exit in the shared forwarding driver

Owner: **devstack**, `tests/e2e/traffic_scenario_test.go:countedFlow` and its
member-traffic caller. This is a shared harness request, not an agent defect.

[AT01](AT01.json) proves generation-5 lease/applied equality, six member accept
rules, a selected `fvnb0`/`netbird` route, origin HTTPS 200 and device upload /
download increments of 2214 / 2411 bytes. It repeats after pause/unpause.
[AT02](AT02.json) also records counted A and race-winner transfers. See the
[hop observation](forwarding-observation.md).

The shared driver invokes `wget` without native exit selection. With
`skip_auto_apply=true`, the member's controlled-origin route initially uses
`eth0`. Poll `netbird routes list` for the advertised exit, select its network
ID through `netbird routes select`, then prove the kernel route uses the tunnel
before counting the controlled request. Keep `skip_auto_apply=true`; do not add
host routes, grant an unbound peer, or change node enforcement. The first
attempt's unsuccessful immediate selection is retained, not relabeled passed.

## F3-02 — missing conntrack breaks enforcement after device revocation

Owner: **devstack**, `deploy/devstack/Dockerfile.node` dependency installation.
Related product path: **w2-agent-core / enforcement**,
`node/agent/engine.go:enforce` → `node/enforce/apply.go:Apply` → `deleteConntrack`.
Blocks the complete post-revoke AT02 result and subsequent AT03/AT04/AT08 traffic.

Reproduce on a fresh stack with `make -f mk/at-members.mk at-members`:

1. AT01 and the AT02 final-slot winner have origin responses and increasing
   device counters. AT02 rejects the other contender and protects B's device.
2. Revoke A2 through the member DELETE API and admit the former loser through
   the production adapter. Counts change 3 → 2 → 3 at `2026-10-30T07:09:36Z`.
3. The first immutable observation with `HEALTH_CHECK_FAILED`, stage `apply`,
   message `apply failed` arrives at `07:09:39.760469Z`; failures repeat.
4. `docker exec fvpn-w4-members-node sh -c 'command -v conntrack'` finds no
   executable. `deleteConntrack` invokes `conntrack -D -s/-d` for dropped addresses;
   the image installs only ca-certificates, iproute2, nftables and wget.
5. AT03's paid member selects the exit successfully. Its five SYN packets reach
   node `fvmb0`, but the forward chain is generation 0 with zero accept rules.
   `ctbase` increases 0 → 300 bytes; device counters stay zero, no uplink packets
   or origin response are observed. Lease generation remains current: this is
   not the old F2-01 renewal defect.

Include the required conntrack utility in the shared node image and rerun the
same revoke/replace sequence without restarting the agent or clearing state.
Do not work around withdrawal by skipping `deleteConntrack`, rewriting device
rows, or installing a package ad hoc in the acceptance container. This task did
none of those. The evidence bounds the cause to the missing executable and the
observed apply failure; the hidden command error was not instrumented.

Agent diagnostics follow-up: `noteApply` reports the existing apply error, but
the same observations have all named checks `passed` and an empty
`forwarding_block_reasons` while `member_forwarding_enabled=false`. Keep public
failure reporting coherent using the existing contract vocabulary. No new
HTTP error or reason token is proposed here.

## F3-03 — remaining controlled lab coverage

Owner: **devstack**, documented exports in `tests/e2e/stack_exports.go`,
`tests/e2e/native_exports.go`, `deploy/devstack/compose.yaml`,
`deploy/devstack/entrypoint-member.sh`, `deploy/devstack/coredns/`, and
`tests/e2e/cmd/fvpn-e2e-origin/main.go`.

- AT01: a documented second member network is needed for a Linux network move.
  Pause/unpause is exercised. Four native platforms remain operator runs.
- AT04: the existing origin only serves HTTPS. Supply bounded persistent UDP
  and TCP flow hooks and a second runnable exit before all-node stop timing can
  pass. The TCP refund monitor was attempted but had no positive baseline after
  F3-02; its refusal to claim stop latency is recorded.
- AT05: supply preinstalled/cloud-init/operator-installer boot flavours. The
  current stack executes one installer runtime; token reservations and arm64
  cross-compilation do not prove the other sources or physical arm64.
- AT09: one runnable node was observed. `Stack.Enroll` targets `node` only;
  README's `SEEDED_NODE_2_ID` is a synthetic fixture, forbidden by D-050.
  Supply a second runnable exit hook, direct/forced-relay and UDP/QUIC/PMTU
  controlled endpoints. Native selection of the one exit is already proven.
- AT20/21: the member entrypoint uses `--disable-dns`; one AdGuard filter plus
  an authoritative CoreDNS is not two independent filtering resolvers. Supply
  normal native DNS configuration, a second filter, policy-sync fault hook and
  positive CNAME/AAAA/HTTPS/SVCB/rebinding fixtures. Node-local D-056 delivery
  does not certify that the member uses that resolver.
- AT22: extend file/raw-usage/rollup and backup-expiry fixtures. Current scans
  cover DNS sentinels, resolver settings, service logs, metrics, audit and an
  in-memory dump, plus audit purge and scheduled feed last-good. They do not
  certify every retention surface or a representative URL history.

## Closed or superseded findings

- **F2-01**: closed by #146 and live current-generation lease evidence under
  uninterrupted operation. It is no longer an unconditional harness blocker.
- **F2-05**: #143 durable restart/stable `node-netns` path exercised with the
  same incarnation, fresh accepted observation, readable nft table and no
  installer token. No agent-only service workaround or resolver restart.
- **B4**: all six created/paid/failed orders now use authenticated provider hooks,
  independently identified members, exact HMAC bytes and two-second pacing.
  Ten replays, invalid signature, forged return, cancellation, partial/full
  refund, real expiry and suspension precedence are recorded.
- **B6/B7 / F2-03**: typed applied resolver evidence and exact node-mTLS
  canonical digest match are live. An initial stale profile observation is
  preserved; supplemental convergence is recorded separately.
- **B9**: `attribution:null` is accepted, activated and referenced by a DNS
  profile. Empty and whitespace-only attribution receive 422. The historical
  claim is retired; [AT22.json](AT22.json) contains the exact checks.

The first attempt's payment collision was a **harness defect**, not a provider
or product finding: a long fixture name lost its order suffix at the shared
40-character key boundary. Run UUID and short fixture key are now separate,
and each order asserts a unique member. Its interrupted evidence is retained.

Publication polling initially stopped on a retryable failed step while the
operation was still running. The final harness waits for terminal operation
state; the readback records later step results without erasing earlier
checks. AT08's post-restart 422 `NODE_NOT_SERVICEABLE` remains in `prior_checks`;
no failed prior result is upgraded to a complete pass.

[OPERATORS.md](OPERATORS.md) retains **needs operator run** for the four real
platforms, physical arm64, second physical NAT and candidate hardware.
