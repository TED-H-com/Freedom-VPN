# Follow-up 9 — AT30 re-measured with D-076 fence & SECOPS-19 visibility

Application base `205633f` (#231); harness `205633f0de8b3befbd563a092b856c0bc089e9c9`; run `42396951-2d9a-49f4-95c3-45f33d9519d5`. Shared development host, **not candidate hardware**.

| Measurement | Follow-up 8 (`#221`) | Follow-up 9 (`#231`) |
|---|---|---|
| `T0` pgBackRest full backup / restore target | `20261030-220312F` | `20261031-013901F` / `2026-10-31T01:39:06.450534Z` (LSN `0/7000DA8`) |
| `L0` → `L1` central ACKed usage sequence (`N0` → `N1`) | `2` → `5` | `1` (`28015` B) → `3` (`78833` B) |
| SECOPS-19 during restore hold (`node_observations` / `fvpn-agent usage snapshot`) | zeroed `UsageOutboxStatus` | `unacked_event_count=3` (`2352` B) / `unacked_count=3` (`2352` B), `ledger_epoch=01a18f9f-c20a-7716-9f31-8b078efba999` |
| SECOPS-19 after `complete-recovery` & replay | SQLite inspection only | Both converged to `0` unacked (`unacked_payload_bytes="0"`, `acked_through=4`, `last_event_seq=4`) |
| Usage replay (`L2` vs `L1` and `L0`) | **passed** (`5/5` hashes, `0` lost, `0` dup) | **passed** (`3/3` events, `3/3` hashes, `0` lost, `0` dup, `0` gaps) |
| Pre-replay DB RPO (`T1 - T0`) / Post-replay service usage RPO | `116.651 s` / `0 s` | `90.301 s` / `0 s` |
| Protocol RTO (Step 1 `freeze` → ready) / Total outage RTO | `105.94 s` / `111.51 s` | `112.35 s` / `121.88 s` |
| Provider reconciliation (`Central L2` vs `P1`) | **passed** | **passed** (`m1` `canceled` + `refund=1` + `expired`; `m2` `active`, `entitlement_extended=False`) |
| `T0..T1` revoked device (`peerM2D2`) after restore (`SECOPS-18` / D-076) | **blocked** (re-granted in NetBird) | **passed**: fenced (`device_authorization_fences=1`, worker log + counter `=1`, `external_withdrawal` audit `=1`, `GET /member/devices` `status=pending`, data-plane withdrawn) |
| D-076 Re-grant (`AuthorizeDevice`) & Two Negatives | N/A | **passed**: `T0` key replay kept fence (`1`); fresh `AuthorizeDevice` cleared fence (`0`) and restored flow; central revoke (`revoked`, `fence=0`) and fresh post-restore enroll (`authorized`, `fence=0`) unfenced |
| **AT30 overall status** | **blocked** (`SECOPS-18`) | **passed** |

Full details and command sequences are in [restore-reconcile.md](restore-reconcile.md) and [restore-reconcile.json](restore-reconcile.json). Follow-up 8 records are archived byte-for-byte under [history/follow-up-8](history/follow-up-8/archive-index.json). See [requests](requests.md) and [run instructions](RUN.md).
