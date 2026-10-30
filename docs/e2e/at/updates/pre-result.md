# Pre-result observation authority — follow-up 6

Run `a83d5d64-505c-488c-b560-5a777566702e`, source `1c8997d91a2c57a445fd25c62a0e4561fe5e6e8f`, evidence `2026-10-30T17:10:45Z`. Prefix `w4-updates`, port base 42000.

[Full heartbeat payloads, responses, batch GETs, audit pages and real results](pre-result.json).
Each case has an initial heartbeat and three repetitions with fresh report IDs.
All return 202, with no new accepted result during injection.

| Incarnation / case | Current after every report | Last-good | New unexpected-release audits | Verdict |
|---|---|---|---|---|
| Fresh a: unrelated in both fields | null | null | 1 for the unrelated release | pass |
| Fresh b: target / other unrelated | null | null | 1 for the other release | pass under #167 last-good agreement rule |
| Fresh c: target / boot | null | null | 1 for boot | pass; F5-01 resolved |
| First genuine AT23 success | 1.0.1 | 1.0.1 | not a heartbeat claim | pass |
| Recorded c1: recorded current / recorded last-good | 1.0.1 | 1.0.1 | 0 | pass |
| Recorded c2: target 1.0.2 / recorded last-good | 1.0.1 | 1.0.1 | 0 | requested seed to 1.0.2 absent; unconfirmed=true |
| Recorded c3: unrelated 1.0.3 / recorded last-good | 1.0.1 | 1.0.1 | 0 | requested one audit absent; unconfirmed=true |
| Real updater recovery for the recorded job | 1.0.2 | 1.0.2 | accepted result establishes both | pass |

The recorded phase uses the higher-release recovery job after the tampered
attempt, with genuine prior accepted success and preflight-failure receipts.
The updater is killed at the actual service boundary and production recovery
completes the same authorized transaction. Its result is not fabricated.

Policy: `docs/updates/central-lifecycle.md:427-467` requires agreeing last-good
for first-result seeding, explicitly expects one fresh boot audit, and closes
seeding after the first accepted result even for later jobs. The requested
recorded c2/c3 assertions are retained separately; their mismatch is F6-02,
not a return of the resolved fresh-boot prerequisite. Accepted release authority
was not relaxed and no database observation columns were seeded.
