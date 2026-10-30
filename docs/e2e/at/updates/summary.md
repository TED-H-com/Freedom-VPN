# Updates follow-up 3

Run: 339fcbc0-a3e0-4923-afc3-c2d6e12e9d1a

Source: b936aa71d57e03dd9263b9747e2b863839dbe4b7

These labels follow the orchestrator's follow-up scenario list. They do not redefine the broader §19 cases. Physical arm64 execution, multi-node populations, reinstall and stale-ACK races remain outside this run. Both architectures contain compiled real binaries.

| Scenario | Result | Observation |
|---|---|---|
| [AT23](AT23.md) | pass | Real 1.0.1 candidate answered D-053, locally_succeeded, then central succeeded and confirmed |
| [AT24](AT24.md) | pass | Product API showed unconfirmed before apply and needs_attention on the rejected candidate; no observation rows seeded |
| [AT25](AT25.md) | pass | A byte-tampered real archive produced HASH_MISMATCH before a transaction or executable was installed; last-good still running |
| [AT26](AT26.md) | pass | Candidate tunnel_probe failed; healthy last-good restored; one terminal rollback received a current ACK; central rolled_back, needs_attention=true and unconfirmed=false |
| [AT27](AT27.md) | pass | SIGKILL interrupted apply at final_sampled; production recover completed the same transaction with candidate D-053 health and central confirmation |

See [current requests](requests.md) and the [follow-up 2 AT26 history](history/follow-up-2/README.md).
