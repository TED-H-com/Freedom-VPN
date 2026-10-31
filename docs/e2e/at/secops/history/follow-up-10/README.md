# Wave 4 security and operations acceptance

[Follow-up 10](summary.md) measures the local lab on application base `d9e28f8` (#240/#241/#244/#245), with live harness `319c5259344347e7eef50eb5becb2645785d1047`. Shared development host; no candidate-hardware or production SLA claim.

| Case | Full status | Scope |
|---|---|---|
| [AT30](AT30.json) | **passed** | Physical T0 restore, retained-node replay, provider/device reconciliation, D-077 HTTP re-grant and measured RPO/RTO. |
| [AT28](AT28.json) | blocked | D-066 scan measured; full ownership matrix beyond this run. |
| [AT29](AT29.json) | blocked | Finite past_due grace measured; established transports await devstack follow-up 15, guest clock rollback remains an operator run. |
| [AT31](AT31.json) | blocked | The bulk cohort proves final stream attribution; forced relay awaits devstack follow-up 15 and candidate hardware remains unmeasured. |
| [AT32](AT32.json) | blocked | Physical origins, arm64 and member platforms remain operator work. |

[SECOPS-16](SECOPS-16.json) applies D-066: ordinary logs, captures and readable log files must be clean; root-owned 0600 pre-delete login credentials are a separate class with final scan and deletion proof. Protected key-name counts alone are not leaks under this criterion.

[SECOPS-20](SECOPS-20.json) records the fresh serving gate and NAT read-back. [SECOPS-21](SECOPS-21.json) records the first post-restore generation, node-local retained state, observations and any remaining startup ordering issue. Prior failed observations are retained.

Use [RUN.md](RUN.md), [requests.md](requests.md), [validation.json](validation.json) and [scope-review.json](scope-review.json). The current target requires AT30, SECOPS-16/17/20 and cleanup; SECOPS-21 remains explicitly open when its first-offer proof fails. [Follow-up 9 history](history/follow-up-9/archive-index.json) is byte-identical; [diagnostic attempt](followup10-attempts.json) records the finalized-run preparation.
