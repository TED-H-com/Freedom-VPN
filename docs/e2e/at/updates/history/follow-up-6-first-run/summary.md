# Updates follow-up 6

Run: e8a288c6-ffc3-41e8-ac7a-4c99cbc7e29f

Source: 1c8997d91a2c57a445fd25c62a0e4561fe5e6e8f

These labels follow the orchestrator's follow-up scenario list. They do not redefine the broader §19 cases. Physical arm64 execution, multi-node populations, reinstall and stale-ACK races remain outside this run. Both architectures contain compiled real binaries.

| Scenario | Result | Observation |
|---|---|---|
| [AT23](AT23.md) | blocked | Real 1.0.1 candidate answered D-053, locally_succeeded, then central succeeded and confirmed; follow-up 6 prerequisite/check incomplete: lab_probe_peer |
| [AT24](AT24.md) | blocked | Product API showed unconfirmed before apply and needs_attention on the rejected candidate; no observation rows seeded; follow-up 6 prerequisite/check incomplete: recorded_versions |
| [AT25](AT25.md) | blocked | A byte-tampered real archive produced HASH_MISMATCH before a transaction or executable was installed; last-good still running; follow-up 4 verified: failed_requires_cause; follow-up 6 prerequisite/check incomplete: lab_probe_peer |
| [AT26](AT26.md) | pass | Candidate tunnel_probe failed; healthy last-good restored; one terminal rollback received a current ACK; central rolled_back, needs_attention=true and unconfirmed=false; follow-up 4 verified: revision7_empty_target, revision7_rollback_target, conflict_bounds |
| [AT27](AT27.md) | blocked | SIGKILL interrupted apply at final_sampled; production recover completed the same transaction with candidate D-053 health and central confirmation; follow-up 4 verified: observation_authority; follow-up 6 prerequisite/check incomplete: recorded_audits |

See [current requests](requests.md) and the [follow-up 2 AT26 history](history/follow-up-2/README.md).
