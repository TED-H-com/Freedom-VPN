# Updates follow-up 2

Run: 9c862efb-3595-44e1-a6e9-b9a1f3f6cd67

Source: 5cb36d03d6e07e6e90a34d4ab2fe65511d63b313

These labels follow the orchestrator's follow-up scenario list. They do not redefine the broader §19 cases. Physical arm64 execution, multi-node populations, reinstall and stale-ACK races remain outside this run. Both architectures contain compiled real binaries.

| Scenario | Result | Observation |
|---|---|---|
| [AT23](AT23.md) | pass | Real 1.0.1 candidate answered D-053, locally_succeeded, then central succeeded and confirmed |
| [AT24](AT24.md) | pass | Product API showed unconfirmed before apply and needs_attention on the rejected candidate; no observation rows seeded |
| [AT25](AT25.md) | pass | A byte-tampered real archive produced HASH_MISMATCH before a transaction or executable was installed; last-good still running |
| [AT26](AT26.md) | blocked | w2-fleet-update: central accepted the healthy rollback result but the target remained queued; inspect the exact result and ACK in central_result_receipts |
| [AT27](AT27.md) | pass | SIGKILL interrupted apply at final_sampled; production recover completed the same transaction with candidate D-053 health and central confirmation |

See [requests and analysis](requests.md) for the remaining central transition blocker and exact result/ACK.
