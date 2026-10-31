# Requests to orchestrator — Follow-up 8

Measured application base `6b98422` (#207) includes members follow-up 5 (#205: discovery-denial audit deduplication in `internal/devices/audit.go:noteDiscoveryDecision`). No product, lab, or migration source outside owned paths was modified. Every claim is backed by [restore-reconcile.md](restore-reconcile.md) and [restore-reconcile.json](restore-reconcile.json).

| Request | Status / owner | Evidence and required action |
|---|---|---|
| SECOPS-05 / AT30 usage replay & provider reconciliation | **Usage replay & provider reconciliation passed; #205 discovery-denial deduplication verified** | `restore-reconcile.json` proves: (1) 2-minute fail-safe negative before `freeze`/`begin-recovery` (`T0..T1` archived events skipped while idle; post-`T1` event triggers `ErrAckRegress` and leaves `seq=6` in node `outbox` while central skips `3..5`); (2) 5-step `fvpn-usage-admin` protocol (`freeze` → `begin-recovery` → node `409 LEDGER_EPOCH_MISMATCH` + `GET /api/v1/agent/usage-checkpoint` + replay of `1..N1` with `5/5` canonical hash matches, `0` lost, `0` double-counted → `complete-recovery`); (3) provider reconciliation (`m1` converged to `canceled` + `refund` + `entitlement.state=expired` via `billing.reconcile`; `m2` active without extension); (4) #205 `identity_unmapped` audit emitted once (`count=1`) across multiple sweeps. |
| SECOPS-18, post-restore `T0..T1` device revocation preservation | **Open; `w2-members` (`internal/devices`)** | In `TestAT30_Followup8_LiveRestoreReconcile`, `peerM2D2` (`01a18ed5-a66c-709d-b5f0-a71fb1deefec`, peer `dbih7mf8k4c00083h1n0`) is revoked in `T0..T1` (`DELETE /api/v1/member/devices/{id}` → `202`, `device.revoke` audit, withdrawn from `fvpn-paid-<group_id>` in NetBird `N1`). After restoring central DB to `T0` (`member_devices.status='authorized'`, `device_authorization_observations.observed_authorized=true`), `members.reconcile` (`apps/worker/jobs_members.go:158` → `SyncMember` → `SyncDevice` in `internal/devices/worker.go:72-93`) treats `T0` `member_devices.status='authorized'` as authoritative and calls `s.Adapter.SetAuthorization(ctx, deployment, peer, true, groups)`, silently re-authorizing the revoked peer in NetBird (`l2_observed_authorized=True`) and keeping it in active `device_bindings` (`l2_in_active_device_bindings=True`). **Required action**: When central is in post-restore reconciliation (or when `device_authorization_observations.observed_authorized=true` in a restored DB conflicts with `ObservePeer` showing the peer is NOT in the expected NetBird paid group), fence `SyncDevice` so it does not silently re-authorize a peer whose external group membership was withdrawn after `T0` (either quarantine/revoke on observed external withdrawal when `observed_authorized` was already `true`, or persist a revocation marker/group in NetBird that `SyncDevice`/`Discover` respects), and emit an explicit audit event. |
| SECOPS-19, node retained-archive observability | **Informational / `w1-node-agent` (`node/agent/observe.go`)** | `node/agent/observe.go:111-116` currently sends a zeroed `UsageOutboxStatus{}` in `POST /api/v1/agent/observations`, and no node CLI/HTTP endpoint exposes `/var/lib/vpn-node/state/usage.db` (`state='archived'` rows retained for 72h). Populating `UsageOutboxStatus` (or exposing a read-only `fvpn-agent usage-status` subcommand) would let operators inspect retained archive sequences without opening the SQLite file directly. |
| SECOPS-17, finite grace provider truth (`past_due`) | **Open; `w3-devstack-e2e` payment hook** | On `#207`, `POST /e2e/snapshot` with `status: "past_due"` still returns HTTP `400` (`tests/e2e/cmd/fvpn-e2e-payment/truth.go:66` admits only `active`/`canceled`). AT30 used the supported `canceled` + `refund_minor=1000` transition to prove provider reconciliation; AT29 finite payment grace still awaits `past_due` support in `fvpn-e2e-payment`. |
| SECOPS-16, bulk final per-stream attribution | **Open; `w3-devstack-e2e`** | Unchanged from Follow-up 7 (`tests/e2e/native_exports.go:ScaleNativeMembers` aggregate return). |
| AT29 established transports & guest rollback; AT31 forced relay & hardware; DOC16 | **Open (`w3-devstack-e2e`, operator runs, `w4-handover`)** | Unchanged from Follow-up 7; see `history/follow-up-7/requests.md`. |

# Requests to orchestrator — Follow-up 9

Measured application base `205633f` (#231) includes wire-members follow-up 9 (#227: D-076 restore authorization fence), agent follow-up 19 (#226: SECOPS-19 live `UsageOutboxStatus` in observations), and usage-agent follow-up 1 (#225: read-only `fvpn-agent usage snapshot`). No product, lab, or migration source outside owned paths was modified. Every claim is backed by [restore-reconcile.md](restore-reconcile.md) and [restore-reconcile.json](restore-reconcile.json).

| Request | Status / owner | Evidence and required action |
|---|---|---|
| SECOPS-18 / D-076, post-restore `T0..T1` device revocation fence | **Closed (#227 verified in live AT30 restore)** | `restore-reconcile.json` proves that after restoring central DB to `T0`, `peerM2D2` (`01a18f9b-a2a0-728f-8482-b08cc9b1a3ee`, revoked in `T0..T1`) is fenced instead of re-granted (`device_authorization_fences` row `count=1`, worker log `"device authorization unconfirmed during ledger restore"` with 4 fields, worker counter `fvpn_device_restore_fence_decisions_total=2`, `device.authorization.external_withdrawal` audit `count=1`, `GET /api/v1/member/devices` `status=pending`, data-plane withdrawn). Both negatives pass (`peerM2D1` centrally revoked → `status=revoked`, `fence=0`; `peerM2D3` freshly enrolled after restore → `status=authorized`, `fence=0`). Re-grant via `AuthorizeDevice` (`member.device_authorize`, fresh idempotency key; admin HTTP route pending contract revision 12 / D-077) deletes the fence row and restores single NetBird grant and counted forwarding. |
| SECOPS-19, node unacked outbox & retained-archive observability | **Closed (#226 & #225 verified in live AT30 restore)** | During the restore hold (`reconciling`), `node_observations.report_payload.usage` reported `unacked_event_count=3`, `unacked_payload_bytes="2352"`, `ledger_epoch=01a18f9f-c20a-7716-9f31-8b078efba999`, `meter_epoch_id=01a18f9d-807e-78ee-8f36-aa1592b097e0`, matching `fvpn-agent usage snapshot -config /etc/vpn-node/agent-run.json -json` (`unacked_count=3`, `acked_through=1`, `last_event_seq=4`). After `complete-recovery` and replay, both converged to `0` unacked events and `"0"` bytes (`acked_through=4`, `last_event_seq=4`). |
| SECOPS-05 / AT30 full acceptance | **Closed / AT30 `passed`** | All AT30 pass criteria (isolated PITR + 6/6 checklist, 2-minute fail-safe negative, 5-step `fvpn-usage-admin` restore protocol, retained-node usage archive replay with `0` lost / `0` double-counted, provider reconciliation, D-076 peer/revocation fence + negatives + re-grant, SECOPS-19 visibility, and measured RPO/RTO) are now `passed` in [AT30.json](AT30.json). |
| D-077, admin HTTP route for device re-authorization | **Tracked (`contracts` revision 12 / `w3-wire-members`)** | `devices.Service.AuthorizeDevice` (`operation kind = member.device_authorize`) is verified live against the restored PostgreSQL and worker (`restore-reconcile.json`); the admin HTTP route wrapper remains tracked under contract revision 12 / D-077. |
| SECOPS-20, devstack `node-netns` Docker `ip nat POSTROUTING` (`DOCKER_POSTROUTING` `xt SNAT`) vs `#220` whole-ruleset `VerifyServing` | **New (`w3-devstack-e2e`: `deploy/devstack/entrypoint-node.sh` / `w1-enforce-nft`: `node/enforce/verify_foreign.go`)** | Because `deploy/devstack/compose.yaml` attaches two bridge networks (`lab` and `origin`) to `node-netns`, `dockerd` populates `table ip nat` with `POSTROUTING` → `DOCKER_POSTROUTING` (`xt target "SNAT"` for `127.0.0.11`). After `#220` (`2473acf`), `VerifyServing` reads `nft -j list ruleset` and fails closed with `FLOWTABLE_BYPASS` (`table nat family ip hook postrouting priority 100 chain POSTROUTING statement xt`) when `>=1` member grant is compiled. Observed and recorded on the unmodified container in `restore-reconcile.json` (`step1_checkpoint_t0.secops20_docker_postrouting_observation`) before removing the unused container-runtime `ip nat POSTROUTING` / `DOCKER_POSTROUTING` chains in `fvpn-w4-secops-node`. Request: `deploy/devstack/entrypoint-node.sh` should remove `ip nat POSTROUTING` / `DOCKER_POSTROUTING` on container boot (or `verify_foreign.go` should classify Docker's `127.0.0.11` embedded-DNS `DOCKER_POSTROUTING` jump). |
| SECOPS-21, post-restore `nodes.desired_config_generation` regression vs retained node `applied_config_generation` | **New (`w2-policy`: `internal/desiredstate/compiler.go:198-211` / `w3-wire-nodes`: `internal/nodes/observations.go:289`)** | When central PostgreSQL is restored to `T0` (`desired_config_generation=11`) while the node retains its `T1` state (`applied_config_generation=14`), the node's post-restore observation updates `nodes.applied_config_generation=14`. However, `desiredstate.Compiler.Compile` (`internal/desiredstate/compiler.go:198-211`) computes `baseGen = max(nodes.desired_config_generation, max(desired_states.generation))` without including `nodes.applied_config_generation`, so subsequent post-restore compilations emit stale generations (`12`, `13 < 14`) that `fvpn-agent` (`node/internal/agentstore/state.go:257`) rejects with `ErrStaleGeneration`. Request: include `nodes.applied_config_generation` in `baseGen` in `internal/desiredstate/compiler.go:198-211`. |
| AT29 / AT31 / SECOPS-16 / SECOPS-17 | **Deferred to Follow-up 10 (`w3-devstack-e2e` follow-up 14)** | Per Follow-up 9 instructions, AT29, AT31, and SECOPS-17 stay with Follow-up 10 awaiting devstack follow-up 14's exports. |

## Devstack follow-up 15 response (2026-10-31)

AT29: consume `StartEstablishedTransports`, `EstablishedStatus`, `DeviceByteCounters` and `StopEstablishedTransports`. TCP and authenticated member UDP use separate persistent sockets and independent senders; status includes attempts so silence cannot pass after the sender exits. The lease-expiry scenario keeps them active across control loss. `MemberTransportStack` selects an owned member/exit pair.

AT31: `ForceMemberRelay(ctx, memberContainer, true/false)` provides a bounded member underlay UDP fault, connected-path read-back and restoration. Its nft match expires after 180 seconds even if a caller is interrupted. The direct/relay scenario and current limitations are recorded under follow-up 15. This is a path fixture, not a completed 60-peer relay capacity campaign.

SECOPS-16 remains `NativeScale.Streams` from #204. SECOPS-17 remains the #240 fake-provider `past_due` snapshot with paid invoice preserved. SECOPS-20 (seen in the follow-up 9 request on origin/main) is already addressed by #240's node-entrypoint `ip nat` deletion, retained here under D-079. SECOPS-18/19/21 and D-077 are product/harness items; no lab workaround changes those contracts. Usage hold/release and parsed snapshot exports now support the ENR-05 availability-fault case separately from restore recovery.

# Requests to orchestrator — Follow-up 10

Dated 2026-10-31 UTC (2026-10-30 brief). Source includes #240/#241/#244/#245; owned harness and evidence paths only.

| Item | Status | Evidence / action |
|---|---|---|
| SECOPS-16 | **closed** | D-066 ordinary scan and final per-stream protected lifecycle, including the 60-peer bulk export. [SECOPS-16.json](SECOPS-16.json) |
| SECOPS-17 | **closed** | Fake provider past_due preserves the paid invoice; finite 90 s grace, API state, expiry and repeated truth measured. [SECOPS-17.json](SECOPS-17.json) |
| SECOPS-18 | **closed** | D-076 fence, log/counter/audit, withdrawn peer and two negatives re-observed. [SECOPS-18.json](SECOPS-18.json) |
| SECOPS-19 | **closed** | Nonzero hold outbox then zero after replay in observation and CLI. [SECOPS-19.json](SECOPS-19.json) |
| SECOPS-20 | **closed** | #240 lab NAT removal and #241 D-079 present; serving read-back and before history identify the observed cause. [SECOPS-20.json](SECOPS-20.json) |
| SECOPS-21 | **open** | #245 is present. Require first offer acceptance; if compilation precedes the first fresh retained-generation observation, the restored central applied value can still be stale. Review the timestamped first_offer_analysis and order restoration accordingly. Measured first offer 11 at 2026-10-31T04:07:21.367362+00:00; retained node 14; first central receipt of applied 14 at 2026-10-31T04:07:21.863973+00:00. [SECOPS-21.json](SECOPS-21.json) |
| SECOPS-05 / AT30 | **passed** | [Restore measurement table](restore-reconcile.md). |
| D-077 | **closed** | [HTTP call table](D-077.json); member session 401, revoked device 409 STATE_CONFLICT, same-key replay, existing operation. |
| AT29 established transports / AT31 forced relay | **Deferred** | Await devstack follow-up 15; no approximation or rerun. |

Prior request history is preserved byte-for-byte in [follow-up 9](history/follow-up-9/requests.md). No additional product/lab changes requested unless the recorded open findings require them.

# Requests to orchestrator — Follow-up 11 (2026-10-31 UTC; live execution pending)

D-098 is delivered in [history/follow-up-10/](history/follow-up-10/); the existing
follow-up 9 restore snapshot and handover claims test are unchanged.

| Item | Status / owner | Required clarification or export |
|---|---|---|
| Live AT29 / AT31 / AT30 | blocked, not run / orchestrator | The assigned w4-secops stack creates fvpn-w4-secops objects, but the brief allows only fvpn-w4-at-security-ops objects. Confirm the assigned existing namespace before Docker mutations. |
| Explicit manual grant withdrawal | blocked / contracts + w3-wire-members | Revision 13 exposes finite createMemberGrant but no revokeMemberGrant HTTP operation. Finite grant expiration is a separate prepared case. |
| AT29 guest clock rollback | blocked / w3-devstack-e2e + operator | Independently clocked disposable guest; containers share the host clock. |
| AT31 full mixed relay capacity | blocked / operator + acceptance lane | Candidate hardware and a 60-peer mixed-load campaign; the delivered bounded relay export supplies one path fixture. |
| SECOPS-21 | open — D-099 pending / w3-wire-nodes follow-up 29 | D-095 is merged, but no D-099 roadmap row exists on base c6e3eca. No new first-offer generations observed because the live run has not started. |

[Prepared commands, archive pins, all sub-case statuses and exact export limits](2026-10-31-follow-up-11-handover.md).

# Requests to orchestrator — Follow-up 12 (2026-10-31 UTC)

SECOPS-21: **open**, owner `w3-wire-nodes` follow-up 29 / D-099. See [hold and first-offer evidence](SECOPS-21.json); no closure from compiler tests.

The first HTTP probe and first stored row were `16` / `16`, versus retained `15` (first stored row above retained: `True`). The first subsequent applied observation was `17`; applied ACK for the first stored row: `False`.

Request to `w3-devstack-e2e` + `w3-wire-nodes`: export the actual first response delivered to the retained agent, its matching accepted local store, and applied ACK, so closure can be measured without inferring delivery from compiler rows or a separate probe. Keep SECOPS-21 open until all guards hold.

D-102: the dated devstack answers in `docs/e2e/RESPONSES.md` were read from later main `5ce76e9` once published. They document members/update exports; no first-delivered desired-state response capture is documented. The installer handover was read from `63a4116` once published; it reports ENR-06 socket/runtime fixed, with the required full upgrade still failing at final_sample because state/meter.checkpoint is absent. Both documents were absent on the assigned application base at task start. The node D-099 handover was read before planning. [Read revisions and hashes](history/follow-up-12/followup12-prerequisites.json). No external pin, packaged unit or measured application base changed.

D-103 is resolved by this task: only `fvpn-w4-secops-*`, network `fvpn-w4-secops`, port base 30000. The two-run guard refers to the prepared independent transport and restore fresh-stack runs, each torn down before the next.

The temporary lab scheduling request is resolved: the restore run started after the 09:04:09 UTC preflight admitted it. All earlier waits remain in [the preflight history](history/follow-up-12/followup12-lab-preflight.json).

| Sub-case / missing export | Status / owner | Limit |
|---|---|---|
| device_revoke | passed / w3-devstack-e2e | one original TCP/9446 connection and authenticated UDP/9445 socket; independent senders; helper limit 20 minutes |
| forced_relay | passed / w4-at-security-ops | one member, one exit, established TCP and UDP; 180-second automatic fault expiry, 90-second path wait; not 60-peer relay capacity |
| grant_expiry | passed / w3-devstack-e2e | one original TCP/9446 connection and authenticated UDP/9445 socket; independent senders; helper limit 20 minutes |
| guest_clock_rollback | blocked / w3-devstack-e2e / operator | Disposable guest with independently adjustable wall clock; containers share the development host clock. |
| lease_expiry | passed / w3-devstack-e2e | one original TCP/9446 connection and authenticated UDP/9445 socket; independent senders; helper limit 20 minutes |
| manual_grant_withdrawal | blocked / w3-wire-members / contracts | The contract exposes finite createMemberGrant but no revokeMemberGrant HTTP operation; no source-row mutation or invented endpoint used. Finite grant expiration is measured separately. |
| AT31 full capacity | blocked / operator + w4-at-security-ops | Needs chosen candidate hardware and 20-member/60-device mixed-load relay measurements; bounded single-member export is delivered. |

[Dated handover and archive pins](history/follow-up-12/2026-10-31-follow-up-12-handover.md).
