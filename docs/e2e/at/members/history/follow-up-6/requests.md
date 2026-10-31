# Follow-up 6 requests to orchestrator

Use the [current AT table](summary.md), [run ledger](RUN.md) and per-AT JSON
for the observed boundary. No application, contract, migration or shared lab
source was changed. No D-050 projection was seeded or repaired.

## Closed in Follow-up 6: AT02 strict post-withdrawal node-ingress injection (Resolved by #219)

Owner: **w3-devstack-e2e** (delivered in PR #219, wired into `AT02` in Follow-up 6).
Locations: `tests/e2e/internal/labstack/ingress.go:ObserveMemberIngress`,
`tests/e2e/at/members/native_test.go:strictPostWithdrawalIngress`. AT: AT02.
Status: **Resolved** and verified on fresh `w4-members / 24000` run `80db103b-622c-42b1-946d-4628daaa213d` ([AT02.json](AT02.json), [AT02.md](AT02.md)).

After real member DELETE and `SyncDevice` convergence on generation `8`, `AT02` invokes `ObserveMemberIngress` on `fvmb0` (empty `-mechanism` -> `tun-mirred`, link type `65534`, source port `40000`, destination `192.0.4.10:9`):
- **Revoked device `member-a2` (`01a18efa-99e8-75d7-9a6c-47f05b5c55e9`, `100.104.160.29`):** `ctunku` moves `{packets: 0, bytes: 0} -> {packets: 1, bytes: 29}` (`+1` packet, `+29` bytes); `u_01a18efa99e875d79a6c47f05b5c55e9` stays at `{packets: 10, bytes: 2317}`; `d_01a18efa99e875d79a6c47f05b5c55e9` stays at `{packets: 7, bytes: 2411}`; `uplink_empty: true`, `origin_empty: true`, `forwarded_bytes: 0`, `other_device_counters_held: true`.
- **Admitted sibling `member` (`01a18ef9-ae93-7a1c-b934-20bed403bf2f`, `100.104.242.187`):** `u_01a18ef9ae937a1cb93420bed403bf2f` moves `{packets: 31, bytes: 6967} -> {packets: 32, bytes: 6996}` (`+1` packet, `+29` bytes); `d_01a18ef9ae937a1cb93420bed403bf2f` moves `{packets: 22, bytes: 7283} -> {packets: 23, bytes: 7340}` (`+1` packet, `+57` bytes on origin ICMP reply); `ctunku` stays at `{packets: 1, bytes: 29}` (`+0`); `uplink_saw_datagram: true` (`2` capture lines), `origin_saw_datagram: true` (`2` capture lines), `other_device_counters_held: true`.

## F4-01 — empty member lease after second-exit restart (Resolved by #193 / D-067, #213 & #216 revision 11)

Owner: **w2-agent-core**, coordinated with **w3-wire-nodes** and **w1-contracts**.
Locations: `node/agent/sync.go:needRenew`, `node/agent/decide.go:Decide`,
`internal/leases/leases.go:RenewTx` (`includeMembers`). AT: AT09.
Status: **Resolved** by PR #193 (D-067 provisional grant-less leases) and PR #213 / PR #216 (contract revision 11 `awaiting_member_grants` / `AWAITING_MEMBER_GRANTS`), verified on fresh `w4-members / 24000` run `80db103b-622c-42b1-946d-4628daaa213d`.

Reproduce on fresh `w4-members / 24000`: run the complete suite through paid
member admission and both published exits; select `node2` (`01a18eff-b044-7031-9a7c-5025329d8cff`)
and prove its counted transfer (`device_upload_bytes` `0 -> 2305`, `device_download_bytes` `0 -> 2411`
on desired/applied generation `23` with 7 member grants, `issued_at = 2026-10-30T22:59:25.120747+00:00`).
Stop `node2`, `node2-adguard` and `node2-netns` (`stopped_at = 2026-10-30T22:59:47.018508199Z`),
verify zero origin receipts while offline, then start the holder, resolver and
node in that order (`resumed_at = 2026-10-30T23:00:03.796010637Z`) with no policy
change or admin action (`desired_generation` remains `"23"`).

Under D-067 (#193) and revision 11 (#213, #216), a serving lease with zero member
grants while the desired generation has member bindings is issued with a 1-minute
provisional TTL and treated as provisional (`leaseAwaitingMemberGrants`):
`needRenew` renews at the next heartbeat and then with exponential backoff
(`1x`, `2x`, `4x` heartbeat, capped at 60 s) until a lease with grants arrives,
reporting `awaiting_member_grants` and `AWAITING_MEMBER_GRANTS` (never `lease_expired`)
while the provisional lease is in force.

Measured recovery on unchanged generation `23` ([AT09.json](AT09.json)):
- **Lease sequence (`serving_leases` for `node2` on generation `23`):**
  1. `2026-10-30T22:59:25.120747+00:00` — `generation: "23"`, `member_grants: 7` (pre-stop)
  2. `2026-10-30T23:00:04.349901+00:00` — `generation: "23"`, `member_grants: 0` (initial post-start provisional lease with 1-minute TTL while not yet `ready`)
  3. `2026-10-30T23:00:09.419186+00:00` — `generation: "23"`, `member_grants: 0` (D-067 provisional renewal #1 at ~5.1 s / 1x heartbeat)
  4. `2026-10-30T23:00:19.549439+00:00` — `generation: "23"`, `member_grants: 0` (D-067 provisional renewal #2 at ~10.1 s / 2x backoff)
  5. `2026-10-30T23:00:39.805556+00:00` — `generation: "23"`, `member_grants: 7` (D-067 provisional renewal #3 at ~20.3 s / 4x backoff after readiness)
- **Admin API & agent observation sequence on generation `23` (revision 11 verified):**
  - `2026-10-30T23:00:04.375940436Z`: `GET /api/v1/admin/nodes/{node2}` (`provisional_admin_node`) and `node_observations` report `member_forwarding_observed: false`, `block_reasons: ["awaiting_member_grants", "safety_check_failed"]` (`saw_lease_expired: false`), `lease_check: {"status": "failed", "error_code": "AWAITING_MEMBER_GRANTS", "detail": "Serving lease has no member grants yet."}`, `qos_check.error_code: "AWAITING_MEMBER_GRANTS"`, `source_binding_check.error_code: "AWAITING_MEMBER_GRANTS"`, `lease_stage_errors_count: 1`, `safety_status: "failed"`.
  - `2026-10-30T23:00:34.753993516Z`: `node_observations` reports `safety_status: "passed"`, `forwarding: false`, `reasons: ["awaiting_member_grants"]` (`saw_lease_expired: false`), `lease_check: {"status": "failed", "error_code": "AWAITING_MEMBER_GRANTS", "detail": "Serving lease has no member grants yet."}`, `qos_check.error_code: "AWAITING_MEMBER_GRANTS"`, `source_binding_check.error_code: "AWAITING_MEMBER_GRANTS"`, `lease_stage_errors_count: 1`, transitioning `node2` to `ready`.
  - `2026-10-30T23:00:39.865783068Z`: `GET /api/v1/admin/nodes/{node2}` (`recovered_admin_node`) and `node_observations` report `desired_generation: "23"`, `applied_generation: "23"`, `provisioning_state: "ready"`, `publication_intent: "published"`, `publication_observed: "active"`, `member_forwarding_observed: true`, `block_reasons: []`, `lease_check: {"status": "passed", "error_code": null, "detail": null}`.
- **Measured recovery time from container start (`2026-10-30T23:00:03.796010637Z`):**
  - Time to serving-ready (`serving_ready_at = 2026-10-30T23:00:40.771707223Z`, accepted observation + 18 kernel `member_accept_rules`): **`36.976 s`**.
  - Time to first counted member byte (`first_counted_byte_at = 2026-10-30T23:00:45.491347556Z`): **`41.695 s`**.
  - Member TCP attempts through `fvnb0` (`11 to_origin / 7 from_origin`), `node2` `fvmb0` (`11 to_origin / 7 from_origin`), `fvup0` (`11 to_origin / 7 from_origin`), and `origin eth0` (`11 to_origin / 7 from_origin`) succeed with `http_status: 200`, `origin_receipt: true`, `origin_snat_matches: true`, and fresh kernel counter increments (`device_upload_bytes: "0" -> "2305"`, `device_download_bytes: "0" -> "2461"`).

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

## Note for devstack follow-up 14 / worker owner: `fvpn-worker` container restart policy on transient `lock_timeout`

Owner: **w3-devstack-e2e** (`deploy/dev/compose.e2e.yaml`), coordinated with **w3-wire-nodes** (`apps/worker/jobs_nodes.go:RunNodeSchedule`).
Observation: During concurrent admin mutations (such as publishing DNS rules or feeds while `RunNodeSchedule` ticks), `RunNodeSchedule` can hit the 2-second Postgres `lock_timeout` on `SELECT id FROM fleet_config FOR UPDATE` (`internal/nodes/commandtx/transaction.go:36`) and return that error to `app.Run(ctx)`, causing `fvpn-worker` to exit (`STARTUP_OR_RUNTIME_FAILURE`) expecting a process supervisor (`systemd` in production) to restart it. Because `deploy/dev/compose.e2e.yaml` configures `RestartPolicy: no` on the `worker` service, an unhandled `lock_timeout` leaves `fvpn-worker` stopped until a test explicitly starts it. `tests/e2e/at/members/followup_test.go` (`feedLastGood`) supervises `fvpn-w4-members-worker` during its 5-minute scheduled feed refresh wait; setting `restart: on-failure` on `worker` in `deploy/dev/compose.e2e.yaml` (or treating `lock_timeout` in `RunNodeSchedule` as a retryable tick error) would eliminate this across all suites.

## Closed prior findings and preserved harness findings

- AT02 post-withdrawal ingress injection: resolved by PR #219 (`ObserveMemberIngress` in `tests/e2e/internal/labstack`). `AT02` now proves directly on `fvmb0` (`tun-mirred`, link type `65534`, source port `40000`) that frames from the revoked device `member-a2` increment `ctunku` by `+1` packet with `0` forwarded bytes (`uplink_empty: true`, `origin_empty: true`) while an admitted sibling's frame increments its `u_<hex>` by `+1` packet and reaches `fvup0` and `origin`.
- F4-01: resolved by PR #193 (D-067 provisional grant-less leases) and PR #213 / PR #216 (contract revision 11 `awaiting_member_grants` / `AWAITING_MEMBER_GRANTS`). Cold restart of `node2` (`node2`, `node2-adguard`, `node2-netns`) reports `awaiting_member_grants` and `AWAITING_MEMBER_GRANTS` (`saw_lease_expired: false`, `lease_stage_errors_count: 1`) during the provisional window and restores the 7-grant lease and counted member TCP traffic through `fvmb0` on unchanged desired generation `23` (`36.976 s` to serving-ready, `41.695 s` to first counted member byte).
  Follow-up 5's record remains archived under [history/follow-up-5](history/follow-up-5/AT09.json) and Follow-up 4's pre-D-067 record under [history/follow-up-4](history/follow-up-4/AT09.json).
- F3-02: current images contain conntrack. AT02 requires a continuous agent PID,
  coherent serving health, revoked-source denial/empty conntrack, strict `fvmb0` ingress injection proof, and counted
  sibling/replacement traffic. The historical failure remains archived.
- F3-01: selection waits for the advertised route and explicitly selects the
  network matching the intended node's peer. `skip_auto_apply` remains true.
- D-056/D-061: typed resolver evidence, canonical mTLS document hashes and actual
  block answers replace the former missing-delivery diagnosis. Null feed
  attribution is valid; only blank/whitespace notes are rejected.
