# Follow-up 10 — restore and security/operations evidence

AT30: **passed**. Run `0e8ff9a4-4fa1-4154-ae35-847a5be4d985`; application base `d9e28f8` includes #240, #241, #244 and #245. 2026-10-31T04:00:29Z → 2026-10-31T04:23:47Z. Shared development host, not candidate hardware.

| Measurement | Follow-up 9 (#242) | Follow-up 10 |
|---|---|---|
| T0 / backup | 2026-10-31T01:39:06.450534Z / 20261031-013901F | 2026-10-31T04:03:16.470627Z / 20261031-040312F |
| L0 → L1 sequence | 1 → 3 | 2 → 4 |
| L0 → L1 bytes | 28015 → 78833 | 28139 → 79058 |
| Primary-overwrite refusal / checklist / PITR seconds | 2 / 6/6 / 7.171869656 | 2 / 6/6 / 9.749752081 |
| Negative restore window (s) / ACK / max seen | 120.124449594 / 1 / 4 | 120.143013897 / 2 / 5 |
| Pre-replay database RPO (s) | 90.300808 | 95.507504 |
| Post-replay service usage RPO (s) | 0 | 0 |
| Protocol RTO (s) | 112.345194834 | 93.532554563 |
| Total outage RTO (s) | 121.87620611 | 106.868537734 |
| Replay events / hashes / lost / duplicate / gaps | 3 / 3 / 0 / 0 / 0 | 4 / 4 / 0 / 0 / 0 |
| Provider reconciliation | passed | passed |
| SECOPS-19 hold unacked / bytes → replay unacked / bytes | 3 / 2352 → 0 / 0 | 3 / 2352 → 0 / 0 |
| SECOPS-19 CLI hold unacked / bytes → replay unacked / bytes | 3 / 2352 → 0 / 0 | 3 / 2352 → 0 / 0 |
| Central revoke / fresh post-restore enrollment negatives | True / True | True / True |
| D-076 fence / regrant | passed | passed |

D-077 uses an admin session, CSRF, Origin, a fresh UUID key and no body. A still-authorized fenced device returns its existing `member.device_authorize` operation with the original desired generation. Completion is read from `GET /admin/devices` after worker sync.

| Request | HTTP / error | Operation / desired generation | Fence | Flag |
|---|---|---|---|---|
| GET member/devices before | 200  | — / — | 1 | True |
| GET admin/devices before | 200  | — / — | 1 | True |
| POST authorize with member cookie | 401 UNAUTHENTICATED | — / — | 1 | True |
| POST authorize admin CSRF Origin fresh key no body | 202  | 01a1901f-e211-722b-bd48-1a14d8876476 / 1 | 0 | — |
| POST authorize same key replay | 202  | 01a1901f-e211-722b-bd48-1a14d8876476 / 1 | 0 | — |
| GET admin/devices after worker sync | 200  | — / — | 0 | False |
| GET member/devices after worker sync | 200  | — / — | 0 | False |
| POST authorize revoked device | 409 STATE_CONFLICT | — / — | 0 | False |

SECOPS-16: **closed**; SECOPS-17: **closed**; SECOPS-18: **closed**; SECOPS-19: **closed**; SECOPS-20: **closed**; SECOPS-21: **open**.

SECOPS-20: `ip_nat_present=False`, `member_forwarding_enabled=True`, `loopback_scoped=0` entries. #240 deletes the lab NAT before agent startup and explains this serving transition. #241 is present; the empty loopback-scoped list does not exercise its exception. Follow-up 9’s before capture is unchanged in history.

SECOPS-21: compare all central offers and applied observations in [SECOPS-21.json](SECOPS-21.json). The agent silently ignores `ErrStaleGeneration`; zero matching log lines alone is insufficient. No harness generation adjustment is made.

First post-restore offer: `11` at `2026-10-31T04:07:21.367362+00:00`; retained node generation before central services resumed: `14` (node applied `14`, central T1 observation `12`). Below retained: `True`. When below retained, the remaining issue is startup ordering: compilation ran before fresh retained-generation observation. #245 can only use the applied generation already stored centrally.

AT29 established TCP/UDP and AT31 forced relay remain deferred to devstack follow-up 15. The bulk native run is only SECOPS-16 stream attribution. Physical arm64, candidate hardware and guest clock rollback remain operator work. Portal member/admin API state is measured for SECOPS-17; browser rendering is not exercised.

Structured measurements: [restore-reconcile.json](restore-reconcile.json), [D-077.json](D-077.json), [secret-scan.json](secret-scan.json), [SECOPS-17.json](SECOPS-17.json).
