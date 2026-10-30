# Updates follow-up 4

Run: 9cb542dd-53af-4566-97e8-4365260847c5

Source: 136bd15d20359c64f2144cd3359f63ac28825988

These labels follow the orchestrator's follow-up scenario list. They do not redefine the broader §19 cases. Physical arm64 execution, multi-node populations, reinstall and stale-ACK races remain outside this run. Both architectures contain compiled real binaries.

| Scenario | Result | Observation |
|---|---|---|
| [AT23](AT23.md) | pass | Real 1.0.1 candidate answered D-053, locally_succeeded, then central succeeded and confirmed |
| [AT24](AT24.md) | pass | Product API showed unconfirmed before apply and needs_attention on the rejected candidate; no observation rows seeded |
| [AT25](AT25.md) | pass | A byte-tampered real archive produced HASH_MISMATCH before a transaction or executable was installed; last-good still running; follow-up 4 verified: failed_requires_cause |
| [AT26](AT26.md) | pass | Candidate tunnel_probe failed; healthy last-good restored; one terminal rollback received a current ACK; central rolled_back, needs_attention=true and unconfirmed=false; follow-up 4 verified: revision7_empty_target, revision7_rollback_target, conflict_bounds |
| [AT27](AT27.md) | pass | SIGKILL interrupted apply at final_sampled; production recover completed the same transaction with candidate D-053 health and central confirmation; follow-up 4 verified: observation_authority |

See [current requests](requests.md) and the [follow-up 2 AT26 history](history/follow-up-2/README.md).

Completed: 2026-10-30T06:20:24Z. Complete fresh-stack make target exit: **0** (Go test: 0).
See [machine-readable run metadata and digests](run.json) and [follow-up 4 probe details](follow-up-4.md).

AT26's revision 7 target fields are explicit null without a current receipt;
after rollback they exactly match the accepted result's candidate release,
restored checks and failed candidate checks. All five conflicting requests
returned 409 ACK_CONFLICT; both distinct hashes have one conflict row and one
audit, with the accepted receipt unchanged. AT27 preserves accepted release
authority through disagreeing and incomplete reports, then confirms agreement.
The missing-error failed result returns 422 VALIDATION_FAILED and leaves its
new job queued without attention.

The [initial run](history/follow-up-4-rate-limit/README.md) exited 2 because one
rapid conflict replay hit the API rate limiter. Its blocked AT26 capture is
preserved. The final run spaces the probes within existing limits and asserts
each response once. No product configuration or projector-owned row changed.
