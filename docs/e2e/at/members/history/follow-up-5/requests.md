# Follow-up 5 requests to orchestrator

Use the [current AT table](summary.md), [run ledger](RUN.md) and per-AT JSON
for the observed boundary. No application, contract, migration or shared lab
source was changed. No D-050 projection was seeded or repaired.

## F4-01 — empty member lease after second-exit restart (Resolved by #193 / D-067)

Owner: **w2-agent-core**, coordinated with **w3-wire-nodes** for lease issuance.
Locations: `node/agent/sync.go:needRenew`, `node/agent/decide.go:Decide`,
`internal/leases/leases.go:RenewTx` (`includeMembers`). AT: AT09.
Status: **Resolved** by PR #193 (`705170d`, D-067 provisional grant-less leases) and verified on fresh `w4-members / 24000` run `16aa2d1b-d46f-4e88-a509-a8a9c9076006`.

Reproduce on fresh `w4-members / 24000`: run the complete suite through paid
member admission and both published exits; select `node2` (`01a18e46-3df2-7053-a540-8422dca18746`)
and prove its counted transfer (`device_upload_bytes` `0 -> 2305`, `device_download_bytes` `0 -> 2411`
on desired/applied generation `25` with 7 member grants, `issued_at = 2026-10-30T19:36:11.464981+00:00`).
Stop `node2`, `node2-adguard` and `node2-netns` (`stopped_at = 2026-10-30T19:36:33.597391519Z`),
verify zero origin receipts while offline, then start the holder, resolver and
node in that order (`resumed_at = 2026-10-30T19:36:50.392336111Z`) with no policy
change or admin action (`desired_generation` remains `"25"`).

Under D-067 (#193), a serving lease with zero member grants while the desired
generation has member bindings is treated as provisional (`leaseAwaitingMemberGrants`):
`needRenew` renews at the next heartbeat and then with exponential backoff
(`1x`, `2x`, `4x` heartbeat, capped at 60 s) until a lease with grants arrives.

Measured recovery on unchanged generation `25` ([AT09.json](AT09.json)):
- **Lease sequence (`serving_leases` for `node2` on generation `25`):**
  1. `2026-10-30T19:36:11.464981+00:00` — `generation: "25"`, `member_grants: 7` (pre-stop)
  2. `2026-10-30T19:36:52.908587+00:00` — `generation: "25"`, `member_grants: 0` (initial post-start lease while not yet `ready`)
  3. `2026-10-30T19:36:58.749086+00:00` — `generation: "25"`, `member_grants: 0` (D-067 provisional renewal #1 at ~5.8 s / 1x heartbeat)
  4. `2026-10-30T19:37:08.853644+00:00` — `generation: "25"`, `member_grants: 0` (D-067 provisional renewal #2 at ~10.1 s / 2x backoff)
  5. `2026-10-30T19:37:29.064693+00:00` — `generation: "25"`, `member_grants: 7` (D-067 provisional renewal #3 at ~20.2 s / 4x backoff after readiness)
- **Admin API & agent observation sequence on generation `25`:**
  - `2026-10-30T19:36:53.707107Z`: `GET /api/v1/admin/nodes/{node2}` (`provisional_admin_node`) and `node_observations` report `member_forwarding_observed: false`, `block_reasons: ["lease_expired", "safety_check_failed"]`, `lease_check: {"status": "failed", "error_code": "LEASE_EXPIRED", "detail": "Serving lease has no member grants yet."}`, `safety_status: "failed"` (as noted in PR #193, the wire reason remains `lease_expired` until contract revision 10 introduces `awaiting_member_grants`).
  - `2026-10-30T19:37:24.045812Z`: `node_observations` reports `safety_status: "passed"`, `forwarding: false`, `reasons: ["lease_expired"]`, `lease_check: {"status": "failed", "error_code": "LEASE_EXPIRED", "detail": "Serving lease has no member grants yet."}`, transitioning `node2` to `ready`.
  - `2026-10-30T19:37:29.131033Z`: `GET /api/v1/admin/nodes/{node2}` (`recovered_admin_node`) and `node_observations` report `desired_generation: "25"`, `applied_generation: "25"`, `provisioning_state: "ready"`, `publication_intent: "published"`, `publication_observed: "active"`, `member_forwarding_observed: true`, `block_reasons: []`, `lease_check: {"status": "passed", "error_code": null, "detail": null}`.
- **Measured recovery time from container start (`2026-10-30T19:36:50.392336111Z`):**
  - Time to serving-ready (`serving_ready_at = 2026-10-30T19:37:29.628457149Z`, accepted observation + 18 kernel `member_accept_rules`): **`39.236 s`**.
  - Time to first counted member byte (`first_counted_byte_at = 2026-10-30T19:37:34.374304456Z`): **`43.982 s`**.
  - Member TCP attempts through `fvnb0` (`10 to_origin / 7 from_origin`), `node2` `fvmb0` (`10 to_origin / 7 from_origin`), `fvup0` (`10 to_origin / 7 from_origin`), and `origin eth0` (`10 to_origin / 7 from_origin`) succeed with `http_status: 200`, `origin_receipt: true`, `origin_snat_matches: true`, and fresh kernel counter increments (`device_upload_bytes: "0" -> "2305"`, `device_download_bytes: "0" -> "2411"`).

## F4-03 — remaining transport/platform matrix

Owner: **w3-devstack-e2e and the platform/operator campaign**.
Locations: `tests/e2e/second_exit.go:StartSecondExit`,
`tests/e2e/cmd/fvpn-e2e-origin/probe_udp.go`, and `docs/e2e/RUN.md`.
ATs: AT01, AT04, AT09.

Reproduce: prepare the signed baseline, enroll/publish both actual exits, admit
paid native peers and run the owned harness. HTTPS traverses both exits with
selected-node SNAT and device counter evidence. The available UDP/4443 hook is
an authenticated updater-only exception probe; it is not a member echo/QUIC
fixture and must not be counted as member traffic. No controlled forced-relay,
QUIC or PMTU matrix is exported.

Request: add a bounded member TCP/UDP/QUIC/PMTU fixture and a documented
forced-relay transport fault through the shared lab. Keep the updater probe
separate. Run the exact four-platform/network-change and physical second-NAT
scripts in [OPERATORS.md](OPERATORS.md). Persistent TCP refund, expiry and
suspension observations are retained independently; they cannot pass the UDP
or physical-network assertions.

## F4-04 — member UDP withdrawal on every exit

Owner: **w3-devstack-e2e**. Location:
`tests/e2e/cmd/fvpn-e2e-origin/probe_udp.go`; AT04.
The existing hook serves the updater exception only. Reproduce with the real
paid cohorts and refund/expiry/suspension events in AT04: persistent TCP can be
measured, but there is no sanctioned member UDP echo peer. Add the bounded member
fixture requested in F4-03, then run existing and new UDP flows through both
published exits. Do not reuse or count the non-member updater probe. Extend `deploy/devstack/probe-portal.mjs` with rendered suspended/expired state readback, coordinated with the portal owner; the current probe establishes the billing return only.

## F4-05 — native DNS configuration and automatic controlled fallback

Owner: **w3-devstack-e2e**, then the native-client integration owner if required.
Locations: `deploy/devstack/entrypoint-member.sh`,
`tests/e2e/native_exports.go:JoinNativeMember`, `tests/e2e/second_exit.go`.
ATs: AT20, AT21.

Reproduce: read only `DisableDNS` from the running member's
`/var/lib/netbird/default.json`; it is true. Pause each of `adguard` and
`node2-adguard` in turn: its loopback DNS endpoint becomes unavailable while
the other continues to apply the synthetic block. Typed node evidence and the
node-mTLS document hash establish resolver delivery separately.

Request: export a native member DNS configuration path using the product's
issued resolver document and both authorized filtering resolvers. Then measure
automatic fallback, no public fallback and recovery. Explicit loopback queries
are not native-client failover. The synchronization corruption hook is tested
separately; Private DNS/DoH remain operator runs with member guidance.

## F4-06 — complete privacy and retention surfaces

Owner: **w3-devstack-e2e**, with usage/database/backup owners for retention inputs.
Locations: `tests/e2e/backup_scenario_test.go`, `docs/e2e/scenarios/backup.md`,
`tests/e2e/at/members/dns_privacy_test.go:privacy` and
`privacy_readback_test.go:secondPrivacy`.
AT: AT22.

Reproduce: run the synthetic DNS sentinel, disabled querylog/statistics
readback, log/metrics/audit/pg_dump scans and guarded audit purge. The extra
resolver has its own config/log inventory. The scheduled partial target
publishes a self-authored licensed feed, observes the real failed refresh and
last-good delivery. These checks do not age representative raw-usage/rollup,
file and backup artifacts through every configured retention deadline, or
cover a representative member URL history workload.

Request: provide sanctioned retention inputs for those remaining surfaces and
a bounded member URL privacy sample through shared hooks; do not backdate or
repair projector-owned rows. Record the selected commercial feed's actual
license separately: the TEST ONLY self-authored feed is not commercial-source
approval. Full AT22 stays blocked despite passed subsets.

## F4-07 — controlled DNS RR/cache/rebinding matrix

Owner: **w3-devstack-e2e**, with DNS/netlab owners reviewing fixture semantics.
Locations: `deploy/devstack/coredns/example.invalid.zone`,
`tests/e2e/at/members/dns_privacy_test.go:dnsRules`.
AT: AT20.

Reproduce: publish canonical rules through the API and inspect the node's typed
applied document, then issue the recorded A/AAAA/CNAME/HTTPS/SVCB queries. API
normalization rejection and resolver application are measured, but the shared
zone does not provide the complete positive/negative CNAME, HTTPS/SVCB, cache
expiry and rebinding sequence. Native DNS is additionally blocked by F4-05.

Request: extend the controlled zone and sanctioned rebinding fixture with exact
answers/TTLs, including label-boundary controls and an allowed domain that
changes to an isolated private target. Prove that DNS allowlisting never opens
that network. Never query unrelated Internet domains.

## F4-10 — source flavours and physical architecture

Owner: **packaging/devstack and the operator campaign**.
Locations: `deploy/devstack/entrypoint-node.sh`,
`tests/e2e/stack_exports.go:EnrollAt`, [OPERATORS.md](OPERATORS.md).
AT: AT05.

Two real nodes use the same installer entrypoint. Installer/preinstalled token
reservation and one-time replay contracts pass, but a preinstalled reservation
is not a preinstalled boot and a second installer is not cloud-init. Supply the
other two boot-source fixtures/provenance and run physical arm64. The signed
arm64 artifact and arm64 probe cross-build are not a hardware run.

## Closed prior findings and preserved harness findings

- F4-01: resolved by PR #193 (D-067 provisional grant-less leases). Cold restart
  of `node2` (`node2`, `node2-adguard`, `node2-netns`) restores the 7-grant lease
  and counted member TCP traffic through `fvmb0` on unchanged desired generation
  `25` (`39.236 s` to serving-ready, `43.982 s` to first counted member byte).
  Follow-up 4's pre-D-067 record remains archived under [history/follow-up-4](history/follow-up-4/AT09.json).
- F3-02: current images contain conntrack. AT02 requires a continuous agent PID,
  coherent serving health, revoked-source denial/empty conntrack and counted
  sibling/replacement traffic. The historical failure remains archived.
- F3-01: selection waits for the advertised route and explicitly selects the
  network matching the intended node's peer. `skip_auto_apply` remains true.
- D-056/D-061: typed resolver evidence, canonical mTLS document hashes and actual
  block answers replace the former missing-delivery diagnosis. Null feed
  attribution is valid; only blank/whitespace notes are rejected.

The strict post-revoke node-ingress measurement remains the **devstack** owner's
separate [recorded limitation](../../runs/2026-10-30/followup10-run3/device-revoke.md):
route withdrawal prevents that later packet from reaching the node. This harness
does not restore a revoked route or relabel upstream withdrawal as a kernel drop.
