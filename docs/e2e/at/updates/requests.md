# Requests to orchestrator — follow-up 6

Run `a83d5d64-505c-488c-b560-5a777566702e`, source `1c8997d91a2c57a445fd25c62a0e4561fe5e6e8f`, evidence `2026-10-30T17:10:45Z`. Prefix `w4-updates`, port base 42000.

Make exit **2**. AT26 passes. AT23/AT25 are blocked on the shared lab
probe prerequisite; AT24/AT27 on the requested recorded-incarnation assertions.
All original update scenarios and follow-up 4 checks still pass.

## F6-02 — recorded c2/c3 conflict with accepted-result authority

**Owner: w2-fleet-update; expectation decision by orchestrator.** Resolve the
requested recorded-incarnation c2 seeding and c3 one-audit expectations against
the documented incarnation-wide acceptance rule. Do not weaken authority to
make this harness pass. On the real recorded incarnation, c1 leaves current
and last-good at 1.0.1 with zero audits, as requested. Every c2 report retains
1.0.1 rather than seeding the job target 1.0.2; every c3 report retains 1.0.1
and produces zero unexpected-release audits. c2/c3 withhold confirmation.
Each has four HTTP 202 reports and no new accepted result. The later real
updater recovery succeeds and establishes both columns as 1.0.2.

`docs/updates/central-lifecycle.md:417-423,467` says accepted results own serving
versions and closes seeding even for later jobs. `internal/fleetupdate/serving.go:125`
guards both seeding and audit creation with `!accepted`; lines 142-156 write
current/audits only inside that branch. The implementation matches that policy,
but the requested c2/c3 assertions cannot pass under it. Preserve the explicit
assertions pending a decision. [Exact requests, responses, batch GETs, audit
pages, before/after and real accepted results](pre-result.json).

Fresh b/c also differ from the older f7 seeding behavior: fleet f8 (#167) requires
reported last-good to equal recorded last-good before any current write
(`serving.go:142`). With null baseline, b/c keep current=null. This is recorded
as the current policy's correct behavior, not as a fabricated target seed.

## F5-02 reopened — peer setup not exposed to the shared lab path

**Owner: w3-devstack-e2e.** Export the existing dedicated probe setup and
read-only exception observer through the shared lab API, including its generated
probe config, so this harness can reuse #170 without a second pipeline. Actual
AT23/AT25 invocation configs and `/etc/vpn-node/updater-recover.json` have no
probe_peer. Both set agent_config to the existing `/etc/vpn-node/agent-run.json`.
No challenge probe or refusal was emitted; D-063/D-064 issuance is unverified.

`tests/e2e/cmd/fvpn-e2e-nodeconfig/main.go:56-65` sets the correct path but no peer.
`tests/e2e/probe_scenario_test.go:45-134` creates the dedicated setup-key peer,
transport policy and narrow route in a private test-only runner method; only
`tests/e2e/scenarios_test.go:550` calls it. The public `Stack` methods in
`tests/e2e/stack_exports.go` and `PrepareServingBaseline` have no equivalent.
[Actual node configs, fresh challenges, attestations and stderr inspection](probe-issuance.json).
No peer or alias file was created by this harness. AT25's hash rejection occurs
before health, so it cannot issue a fresh challenge; clarify whether the probe
requirement should apply to a separate valid-candidate invocation for AT25.

## Resolved

- **F5-01:** resolved by the orchestrator's fresh-null expectation. Fresh case c
  produces exactly one boot-release/job audit despite repetitions. This main's
  f8 also retains both null columns because last-good disagrees. Policy citation:
  `docs/updates/central-lifecycle.md:458-465`. First genuine AT23 success then
  establishes current and last-good at 1.0.1; no boot provenance was seeded.
- **F5-02 path component:** #170 supplies the real, existing
  `/etc/vpn-node/agent-run.json`. F6-01 is not opened: no
  `agent_config_unavailable` refusal occurred. The peer integration portion
  remains open above.
- #129 direct terminal rollback, #136 revision 7 target evidence and #140
  accepted-result authority/receipt bounds remain verified. AT26 retains the
  exact accepted result, current ACK and raw batch JSON.

Tested main includes devstack f10 #170, updater f11 #168, fleet f8 #167,
agent f12 #166 (and f11 #162), wire-nodes f16 #160 and netlab f6 #159.
See [source snapshot](source-main.json) and [unchanged follow-up 5 history](history/follow-up-5/README.md).
No product, contract or migration changes. Physical arm64 and the full original
§19 matrix remain later operator work; both architectures compile.
