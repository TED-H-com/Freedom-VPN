# Requests to orchestrator — follow-up 5

Run `743c186d-5765-43e9-8b40-7839be178e13`, source `dd72ed4e43e2be21a852e6741d8caa06a8f6ad48`, at `2026-10-30T08:35:19Z`.
Fresh-stack make exit **2**; records were written before failure and teardown.
AT23/25/26 pass; AT24/27 are blocked by the recorded-boot prerequisite. The
original five update scenarios and follow-up 4 checks still pass.

## F5-01 — recorded-boot prerequisite absent

**Owners: w3-devstack-e2e and w2-fleet-update; fixture/expectation decision by the
orchestrator.** Provide a supported product path for the requested fresh
incarnation to have a recorded boot release before its first accepted result,
or clarify case (c)'s expectation when both central columns start null. Do not
seed the columns or relax observation authority to accommodate this harness.

Observed baseline: current_release_id=null, last_good_release_id=null,
accepted results=0. Twelve authenticated heartbeats all return 202. Case (a)
leaves both columns null/unconfirmed and creates one audit. Case (b) moves
current to the authorized job target, leaves last-good null, and creates one
audit for its distinct unrelated last-good. Repetitions do not add audits.
Case (c) also preserves last-good=null but creates exactly one additional audit
for boot, because boot was never centrally recorded. This is not evidence of
a fleet-update allowlist defect: `internal/fleetupdate/serving.go:133` permits
only the target/current/last-good; lines 143–147 audit unexpected claims.
`internal/nodes/observations.go:222` delegates version ownership to fleet and
clears its own write inputs. Jobless arrival cannot establish boot provenance
(`internal/fleetupdate/serving.go:218`). The source policy is described at
`docs/updates/central-lifecycle.md:425`.

The real AT23 updater subsequently produces one accepted success result,
establishing both current and last-good as 1.0.1. No success was forged for
this probe. [Raw requests, responses, batch GETs and admin audit pages](pre-result.json)
and [case table](pre-result.md) include all repetitions and the exact success
receipt. AT24/AT27 retain blocked status until the requested recorded-boot
fixture is genuinely exercised.

## F5-02 — no lab probe peer; default agent config path absent

**Owner: w3-devstack-e2e (follow-up 10 lab setup).** This run's main contains
follow-up 9 (#153), not follow-up 10. Both AT23 and AT25 read **no probe peer in
lab**. `/etc/vpn-node/updater-recover.json` and the actual updater invocation
configs have no probe_peer and no agent_config. `/etc/vpn-node/agent.json` does
not exist; the lab agent uses `/etc/vpn-node/agent-run.json`.

When adding the lab peer, supply the actual agent config path in updater config.
`tests/e2e/cmd/fvpn-e2e-nodeconfig/main.go:50` constructs the updater config and
line 63 writes the recovery config; `tests/e2e/cmd/fvpn-e2e-prepare/main.go:391`
writes agent-run.json. `docs/updates/node-updater.md:63` documents the default
path and `:65` the fixed `agent_config_unavailable` class. No such class was
emitted in this run: with probe_peer absent, these inputs are not validated.
The expected class for adding a peer without fixing that missing path is
`agent_config_unavailable`; it is a prospective finding, not a fabricated log.

[Configs/challenges/omission diagnostics](probe-issuance.json) and the
[probe finding](probe-issuance.md) preserve what was actually read. No file,
peer or health response was created to make issuance pass. Conditional probe
inspection is complete, but a configured D-063 issuance path remains untested.

## Tested main and resolved prior requests

[Source-main snapshot](source-main.json): fleet-update f7 #155, updater f10 #154,
devstack f9 #153, agent f10 #150, wire-nodes f15 #148, netlab f4 #144. Agent f11,
wire-nodes f16, netlab f5 and devstack f10 are absent from this checkout.

The #129 terminal rollback, #136 revision 7 target evidence and #140 accepted
result authority/receipt bounds remain verified. Exact current rollback result,
ACK and both batch GETs remain in AT26 and the standalone AT26 JSON files.
[Follow-up 4](history/follow-up-4/README.md) and earlier captures remain unchanged.
No product, contract or migration was edited. Physical arm64 and the broader
original §19 matrix remain outside this run; both architectures compile.
