# Follow-up 12 — transports and restore

AT30 **passed**; SECOPS-21 **open**. Application base `510a4ab7f73bdcfc28aaef53f8d1d6deef97ad67`. Restore run `dd845a7f-2660-431c-a8e6-c513108cc07d`; transport run `cf81f6f8-4ccf-43d2-aae2-10121f6b169d`. Shared development host measurements.

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

[AT29 established transport sub-cases](AT29.md) and [AT31 bounded relay](AT31.md) record individual results and remaining limits.

D-095 hold proved: `True`. First new stored offer `16`; retained node `15`. [HTTP probe, hold and generation history](SECOPS-21.json). D-099 roadmap eligibility: `True`. Compiler success alone does not close SECOPS-21.

[Merged follow-up 10 archive](history/follow-up-10/restore-reconcile.json) is byte-identical to PR #265; [follow-up 9](history/follow-up-9/restore-reconcile.json) remains at its D-098 handover claims path.
