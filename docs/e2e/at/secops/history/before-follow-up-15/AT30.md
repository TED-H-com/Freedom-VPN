# AT30 restore — Follow-up 13

AT30 **passed**; SECOPS-21 **closed**. Restore run `5218b7d5-1415-4e1b-8618-48ea2d3cf03e` on application base `7e044c71ecf1d07f63e89721bd213db6611a6a47`. Shared development host measurements; no production SLA.

Pre-replay database RPO (s): `92.163101` s versus `106.45` s; delta `-14.286899` s.
Total outage RTO (s): `81.016113709` s versus `74.53` s; delta `6.486113709` s.

| Measurement | Follow-up 12 | Follow-up 13 |
|---|---|---|
| T0 / backup | 2026-10-31T09:06:59.403292Z / 20261031-090656F | 2026-10-31T11:52:14.423299Z / 20261031-115210F |
| L0 → L1 sequence | 2 → 5 | 1 → 3 |
| L0 → L1 bytes | 28103 → 79142 | 28067 → 78746 |
| Primary-overwrite refusal / checklist / PITR seconds | 2 / 6/6 / 5.264910803 | 2 / 6/6 / 5.092580478 |
| Negative restore window (s) / ACK / max seen | 120.071618829 / 2 / 6 | 120.121679141 / 1 / 5 |
| Pre-replay database RPO (s) | 106.450816 | 92.163101 |
| Post-replay service usage RPO (s) | 0 | 0 |
| Protocol RTO (s) | 66.903926483 | 72.950138032 |
| Total outage RTO (s) | 74.531204784 | 81.016113709 |
| Replay events / hashes / lost / duplicate / gaps | 5 / 5 / 0 / 0 / 0 | 3 / 3 / 0 / 0 / 0 |
| Provider reconciliation | passed | passed |
| SECOPS-19 hold unacked / bytes → replay unacked / bytes | 4 / 3198 → 0 / 0 | 4 / 3006 → 0 / 0 |
| SECOPS-19 CLI hold unacked / bytes → replay unacked / bytes | 4 / 3198 → 0 / 0 | 4 / 3006 → 0 / 0 |
| Central revoke / fresh post-restore enrollment negatives | True / True | True / True |
| D-076 fence / regrant | passed | passed |

First delivered offer: HTTP `200`, generation `19` versus retained `15`, delivered `2026-10-31T11:56:09.869156137Z`; complete write `True`. Body SHA-256 `9db8da4cb30ef897e1eaf7f3494eaf8063c7823fcf7d3473d5250052ffc8b0f6`; local accepted/applied `19` / `19`, digest `9db8da4cb30ef897e1eaf7f3494eaf8063c7823fcf7d3473d5250052ffc8b0f6`. Applied ACK HTTP `202`, generation `19`, report `01a191d1-3799-7cee-ba57-53efe5f3dc18`, joined to the [central receipt and ordering evidence](SECOPS-21.json). Capture armed once; no retry or generation repair.

The first separate HTTP **probe** returned `19` at `2026-10-31T11:56:10.163037901Z`. The first new stored **compiler row** was `16`. Neither substitutes for the captured delivery. D-095 exact HTTP 503 / `pending fresh observation after restore` drift and unchanged generation, row count and envelope fingerprint: `True`.
