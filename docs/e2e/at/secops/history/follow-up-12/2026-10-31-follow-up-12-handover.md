# 2026-10-31 — Follow-up 12 handover

Follow-up 10 archive: `docs/e2e/at/secops/history/follow-up-10/`, exact PR #265 commit `2f1f939`, run `0e8ff9a4-4fa1-4154-ae35-847a5be4d985`. Pin `restore-reconcile.json` → `measurement_comparison.follow_up_10` there. Keep `history/follow-up-9/restore-reconcile.json` for handover follow-up 13.

AT30 regression: **passed**. SECOPS-21: **open**; first new stored offer `16`, retained `15`.

| AT29 / AT31 sub-case | Result |
|---|---|
| device_revoke | passed |
| forced_relay | passed |
| grant_expiry | passed |
| guest_clock_rollback | blocked |
| lease_expiry | passed |
| manual_grant_withdrawal | blocked |

| Sub-case / window | Independent sender attempts and replies |
|---|---|
| device_revoke / cutoff | TCP attempts 146 → 156, replies 80 → 80; UDP attempts 212 → 232, replies 77 → 77 |
| forced_relay / relay_samples | TCP attempts 10 → 26, replies 9 → 20; UDP attempts 9 → 32, replies 9 → 23 |
| forced_relay / direct_restored_samples | TCP attempts 68 → 78, replies 62 → 72; UDP attempts 74 → 84, replies 65 → 75 |
| grant_expiry / cutoff | TCP attempts 328 → 338, replies 325 → 325; UDP attempts 331 → 352, replies 324 → 324 |
| lease_expiry / cutoff | TCP attempts 161 → 171, replies 131 → 131; UDP attempts 191 → 211, replies 131 → 131 |

The transport helper lifetime remains 20 minutes, relay match 180 seconds and each relay-path wait 90 seconds. Missing counters are not zero-byte proof. Per-sample counter presence and monotonic timing remain in the JSON evidence.

[Earlier transport attempts](followup12-attempts.json) remain separately preserved with hashes: attempt 1 omitted publication, and attempt 2 measured grant cutoff but hit the recovery timestamp precision check. Neither is silently replaced by the final run or reported as a complete transport pass.

SECOPS-21: the separately labeled first HTTP **probe** returned generation `16`; the first new stored compiler row was `16`; the retained node was `15`. First-delivered agent response evidence is required for closure. The final local generation and central applied observations are recorded separately in [SECOPS-21.json](SECOPS-21.json).

The first subsequent applied observation was `17`. Applied ACK for the first stored row: `False`. First-delivered-response capture proved: `False`; first-offer local acceptance proved: `False`. The final snapshot shows local accepted/applied `20` / `20`, central applied `19`. These later values do not close missing first-offer guards. No generation repair was performed.

| Measurement | Archived follow-up 10 | Archived follow-up 11 (no new run) | Follow-up 12 |
|---|---|---|---|
| T0 / backup | 2026-10-31T04:03:16.470627Z / 20261031-040312F | 2026-10-31T04:03:16.470627Z / 20261031-040312F | 2026-10-31T09:06:59.403292Z / 20261031-090656F |
| L0 → L1 sequence | 2 → 4 | 2 → 4 | 2 → 5 |
| L0 → L1 bytes | 28139 → 79058 | 28139 → 79058 | 28103 → 79142 |
| Primary-overwrite refusal / checklist / PITR seconds | 2 / 6/6 / 9.749752081 | 2 / 6/6 / 9.749752081 | 2 / 6/6 / 5.264910803 |
| Negative restore window (s) / ACK / max seen | 120.143013897 / 2 / 5 | 120.143013897 / 2 / 5 | 120.071618829 / 2 / 6 |
| Pre-replay database RPO (s) | 95.507504 | 95.507504 | 106.450816 |
| Post-replay service usage RPO (s) | 0 | 0 | 0 |
| Protocol RTO (s) | 93.532554563 | 93.532554563 | 66.903926483 |
| Total outage RTO (s) | 106.868537734 | 106.868537734 | 74.531204784 |
| Replay events / hashes / lost / duplicate / gaps | 4 / 4 / 0 / 0 / 0 | 4 / 4 / 0 / 0 / 0 | 5 / 5 / 0 / 0 / 0 |
| Provider reconciliation | passed | passed | passed |
| SECOPS-19 hold unacked / bytes → replay unacked / bytes | 3 / 2352 → 0 / 0 | 3 / 2352 → 0 / 0 | 4 / 3198 → 0 / 0 |
| SECOPS-19 CLI hold unacked / bytes → replay unacked / bytes | 3 / 2352 → 0 / 0 | 3 / 2352 → 0 / 0 | 4 / 3198 → 0 / 0 |
| Central revoke / fresh post-restore enrollment negatives | True / True | True / True | True / True |
| D-076 fence / regrant | passed | passed | passed |

Cleanup: **passed**. Both run records include teardown exit status, state-directory removal and final container/network/volume/image counts. [Transport run](followup12-transports-run.json); [restore run](followup12-restore-run.json).

Both final inventories are zero for containers, networks, volumes and images, with generated state removed. [Current cleanup record](cleanup.json). The lab scheduling wait ended at 09:04:09 UTC; [preflight history](followup12-lab-preflight.json) records the required retries.

Follow-up 11 root records (105 files) are byte-identical to PR #274 under [history/follow-up-11/](history/follow-up-11/), with SHA-256 index entries. Its restore measurements are retained follow-up 10 values, not a new execution. No older snapshot was moved or rewritten.

Exact samples and export limits: [AT29](AT29.json), [AT31](AT31.json), [SECOPS-21](SECOPS-21.json). The existing handover test already reads the follow-up 9 archive; no further test repoint is needed.
