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
