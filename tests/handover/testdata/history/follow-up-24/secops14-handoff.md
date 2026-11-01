## Summary

Security-ops follow-up 14 (session launched 23:26 EDT on the #357 tip; 1 commit, 4 owned test files, merge-tree clean, prepush-scan clean; no live suite, Docker inventory 0). It fixes agent-core 30's findings 3 and 4 (#357's month-boundary sweep) in owned test code only.

## Summary

Dated **2026-10-31 EDT / 2026-11-01 UTC**. Commit `bac26ac8874d490171749795cd948af9fcfca788`, base `59e4e5597848b3fb1e20164414e3383307538fb9`.

Fixed agent-core 30 findings 3 and 4 entirely in owned test code. PITR's ten events and the rate-class event now call `usagePeriodForStart` with their own interval start. The shared helper returns that start's UTC month. Product ingestion's strict half-open overlap rule is unchanged.

One offline table test proves 44 one-second intervals: ten PITR events plus one rate-class event at each of four instants. The sweep also found and fixed an owned downstream single-month assumption in `restore_test.go`: replay now checks the expected bytes and quality for each retained period, then requires the same combined 3,000 bytes.

**No live suite ran and no Docker object was created.** Recorded live rows are unchanged: Follow-up 13 SECOPS-21 closed and AT30 passed; full AT29 / AT31 remain blocked. The next authorized live run uses the corrected helper. All requested gates passed. No dependencies or product code changed.

## Requirement and AT coverage

- MTR03 / DATA01, §10.4, §10.9, §17.3 and §19 AT30: synthetic source attribution and retained replay assertions respect UTC month boundaries. This is offline harness regression coverage, not a new live AT30 result.
- `TestMTR03_UsagePeriodOverlapsInterval`, build tag `e2e`, needs no stack, database, Docker or root. It uses `usage.ParseMonth` and the exact predicate `period.Start.Before(intervalEnd) && period.End.After(intervalStart)` from ingestion.
- Table instants: `2026-11-01T00:00:00Z`, `2026-11-01T00:00:05Z`, `2026-11-15T12:00:00Z`, and `2027-01-01T00:00:00Z`. The year case includes `[2026-12-31T23:59:59Z, 2027-01-01T00:00:00Z)`. The first-ten-seconds PITR case has five intervals in each month; exclusive-end boundary events require the preceding month. Mid-month is the normal control.

