# Requests to orchestrator — Follow-up 5

Measured main: `446b6b0` (#162), final run
`86e6106a-1be1-4003-9b0f-35a30e9cda12`, fresh `w4-secops` / `30000`.
See [dependencies](dependencies.json) for exact merged roadmap entries.

| ID / state | Cases | Owner / location | Evidence and next action |
|---|---|---|---|
| SECOPS-03 / resolved | AT29, AT31 | devstack #143/#153, nodes #160, agent #162 | Fresh ready/passed/healthy node, DNS and serving checks pass. Current selected-exit flow is counted. |
| SECOPS-04 / paid and grant subsets passed; remaining hooks blocked | AT29 | devstack `tests/e2e/cmd/fvpn-e2e-prepare/main.go:prepare`; `tests/e2e/stack_exports.go:PauseAgent`; node/fleet configuration owner | `deadlines.json`: real paid/grant source ends, control loss, SIGKILL agent, grant case SIGSTOP watchdog, allowset expiry, worker freeze, and valid-policy recovery. Remaining: prepared `grace_seconds=0` at main.go:243 has no documented finite-grace hook; `fleet_config.max_offline_lease_seconds` has no public setter or exported helper, so the node ceiling remains 86400. Provide sanctioned product inputs for these settings; never hand-write fleet or entitlement rows. Established TCP/UDP and guest rollback remain separate unrun rows. |
| SECOPS-05 / historical physical half passed; full reconciliation open | AT30 | backups and remaining acceptance integration | Original `pitr.json`/source preserved. Actual retained node SQLite replay and independently current payment, peer and revocation-intent reconciliation still need isolated restored-service integration. No new full-service RPO/RTO claim. |
| SECOPS-06 / resolved scoped login and classes | AT28, AT31 | control #106/#141/#145/#161 | 20/20 logins; exact burst refusals and matching metrics retained. Current body caps now agree. IPv6 /64 remains component verified, not a live IPv6-origin claim. |
| SECOPS-07 / prior DOC01–14 resolved; DOC15 open | AT32 | handover #138/#157, devstack docs | See DOC15 below. Physical platform statements remain correctly qualified. |
| SECOPS-08 / needs operator run | AT29–AT32 | operator | Real Windows/macOS/iOS/Android, physical arm64, second NAT, candidate hardware and disposable guest clock rollback remain in `operator-runs.md`. |
| SECOPS-09 / selected counted flow resolved | AT29, AT31 | devstack #153, agent #146/#162 | Explicit selection of the real published exit yields `fvnb0`, HTTPS 200, authenticated node-SNAT arrival and exact-device byte growth. Retire the missing/empty-member-lease serving blocker for this evidence. It is not the reason remaining cases are blocked. |
| SECOPS-10 / open on devstack follow-up 10 | full AT31 | `tests/e2e/native_exports.go:StartNativePeer,JoinNativeMember`; `deploy/devstack/nginx.conf` | Follow-up 10 was absent at both pre-run checks. Historical unchanged 20×3 attempt remains 50/60, limited by 256 gateway connections; no new 60-device success is claimed. This run drove two counted devices with 20 portal readers. After the budget fix merges, rerun the unchanged 20×3 profile. |
| SECOPS-11 / prior halves retained; current populated serving passed | AT29, AT31 | nodes #109/#148/#160 | Prior immutable negative and zero-binding positive remain in `followup4-publication.json`; this run adds real populated publication plus counted traffic, without forged observations. |
| SECOPS-12 / resolved historical PITR setup | AT30 | backups #111 and acceptance #112 | Original isolated restore image/target evidence retained; no current-schema physical rerun asserted. |
| SECOPS-13 / resolved | verification | nodes #130/#148, `internal/nodes/publication_evidence_test.go:17` | Exact requested owner test still passes against immutable owner testdata, independent of current live.json. |
| SECOPS-14 / resolved | AT28, AT31 | devstack #143 | Current 20 walkers land at GET / 200 and authenticate the expected distinct members. |
| SECOPS-15 / resolved by measured retest | AT28, AT31 | control #161; `docs/control/AUTH.md:Request body limits`, `internal/httpapi/middleware.go` | Valid usage sizes 8 MiB−1 and 8 MiB return 200; +1 returns 413 PAYLOAD_TOO_LARGE. All six other class triplets agree with the 1 MiB boundary. The observed 400 for whitespace on bodyless bootstrap is JSON validation, not oversize refusal. Real node-A heartbeat under node-B identity is refused 403; no telemetry row is forged. Duplicate retries leave one synthetic event row. |
| SECOPS-16 / remains open | AT28 | devstack `deploy/devstack/nginx.conf`; follow-up 10 | Follow-up 10 is absent from measured main, so the earlier exhaustion diagnostic and auth-query leak are not retired. Retained-stream known-canary scans are zero; that does not disprove the earlier `user_code` error-log finding or cover deleted transient native login logs. Add protected scan-before-delete/capture in `tests/e2e/native_exports.go:JoinNativeMember` for complete stream coverage. |
| F3-02 / measured coherence resolved | post-revoke | devstack #153, enforce #159, agent #162 | DELETE device returns 202; next observation applies generation 22, forwarding true, empty reasons. Second member has counted flow before and after; revoked device loses the exit. No package installation, process restart or policy workaround was used to repair revocation. |
| DOC15 / documentation change requested | AT32 | handover `docs/handover/member-guide.md:121,353`; devstack `docs/e2e/RUN.md:207` | These current-status paragraphs still name missing conntrack or lease refresh. #153 installs conntrack and this run measures counted forwarding and post-revoke coherence. Attribute prior blocked records historically and link current scoped evidence, while preserving grace/clock/platform/transport qualifications. |

The extra native hook does not inherit the Compose origin hostname mapping.
The first attempt records the actual DNS refusal. The final probe uses the
controlled origin IP already present in its generated certificate SAN, retaining
TLS verification; it changes no DNS, route, group or policy.

The two-device measurements do not justify a 60-device or candidate-hardware
capacity claim. Node Docker stats returned 0B/0B memory and zero CPU/IO in all
25 samples, so node resource usage is unavailable. Before the next capacity
run, augment `tests/e2e/at/secops/counted_capacity_test.go:countedCapacity` with
usable process/cgroup telemetry; no node memory limit can be inferred from
these zeros. Central-service snapshots and raw IO are preserved, without tuning
the test after seeing the results.
