## Requirement and AT coverage

| Case | Requirements | November result | Evidence |
|---|---|---|---|
| AT14 | MTR01, MTR05, API01, DATA01 | pass | `docs/e2e/at/usage/AT14.json` / `.md` |
| AT15 | MTR02, MTR03, MTR04, API01, DATA01 | pass | `docs/e2e/at/usage/AT15.json` / `.md` |
| AT16 | MTR01, MTR02, MTR03, MTR04, UPD03 | pass | `docs/e2e/at/usage/AT16.json` / `.md` |
| AT17 | MTR01, MTR02, MTR04, API01, DATA01 | pass | `docs/e2e/at/usage/AT17.json` / `.md` |
| AT18 | FUP01, FUP02, FUP03 | pass | `docs/e2e/at/usage/AT18.json` / `.md` |
| AT19 | FUP01, FUP02, FUP03, FUP04, ENR03, API01 | pass | `docs/e2e/at/usage/AT19.json` / `.md` |
| AT20 | ENR03, MTR02, MTR03, MTR04, API01, DATA01 | pass | `docs/e2e/at/usage/AT20.json` / `.md` |

AT15 / §10.5–§10.8 and §14.5: each live record derives its period from its
own interval start, including offsets +4 s and +8 s. Binding-validity timing
checks are unchanged. Replay compares the rollup sum across both relevant
months, so month-spanning batches retain the no-double-count guarantee.

AT17 / §10.3, §10.5, §10.7, §19.5: the summary's returned period selects
the daily request and expected half-open month bounds. The summary bounds are
validated too. `2026-13` remains deliberately invalid and returns 422
VALIDATION_FAILED. Live November summary and daily totals are identical:
**12952 upload + 18151 download = 31103 bytes**, bounds
`[2026-11-01T00:00:00Z, 2026-12-01T00:00:00Z)`.

AT20 is the usage ENR-05 extension, not §19's DNS AT20. It selects the latest
held period with nonzero member traffic in both directions from the immutable
snapshot, and uses that same period for before/after/replay queries. Incarnation
checks include all held months. D-130 fresh traffic takes its own pre-traffic
current/next-month baselines while stopped and selects by the counted event's
interval start. November retained bytes: **33681 + 7736 = 41417**. Fresh counted
bytes: **1066 + 1514 = 2580**, exactly attributed once. Origin ACK 2 stays
independent from current ACK 1→2; stored ACK 1 and 2 replay checks pass.

Offline table proof covers month start, first seconds, offsets crossing a
month, mid-month, year end, year start and offsets crossing
2026-12-31 → 2027-01-01. It models a summary obtained before midnight and held
traffic snapshotted before a read-time rollover, checks the unchanged strict
period-overlap inequalities, and exercises a +08 display timezone. Negative
cases reject invalid summary bounds/months and empty/gap/other-member held
snapshots; multi-month selection and incarnation-wide sums are checked.
The fixed-clock October/November mechanism files are byte-unchanged.
