# Updates follow-up 5

Run: 743c186d-5765-43e9-8b40-7839be178e13

Source: dd72ed4e43e2be21a852e6741d8caa06a8f6ad48

These labels follow the orchestrator's follow-up scenario list. They do not redefine the broader §19 cases. Physical arm64 execution, multi-node populations, reinstall and stale-ACK races remain outside this run. Both architectures contain compiled real binaries.

| Scenario | Result | Observation |
|---|---|---|
| [AT23](AT23.md) | pass | Real 1.0.1 candidate answered D-053, locally_succeeded, then central succeeded and confirmed |
| [AT24](AT24.md) | blocked | Product API showed unconfirmed before apply and needs_attention on the rejected candidate; no observation rows seeded; follow-up 5 prerequisite/check incomplete: pre_result_versions |
| [AT25](AT25.md) | pass | A byte-tampered real archive produced HASH_MISMATCH before a transaction or executable was installed; last-good still running; follow-up 4 verified: failed_requires_cause |
| [AT26](AT26.md) | pass | Candidate tunnel_probe failed; healthy last-good restored; one terminal rollback received a current ACK; central rolled_back, needs_attention=true and unconfirmed=false; follow-up 4 verified: revision7_empty_target, revision7_rollback_target, conflict_bounds |
| [AT27](AT27.md) | blocked | SIGKILL interrupted apply at final_sampled; production recover completed the same transaction with candidate D-053 health and central confirmation; follow-up 4 verified: observation_authority; follow-up 5 prerequisite/check incomplete: pre_result_audits |

See [current requests](requests.md) and the [follow-up 2 AT26 history](history/follow-up-2/README.md).

Completed: 2026-10-30T08:35:19Z. Full fresh-stack make target exit: **2** (Go test: 1).
The five original update scenarios and all follow-up 4 checks passed. AT24 and
AT27 remain blocked by follow-up 5 case (c): the fresh incarnation had no
centrally recorded boot release before any result. Cases (a)/(b) preserve the
columns and produce exactly one audit per unexpected release; case (c) produces
one additional boot-release audit, which is consistent with the actual null
baseline but differs from the requested recorded-boot fixture. No assertion
was weakened and no projection row was seeded. The real AT23 success then
establishes last-good with exactly one accepted result.

Probe finding for AT23/AT25: **no probe peer in lab**. The default
`/etc/vpn-node/agent.json` is also absent; the actual lab agent uses
`/etc/vpn-node/agent-run.json`. No probe omission diagnostic was emitted because
no peer was configured. D-063 issuance is not claimed as verified.

See [run metadata and all four release digests](run.json), [tested main](source-main.json),
[pre-result requests, batch GETs and audit pages](pre-result.json),
[probe configs and diagnostics](probe-issuance.json), and [owner requests](requests.md).
The complete [follow-up 4 evidence](history/follow-up-4/README.md) is preserved.
