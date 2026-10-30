# Updates follow-up 1

Run: 815b716a-b8da-40a6-aa9d-6446eab8da07

Source: d3009ea28352acb43e0dae5c95acd62f197ef619

These labels follow the orchestrator's follow-up scenario list. They do not redefine the broader §19 cases. Physical arm64 execution, multi-node populations, reinstall and stale-ACK races remain outside this run. Both architectures contain compiled real binaries.

| Scenario | Result | Observation |
|---|---|---|
| [AT23](AT23.md) | blocked | w2-agent-core / w3-wire-fleet: local D-053 success and central succeeded, but later agent observations do not confirm the update generation |
| [AT24](AT24.md) | blocked | w3-wire-fleet: unconfirmed/needs_attention projection not observed |
| [AT25](AT25.md) | blocked | w4-node-updater: tampered artifact rejection or central result incomplete |
| [AT26](AT26.md) | blocked | w4-node-updater / w3-wire-fleet: candidate failed and last-good rollback is recorded locally, but central rollback acceptance is incomplete; inspect journal and result outbox |
| [AT27](AT27.md) | blocked | w4-node-updater / w2-agent-core / w3-wire-fleet: see interrupted/recovered journal and observation evidence; complete recovery requires central confirmation |

The local paths were exercised successfully: 1.0.1 passed all five health
checks; byte-tampered 1.0.2 stopped at `HASH_MISMATCH` with no candidate
executable; recovery continued the same transaction from `final_sampled`
to `locally_succeeded`; the deliberately unhealthy 0.1.0 candidate rolled
back to healthy 1.0.2. These local facts do not clear central acceptance.

Central confirmation is blocked by null update fields in later agent
observations. Preflight failure delivery is rejected with HTTP 422 because
its payload has `health=null`; this leaves subsequent results pending and
prevents the expected needs-attention display. The earlier diagnostic run
also independently found the rollback health/allowlist mismatch. See
[requests and file references](follow-up-1.md#requests-to-orchestrator),
[final outbox evidence](result-delivery.md), and the
[separate rollback diagnostic](rollback-delivery-finding.md).

`make -f mk/at-updates.mk at-updates` wrote all records and returned exit 2
(the Go test returned 1). Its cleanup removed the owned stack. No physical
arm64 or complete original §19 acceptance is claimed.
